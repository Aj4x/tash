//go:build windows

package main

import (
	"os"
	"syscall"
)

func StopTaskProcess(p *os.Process) error {
	return p.Kill()
}

func TaskProcessAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}
}
