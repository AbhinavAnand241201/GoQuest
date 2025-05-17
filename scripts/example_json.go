package main

import (
	"context"
	"time"

	"github.com/AbhinavAnand241201/goquest/pkg/task"
)

// JsonTasks demonstrates task execution with JSON output
type JsonTasks struct {
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
	tasks := JsonTasks{}

	// Task1: Simple success
	tasks.Task1.Run = func(ctx context.Context) (interface{}, error) {
		time.Sleep(100 * time.Millisecond)
		return "Hello from task1", nil
	}

	// Task2: Success with delay
	tasks.Task2.Run = func(ctx context.Context) (interface{}, error) {
		time.Sleep(1 * time.Second)
		return "Delayed task", nil
	}

	// Task3: Success with number result
	tasks.Task3.Run = func(ctx context.Context) (interface{}, error) {
		return 42, nil
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
