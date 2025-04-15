package main

import (
	"os"
	"strings"
	"testing"
)

func TestModel_AppendTask(t *testing.T) {
	tasksData, err := os.ReadFile("sampletasks.txt")
	if err != nil {
		t.Fatal(err)
	}
	tasks := string(tasksData)
	m := NewModel()
	for _, task := range strings.Split(tasks, "\n") {
		m.appendTask(task)
	}
	if len(m.Tasks) != 12 {
		t.Fatalf("Expected 12 tasks, got %d", len(m.Tasks))
	}
	const sysTask = "sys:disk-space"
	const cmdTask = "cmd:ls"
	const normalTask = "weather"
	foundSys, foundCmd, foundNormal := false, false, false
	for _, task := range m.Tasks {
		t.Log("'" + task.Id + "'")
		if task.Id == sysTask {
			foundSys = true
		}
		if task.Id == cmdTask {
			foundCmd = true
		}
		if task.Id == normalTask {
			foundNormal = true
		}
	}
	if !foundSys {
		t.Fatalf("Expected to find task '%s'", sysTask)
	}
	if !foundCmd {
		t.Fatalf("Expected to find task '%s'", cmdTask)
	}
	if !foundNormal {
		t.Fatalf("Expected to find task '%s'", normalTask)
	}
}
