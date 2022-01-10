package main

import (
	"errors"
	"os"
)

func log(log_path, string2 string) error {
	if log_path == "" {
		return errors.New("path is none")
	}
	file, err := os.OpenFile(log_path, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0755)
	defer file.Close()
	if err != nil {
		return err
	}
	_, err = file.WriteString(string2 + "\n")
	return err
}

func addlog(log_path, string2 string) error {
	return log(log_path, "add:    "+string2)
}

func dellog(log_path, string2 string) error {
	return log(log_path, "del:    "+string2)
}
