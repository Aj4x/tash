//go:build !windows

package task

import (
	"os"
	"syscall"
)

// StopTaskProcess stops a running task process by sending a SIGINT signal
func StopTaskProcess(p *os.Process) error {
	return syscall.Kill(-p.Pid, syscall.SIGINT)
}

// TaskProcessAttr returns the system process attributes for task execution
func TaskProcessAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		Setpgid: true,
	}
}
