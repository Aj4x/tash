package task

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseTaskLine(t *testing.T) {
	// Inline sample tasks content
	tasksData := `task: Available tasks for this project:
* cowsay:                 Displays a cute ASCII art cow with a greeting message, mimicking the 'cowsay' program                           (aliases: cow, moo)
* date-time:              Shows the current date and time along with a calendar for the current month                                     (aliases: date, time, dt)
* default:                List all                                                                                                        (aliases: list, ls)
* generate-lorem:         Outputs a paragraph of Lorem Ipsum placeholder text that can be used for testing text display capabilities      (aliases: lorem, ipsum)
* random-quotes:          Shows a collection of famous programming and computer science quotes from well-known figures in the field       (aliases: quotes, q)
* weather:                Displays a simulated weather forecast for demonstration purposes                                                (aliases: wthr, w)
* cmd:dir:                
* cmd:ls:                 
* sys:disk-space:         Displays information about disk space usage on all mounted filesystems                                  (aliases: df, disk)
* sys:network-info:       Displays information about network interfaces and current network connections                           (aliases: netinfo, net)
* sys:process-list:       Displays a list of the top running processes on the system with details about CPU and memory usage      (aliases: ps, proc)
* sys:system-info:        Displays detailed information about the current system including OS, CPU, and memory                    (aliases: sysinfo, si)`

	// Define expected task structures for detailed verification
	expectedTasks := map[string]Task{
		"cowsay": {
			Id:      "cowsay",
			Desc:    "Displays a cute ASCII art cow with a greeting message, mimicking the 'cowsay' program",
			Aliases: []string{" cow", " moo"},
		},
		"date-time": {
			Id:      "date-time",
			Desc:    "Shows the current date and time along with a calendar for the current month",
			Aliases: []string{" date", " time", " dt"},
		},
		"default": {
			Id:      "default",
			Desc:    "List all",
			Aliases: []string{" list", " ls"},
		},
		"generate-lorem": {
			Id:      "generate-lorem",
			Desc:    "Outputs a paragraph of Lorem Ipsum placeholder text that can be used for testing text display capabilities",
			Aliases: []string{" lorem", " ipsum"},
		},
		"random-quotes": {
			Id:      "random-quotes",
			Desc:    "Shows a collection of famous programming and computer science quotes from well-known figures in the field",
			Aliases: []string{" quotes", " q"},
		},
		"weather": {
			Id:      "weather",
			Desc:    "Displays a simulated weather forecast for demonstration purposes",
			Aliases: []string{" wthr", " w"},
		},
		"cmd:dir": {
			Id:      "cmd:dir",
			Desc:    "",
			Aliases: nil,
		},
		"cmd:ls": {
			Id:      "cmd:ls",
			Desc:    "",
			Aliases: nil,
		},
		"sys:disk-space": {
			Id:      "sys:disk-space",
			Desc:    "Displays information about disk space usage on all mounted filesystems",
			Aliases: []string{" df", " disk"},
		},
		"sys:network-info": {
			Id:      "sys:network-info",
			Desc:    "Displays information about network interfaces and current network connections",
			Aliases: []string{" netinfo", " net"},
		},
		"sys:process-list": {
			Id:      "sys:process-list",
			Desc:    "Displays a list of the top running processes on the system with details about CPU and memory usage",
			Aliases: []string{" ps", " proc"},
		},
		"sys:system-info": {
			Id:      "sys:system-info",
			Desc:    "Displays detailed information about the current system including OS, CPU, and memory",
			Aliases: []string{" sysinfo", " si"},
		},
	}

	var parsedTasks []Task
	taskMap := make(map[string]Task)

	// Process each line
	for _, taskLine := range strings.Split(tasksData, "\n") {
		task, ok := ParseTaskLine(taskLine)
		if ok {
			parsedTasks = append(parsedTasks, task)
			taskMap[task.Id] = task
		}
	}

	// Test 1: Verify the total number of parsed tasks
	if len(parsedTasks) != 12 {
		t.Fatalf("Expected 12 tasks, got %d", len(parsedTasks))
	}

	// Test 2: Check all task types are correctly parsed
	const sysTask = "sys:disk-space"
	const cmdTask = "cmd:ls"
	const normalTask = "weather"
	foundSys, foundCmd, foundNormal := false, false, false

	for _, task := range parsedTasks {
		t.Logf("Task ID: '%s'", task.Id)
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

	// Test 3: Check that each task was parsed correctly with detailed validation
	for id, expectedTask := range expectedTasks {
		parsedTask, exists := taskMap[id]
		if !exists {
			t.Errorf("Expected task with ID '%s' was not found", id)
			continue
		}

		// Check ID
		if parsedTask.Id != expectedTask.Id {
			t.Errorf("Task '%s': expected ID '%s', got '%s'", id, expectedTask.Id, parsedTask.Id)
		}

		// Check description
		if parsedTask.Desc != expectedTask.Desc {
			t.Errorf("Task '%s': expected description '%s', got '%s'", id, expectedTask.Desc, parsedTask.Desc)
		}

		// Check aliases
		if !reflect.DeepEqual(parsedTask.Aliases, expectedTask.Aliases) {
			t.Errorf("Task '%s': expected aliases %v, got %v", id, expectedTask.Aliases, parsedTask.Aliases)
		}
	}

	// Test 4: Additional specific cases
	// Test tasks with empty descriptions
	emptyDescTasks := []string{"cmd:dir", "cmd:ls"}
	for _, id := range emptyDescTasks {
		task, exists := taskMap[id]
		if !exists {
			t.Errorf("Empty description task '%s' not found", id)
			continue
		}
		if task.Desc != "" {
			t.Errorf("Task '%s' should have empty description, got '%s'", id, task.Desc)
		}
	}

	// Test tasks with namespaces (sys: and cmd:)
	namespacedTasks := []string{"sys:disk-space", "sys:network-info", "sys:process-list", "sys:system-info", "cmd:dir", "cmd:ls"}
	for _, id := range namespacedTasks {
		if _, exists := taskMap[id]; !exists {
			t.Errorf("Namespaced task '%s' not found", id)
		}
	}

	// Test 5: Test the header line doesn't parse as a task
	headerLine := "task: Available tasks for this project:"
	_, ok := ParseTaskLine(headerLine)
	if ok {
		t.Error("Header line should not be parsed as a task")
	}

	// Test 6: Test for specific tasks with specific attributes
	// Check a task with multiple aliases
	dateTimeTask, exists := taskMap["date-time"]
	if !exists {
		t.Fatal("Expected to find 'date-time' task")
	}
	if len(dateTimeTask.Aliases) != 3 {
		t.Errorf("Expected 'date-time' task to have 3 aliases, got %d", len(dateTimeTask.Aliases))
	}

	// Check a task with single alias
	weatherTask, exists := taskMap["weather"]
	if !exists {
		t.Fatal("Expected to find 'weather' task")
	}
	if len(weatherTask.Aliases) != 2 {
		t.Errorf("Expected 'weather' task to have 2 aliases, got %d", len(weatherTask.Aliases))
	}
}
