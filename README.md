# ovscni
kubernetes network  ovs plugin  demo

### 学习ovs用

### 环境
```shell script
    centos7       Linux k8s-master 5.4.131-1.el7.elrepo.x86_64 #1 SMP Sun Jul 11 08:52:19 EDT 2021 x86_64 x86_64 x86_64 GNU/Linux
    kubernetes    version v1.23.1
    go            1.16 go version go1.16.10 linux/amd64
    openvswitch   ovs_version: "2.12.0"
    vmware 15
```
### 安装基础环境

```shell script

    添加第二张网卡
    ens37   nat模式
    
    
    添加阿里云 yum 源
    wget -O /etc/yum.repos.d/CentOS-Base.repo https://mirrors.aliyun.com/repo/Centos-7.repo
    sed -i -e '/mirrors.cloud.aliyuncs.com/d' -e '/mirrors.aliyuncs.com/d' /etc/yum.repos.d/CentOS-Base.repo
    
    
    关闭 selinux
    vim /etc/selinux/config
    
    重启 reboot
    
    关闭防火墙
    systemctl stop firewalld
    systemctl disable firewalld
    
    安装 ovs
    
    yum install -y centos-release-openstack-train.noarch
    yum install -y openvswitch openvswitch-devel openvswitch-ipsec openvswitch-test
    
    #网桥工具
    yum -y install bridge-utils
    
    这个问题 下载下面那个包
    #net_mlx5: cannot load glue library: libibverbs.so.1: cannot open shared object file: No such file or directory
    #net_mlx5: cannot initialize PMD due to missing run-time dependency on rdma-core libraries (libibverbs, libmlx5)
    #PMD: net_mlx4: cannot load glue library: libibverbs.so.1: cannot open shared object file: No such file or directory
    #PMD: net_mlx4: cannot initialize PMD due to missing run-time dependency on rdma-core libraries (libibverbs, libmlx4)
    
    yum install libibverbs -y
    
    systemctl enable openvswitch
    systemctl start openvswitch

    
    开启 交换机的 stp 生成树协议  防止二层环路
    
    ovs-vsctl get Bridge ovs-br0 stp_enable
    ovs-vsctl set Bridge ovs-br0 stp_enable=true
    
    安装 docker
    yum install -y docker
    systemctl enable docker
    systemctl start docker
    
    kubeadm  安装 k8s
    
```


### 安装 ovscni  网络插件
````shell script
    
    复制 ovscni 到  /opt/cni/bin/ovscni
    
    修改 10-ovscni.conf 中的 cidr 和虚拟机一个网段
    复制 10-ovscni.conf 到  /etc/cni/net.d/10-ovscni.conf
   
````

### 检查
```shell script

    查看节点 状态
    kubectl  get node
    运行一个pod 查看 创建 pod是否正常
    kubectl  run nginx --image=nginx 
```