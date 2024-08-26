package main

import (
	"context"
	"ctl/settings"
	"ctl/task"
	"fmt"
	"os"
)

func main() {
	paramsAmount := len(os.Args)

	if paramsAmount < 2 {
		fmt.Println("Wrong args amount")
		fmt.Println("Set \"Task type\" value as 1-st param")
		os.Exit(125)

		return
	}

	s, err := initSettings("settings.json")
	if err != nil {
		fmt.Println("Failed to read settings: " + err.Error())
		os.Exit(1)
	}

	taskType := task.Type(os.Args[1])
	if len(taskType) == 0 {
		fmt.Println("Set \"Task type\" value as 1-st param")
		os.Exit(2)
		return
	}

	t, err := task.GetTaskByType(taskType, s, os.Args[1])
	if err != nil {
		fmt.Printf("Task type validation error: %v\n", err)
		os.Exit(3)
		return
	}

	if err := t.Run(context.Background()); err != nil {
		fmt.Printf("Task \"%s\" failed: %v\n", taskType, err)
		os.Exit(4)
	}
}

func initSettings(path string) (settings.Settings, error) {
	s := settings.Settings{}
	err := s.Read(path)
	return s, err
}
