package main

import (
	"os"
)

func log(string2 string) error {
	file, err := os.OpenFile("/root/log.log", os.O_APPEND|os.O_RDWR, 0755)
	defer file.Close()
	if err != nil {
		return err
	}
	_, err = file.WriteString(string2 + "\n")
	return err
}

func addlog(string2 string) error {
	return log("add:    " + string2)
}

func dellog(string2 string) error {
	return log("del:    " + string2)
}
