package main

import (
	"context"
	"fmt"
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
		{
			Name: "error_task",
			Run: func(ctx context.Context) (interface{}, error) {
				// Simulate a task that fails
				time.Sleep(200 * time.Millisecond)
				return nil, fmt.Errorf("this task always fails")
			},
			Depends: []string{},
		},
		{
			Name: "dependent_error_task",
			Run: func(ctx context.Context) (interface{}, error) {
				// This task depends on error_task and should not run
				time.Sleep(300 * time.Millisecond)
				return "This task should not run", nil
			},
			Depends: []string{"error_task"},
		},
	})
}
