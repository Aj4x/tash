//go:build windows

package task

import (
	"os"
	"syscall"
)

// StopTaskProcess stops a running task process by killing it
func StopTaskProcess(p *os.Process) error {
	return p.Kill()
}

// TaskProcessAttr returns the system process attributes for task execution on Windows
func TaskProcessAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}
}
