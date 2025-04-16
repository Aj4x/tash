//go:build !windows

package main

import (
	"os"
	"syscall"
)

func StopTaskProcess(p *os.Process) error {
	return syscall.Kill(-p.Pid, syscall.SIGINT)
}

func TaskProcessAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		Setpgid: true,
	}
}
