package main

import (
	"context"
	"time"

	"github.com/AbhinavAnand241201/goquest/pkg/task"
)

func init() {
	task.RegisterTasks([]task.TaskSpec{
		{
			Name: "hello_task",
			Run: func(ctx context.Context) (interface{}, error) {
				// Simulate some work
				time.Sleep(100 * time.Millisecond)
				return "Hello from GoQuest!", nil
			},
			Depends: []string{},
		},
		{
			Name: "delayed_task",
			Run: func(ctx context.Context) (interface{}, error) {
				// Simulate a longer task
				time.Sleep(500 * time.Millisecond)
				return "This task took longer to complete", nil
			},
			Depends: []string{"hello_task"},
		},
	})
}
