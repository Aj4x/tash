package ui

import (
	"encoding/json"
	"testing"

	"github.com/Aj4x/tash/internal/task"
)

func TestParseTasksJsonWithExtendedInput(t *testing.T) {
	inputJson := `
	{
		"tasks": [
			{
				"name": "cowsay",
				"desc": "Displays a cute ASCII art cow with a greeting message, mimicking the 'cowsay' program",
				"summary": "ASCII cow art",
				"aliases": ["cow", "moo"],
				"up_to_date": false,
				"location": {
					"line": 115,
					"column": 3,
					"taskfile": "/home/Aj4x/go/src/github.com/Aj4x/tash/examples/Taskfile.yml"
				}
			},
			{
				"name": "date-time",
				"desc": "Shows the current date and time along with a calendar for the current month",
				"summary": "Display current date and time",
				"aliases": ["date", "time", "dt"],
				"up_to_date": false,
				"location": {
					"line": 47,
					"column": 3,
					"taskfile": "/home/Aj4x/go/src/github.com/Aj4x/tash/examples/Taskfile.yml"
				}
			}
		],
		"location": "/home/Aj4x/go/src/github.com/Aj4x/tash/examples/Taskfile.yml"
	}`

	expectedTasks := []task.Task{
		{
			Id:      "cowsay",
			Desc:    "Displays a cute ASCII art cow with a greeting message, mimicking the 'cowsay' program",
			Summary: "ASCII cow art",
			Aliases: []string{"cow", "moo"},
		},
		{
			Id:      "date-time",
			Desc:    "Shows the current date and time along with a calendar for the current month",
			Summary: "Display current date and time",
			Aliases: []string{"date", "time", "dt"},
		},
	}

	tasks, err := parseTasksJson(inputJson)
	if err != nil {
		t.Fatalf("parseTasksJson() error = %v", err)
	}

	if len(tasks) != len(expectedTasks) {
		t.Errorf("Expected %d tasks, but got %d", len(expectedTasks), len(tasks))
	}

	for i, tk := range tasks {
		expectedTaskJson, _ := json.Marshal(expectedTasks[i])
		actualTaskJson, _ := json.Marshal(tk)
		if string(expectedTaskJson) != string(actualTaskJson) {
			t.Errorf("Mismatch at task %d\nExpected: %s\nGot: %s", i, expectedTaskJson, actualTaskJson)
		}
	}
}
