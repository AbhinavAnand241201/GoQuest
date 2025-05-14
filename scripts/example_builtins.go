package main

import (
	"context"
	"time"

	"github.com/AbhinavAnand241201/goquest/pkg/task"
	"github.com/AbhinavAnand241201/goquest/pkg/task/builtins"
)

// Tasks demonstrates the use of built-in task types
type BuiltinTasks struct {
	HTTPTask struct {
		Run func(context.Context) (interface{}, error) `task:"name=http_task"`
	} `task:"name=http_task"`

	FileReadTask struct {
		Run func(context.Context) (interface{}, error) `task:"name=file_read_task"`
	} `task:"name=file_read_task,depends=http_task"`

	FileWriteTask struct {
		Run func(context.Context) (interface{}, error) `task:"name=file_write_task"`
	} `task:"name=file_write_task,depends=file_read_task"`

	FileDeleteTask struct {
		Run func(context.Context) (interface{}, error) `task:"name=file_delete_task"`
	} `task:"name=file_delete_task,depends=file_write_task"`
}

func init() {
	tasks := BuiltinTasks{}

	// HTTP Task: Fetch data from a public API
	tasks.HTTPTask.Run = func(ctx context.Context) (interface{}, error) {
		httpTask := builtins.Get("https://api.github.com/users/golang").
			WithHeader("Accept", "application/json").
			WithTimeout(10 * time.Second)
		return httpTask.Run(ctx)
	}

	// File Read Task: Read a local file
	tasks.FileReadTask.Run = func(ctx context.Context) (interface{}, error) {
		readTask := builtins.Read("data/input.txt")
		return readTask.Run(ctx)
	}

	// File Write Task: Write processed data to a file
	tasks.FileWriteTask.Run = func(ctx context.Context) (interface{}, error) {
		writeTask := builtins.Write("data/output.txt").
			WithContent("Processed data from HTTP and file operations").
			WithMode(0644)
		return writeTask.Run(ctx)
	}

	// File Delete Task: Clean up temporary files
	tasks.FileDeleteTask.Run = func(ctx context.Context) (interface{}, error) {
		deleteTask := builtins.Delete("data/temp.txt")
		return deleteTask.Run(ctx)
	}

	// Register tasks
	if err := task.RegisterTasks([]task.TaskSpec{
		{
			Name:    "http_task",
			Run:     tasks.HTTPTask.Run,
			Depends: []string{},
		},
		{
			Name:    "file_read_task",
			Run:     tasks.FileReadTask.Run,
			Depends: []string{"http_task"},
		},
		{
			Name:    "file_write_task",
			Run:     tasks.FileWriteTask.Run,
			Depends: []string{"file_read_task"},
		},
		{
			Name:    "file_delete_task",
			Run:     tasks.FileDeleteTask.Run,
			Depends: []string{"file_write_task"},
		},
	}); err != nil {
		panic(err)
	}
}
