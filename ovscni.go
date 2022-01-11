// Copyright 2017 CNI authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This is a sample chained plugin that supports multiple CNI versions. It
// parses prevResult according to the cniVersion
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types"
	current "github.com/containernetworking/cni/pkg/types/100"
	"github.com/containernetworking/cni/pkg/version"
	"github.com/containernetworking/plugins/pkg/ip"
	"github.com/containernetworking/plugins/pkg/ipam"
	"github.com/containernetworking/plugins/pkg/ns"
	bv "github.com/containernetworking/plugins/pkg/utils/buildversion"
	"github.com/vishvananda/netlink"
	"net"
	"os/exec"
	"strings"
	"time"
)

// PluginConf is whatever you expect your configuration json to be. This is whatever
// is passed in on stdin. Your plugin may wish to expose its functionality via
// runtime args, see CONVENTIONS.md in the CNI spec.
type PluginConf struct {
	// This embeds the standard NetConf structure which allows your plugin
	// to more easily parse standard fields like Name, Type, CNIVersion,
	// and PrevResult.
	types.NetConf
	LogPath       string `json:"log_path"`
	RuntimeConfig *struct {
		SampleConfig map[string]interface{} `json:"sample"`
	} `json:"runtimeConfig"`
	Bridge string `json:"bridge"`
	// Add plugin-specifc flags here
}

// parseConfig parses the supplied configuration (and prevResult) from stdin.
func parseConfig(stdin []byte) (*PluginConf, error) {
	conf := PluginConf{}

	if err := json.Unmarshal(stdin, &conf); err != nil {
		return nil, fmt.Errorf("failed to parse network configuration: %v", err)
	}

	// Parse previous result. This will parse, validate, and place the
	// previous result object into conf.PrevResult. If you need to modify
	// or inspect the PrevResult you will need to convert it to a concrete
	// versioned Result struct.
	if err := version.ParsePrevResult(&conf.NetConf); err != nil {
		return nil, fmt.Errorf("could not parse prevResult: %v", err)
	}
	// End previous result parsing

	return &conf, nil
}

func parseValueFromArgs(key, argString string) (string, error) {
	if argString == "" {
		return "", errors.New("CNI_ARGS is required")
	}
	args := strings.Split(argString, ";")
	for _, arg := range args {
		if strings.HasPrefix(arg, fmt.Sprintf("%s=", key)) {
			value := strings.TrimPrefix(arg, fmt.Sprintf("%s=", key))
			if len(value) > 0 {
				return value, nil
			}
		}
	}
	return "", fmt.Errorf("%s is required in CNI_ARGS", key)
}

// cmdAdd is called for ADD requests
func cmdAdd(args *skel.CmdArgs) error {
	//debug
	time.Sleep(2 * time.Second)

	conf, err := parseConfig(args.StdinData)
	if err != nil {
		_ = addlog(conf.LogPath, fmt.Sprintf("parseConfig：%s", err.Error()))
		return err
	}
	ipamres, err := ipam.ExecAdd(conf.IPAM.Type, args.StdinData)
	if err != nil {
		_ = addlog(conf.LogPath, fmt.Sprintf("ipam出错：%s", err.Error()))
		return err
	}
	result, err := current.GetResult(ipamres)
	if err != nil {
		_ = addlog(conf.LogPath, fmt.Sprintf("GetResult：%s", err.Error()))
		return err
	}

	netns, err := ns.GetNS(args.Netns)
	if err != nil {
		_ = addlog(conf.LogPath, fmt.Sprintf("GetNS：%s", err.Error()))
		return fmt.Errorf("failed to open netns %q: %v", args.Netns, err)
	}
	defer netns.Close()
	hostIface, containerinter, err := setupVeth(netns, conf.Bridge, args.ContainerID[:8], args.IfName, 1500, "")
	if err != nil {
		_ = addlog(conf.LogPath, fmt.Sprintf("setupVeth err：%s", err.Error()))
		return err
	}
	// 设置 ovs端口
	_, err = exec.Command("ovs-vsctl", "add-port", conf.Bridge, hostIface.Name).CombinedOutput()
	if err != nil {
		_ = addlog(conf.LogPath, fmt.Sprintf("向ovs添加port失败,%s", err.Error()))
		return err
	}
	if len(result.IPs) == 0 {
		return errors.New("not ip")
	}
	podName, err := parseValueFromArgs("K8S_POD_NAME", args.Args)
	if err != nil {
		podName = "none"
		_ = addlog(conf.LogPath, fmt.Sprintf("podname: %s, ip：%s", podName, result.IPs[0].Address.IP.String()))
	} else {
		_ = addlog(conf.LogPath, fmt.Sprintf("podname: %s, ip：%s", podName, result.IPs[0].Address.IP.String()))
	}

	if err := netns.Do(func(_ ns.NetNS) error {
		contVeth, err := net.InterfaceByName(args.IfName)
		if err != nil {
			_ = addlog(conf.LogPath, fmt.Sprintf("setupVeth err：%s", err.Error()))
			return err
		}
		if podName == "" {
			podName = "node"
		}
		link, err := netlink.LinkByName(args.IfName)
		for _, ipc := range result.IPs {
			_ = addlog(conf.LogPath, fmt.Sprintf("pod : %s 设置 ip", podName))
			if ipc.Address.IP.To4() != nil {
				_ = addlog(conf.LogPath, fmt.Sprintf(" inter : %s, 设置ip:%s, info: %s", contVeth.Name, ipc.Address.IP.To4().String(), podName))
				addr := &netlink.Addr{IPNet: &ipc.Address, Label: ""}
				if err = netlink.AddrAdd(link, addr); err != nil {
					_ = addlog(conf.LogPath, fmt.Sprintf("设置ip出错:%s", err.Error()))
					return fmt.Errorf("failed to add IP addr %v to %q: %v", ipc, contVeth.Name, err)
				}
			}
		}
		return nil
	}); err != nil {
		_ = addlog(conf.LogPath, fmt.Sprintf("setupVeth err：%s", err.Error()))
		return err
	}
	result.DNS = conf.DNS
	result.Interfaces = []*current.Interface{
		hostIface,
		containerinter,
	}

	data, err := json.MarshalIndent(result, "", "    ")
	if err != nil {
		_ = addlog(conf.LogPath, err.Error())
	} else {
		_ = addlog(conf.LogPath, "out: "+string(data))
	}
	return types.PrintResult(result, conf.CNIVersion)
}

func setupVeth(netns ns.NetNS, bridge_name, hostVethName, ifName string, mtu int, mac string) (*current.Interface, *current.Interface, error) {
	contIface := &current.Interface{}
	hostIface := &current.Interface{}

	err := netns.Do(func(hostNS ns.NetNS) error {
		// create the veth pair in the container and move host end into host netns
		hostVeth, containerVeth, err := ip.SetupVethWithName(ifName, hostVethName, mtu, mac, hostNS)
		if err != nil {
			return err
		}
		contIface.Name = containerVeth.Name
		contIface.Mac = containerVeth.HardwareAddr.String()
		contIface.Sandbox = netns.Path()
		hostIface.Name = hostVeth.Name
		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	// need to lookup hostVeth again as its index has changed during ns move
	hostVeth, err := netlink.LinkByName(hostIface.Name)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to lookup %q: %v", hostIface.Name, err)
	}
	hostIface.Mac = hostVeth.Attrs().HardwareAddr.String()

	return hostIface, contIface, nil
}

// cmdDel is called for DELETE requests
func cmdDel(args *skel.CmdArgs) error {
	time.Sleep(2 * time.Second)
	conf, err := parseConfig(args.StdinData)
	if err != nil {
		return err
	}
	_ = dellog(conf.LogPath, fmt.Sprintf("删除:container: %s, Netns:%s, ifname: %s, stdindata: %s, args: %s", args.ContainerID, args.Netns, args.IfName, args.StdinData, args.Args))

	err = ipam.ExecDel(conf.IPAM.Type, args.StdinData)
	if err != nil {
		_ = dellog(conf.LogPath, fmt.Sprintf("ExecDel：%s", err.Error()))
		return err
	}
	if args.Netns == "" {
		return nil
	}
	err = ns.WithNetNSPath(args.Netns, func(_ ns.NetNS) error {
		var err error
		_, err = ip.DelLinkByNameAddr(args.IfName)
		if err != nil && err == ip.ErrLinkNotFound {
			return nil
		}
		return err
	})
	if err != nil {
		_ = dellog(conf.LogPath, fmt.Sprintf("WithNetNSPath：%s, pod: %s, nsfile: %s", err.Error(), args.Args, args.Netns))
	}
	_, err = exec.Command("ovs-vsctl", "del-port", conf.Bridge, args.ContainerID[:8]).CombinedOutput()

	if err != nil {
		_ = dellog(conf.LogPath, fmt.Sprintf("Command：%s", err.Error()))
		return err
	}

	return nil
}

func main() {
	skel.PluginMain(cmdAdd, cmdCheck, cmdDel, version.All, bv.BuildString("ovscni"))
}

func cmdCheck(args *skel.CmdArgs) error {
	// TODO: implement
	return fmt.Errorf("not implemented")
}
