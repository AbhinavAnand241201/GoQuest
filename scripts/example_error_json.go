package main

import (
	"context"
	"errors"
	"time"

	"github.com/AbhinavAnand241201/goquest/pkg/task"
)

// Tasks demonstrates error handling with JSON output
type Tasks struct {
	Task1 struct {
		Run func(context.Context) (interface{}, error) `task:"name=task1"`
	} `task:"name=task1"`

	Task2 struct {
		Run func(context.Context) (interface{}, error) `task:"name=task2"`
	} `task:"name=task2,depends=task1"`

	Task3 struct {
		Run func(context.Context) (interface{}, error) `task:"name=task3"`
	} `task:"name=task3,depends=task2"`
}

func init() {
	tasks := Tasks{}

	// Task1: Simple success
	tasks.Task1.Run = func(ctx context.Context) (interface{}, error) {
		time.Sleep(100 * time.Millisecond)
		return "Hello from task1", nil
	}

	// Task2: Failure
	tasks.Task2.Run = func(ctx context.Context) (interface{}, error) {
		time.Sleep(500 * time.Millisecond)
		return nil, errors.New("task2 failed")
	}

	// Task3: Should not run due to Task2 failure
	tasks.Task3.Run = func(ctx context.Context) (interface{}, error) {
		return "This should not run", nil
	}

	// Register tasks
	if err := task.RegisterTasks([]task.TaskSpec{
		{
			Name:    "task1",
			Run:     tasks.Task1.Run,
			Depends: []string{},
		},
		{
			Name:    "task2",
			Run:     tasks.Task2.Run,
			Depends: []string{"task1"},
		},
		{
			Name:    "task3",
			Run:     tasks.Task3.Run,
			Depends: []string{"task2"},
		},
	}); err != nil {
		panic(err)
	}
}
