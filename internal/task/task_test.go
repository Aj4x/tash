package task

import (
	"os"
	"strings"
	"testing"
)

func TestParseTaskLine(t *testing.T) {
	tasksData, err := os.ReadFile("testdata/sampletasks.txt")
	if err != nil {
		t.Fatal(err)
	}
	tasks := string(tasksData)
	var parsedTasks []Task
	for _, taskLine := range strings.Split(tasks, "\n") {
		task, ok := ParseTaskLine(taskLine)
		if ok {
			parsedTasks = append(parsedTasks, task)
		}
	}
	if len(parsedTasks) != 12 {
		t.Fatalf("Expected 12 tasks, got %d", len(parsedTasks))
	}
	const sysTask = "sys:disk-space"
	const cmdTask = "cmd:ls"
	const normalTask = "weather"
	foundSys, foundCmd, foundNormal := false, false, false
	for _, task := range parsedTasks {
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
