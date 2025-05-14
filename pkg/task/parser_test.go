package task

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestParser_ParseScript(t *testing.T) {
	// Create a temporary directory for test scripts
	tmpDir, err := os.MkdirTemp("", "goquest-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test scripts
	validScript := filepath.Join(tmpDir, "valid.go")
	invalidScript := filepath.Join(tmpDir, "invalid.go")
	emptyScript := filepath.Join(tmpDir, "empty.go")

	// Write valid script
	err = os.WriteFile(validScript, []byte(`
package main

import "context"

type Tasks struct {
	Task1 struct {
		Run func(context.Context) (interface{}, error) `+"`task:\"name=task1\"`"+`
		Depends []string `+"`task:\"depends=\"`"+`
	} `+"`task:\"name=task1\"`"+`
	Task2 struct {
		Run func(context.Context) (interface{}, error) `+"`task:\"name=task2\"`"+`
		Depends []string `+"`task:\"depends=task1\"`"+`
	} `+"`task:\"name=task2\"`"+`
}

func init() {
	tasks := Tasks{
		Task1: struct {
			Run func(context.Context) (interface{}, error)
			Depends []string
		}{
			Run: func(ctx context.Context) (interface{}, error) {
				return "Hello from task1", nil
			},
			Depends: []string{},
		},
		Task2: struct {
			Run func(context.Context) (interface{}, error)
			Depends []string
		}{
			Run: func(ctx context.Context) (interface{}, error) {
				time.Sleep(100 * time.Millisecond)
				return "Delayed task", nil
			},
			Depends: []string{"task1"},
		},
	}
}
`), 0644)
	if err != nil {
		t.Fatalf("failed to write valid script: %v", err)
	}

	// Write invalid script
	err = os.WriteFile(invalidScript, []byte(`
package main

type InvalidTasks struct {
	TaskNoName struct {
		Run func(context.Context) (interface{}, error) `+"`task:\"name=\"`"+`
		Depends []string `+"`task:\"depends=\"`"+`
	} `+"`task:\"name=\"`"+`
}
`), 0644)
	if err != nil {
		t.Fatalf("failed to write invalid script: %v", err)
	}

	// Write empty script
	err = os.WriteFile(emptyScript, []byte(`
package main
`), 0644)
	if err != nil {
		t.Fatalf("failed to write empty script: %v", err)
	}

	tests := []struct {
		name    string
		script  string
		want    int
		wantErr bool
	}{
		{
			name:    "valid script",
			script:  validScript,
			want:    2,
			wantErr: false,
		},
		{
			name:    "invalid script",
			script:  invalidScript,
			want:    0,
			wantErr: true,
		},
		{
			name:    "empty script",
			script:  emptyScript,
			want:    0,
			wantErr: true,
		},
	}

	parser := NewParser()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tasks, err := parser.ParseScript(tt.script)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseScript() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(tasks) != tt.want {
				t.Errorf("ParseScript() got %d tasks, want %d", len(tasks), tt.want)
			}
		})
	}
}

func TestValidateTaskStruct(t *testing.T) {
	tests := []struct {
		name    string
		task    interface{}
		wantErr bool
	}{
		{
			name: "valid task struct",
			task: struct {
				Task1 struct {
					Run     func(context.Context) (interface{}, error) `task:"name=task1"`
					Depends []string                                   `task:"depends="`
				} `task:"name=task1"`
			}{
				Task1: struct {
					Run     func(context.Context) (interface{}, error) `task:"name=task1"`
					Depends []string                                   `task:"depends="`
				}{
					Run: func(ctx context.Context) (interface{}, error) {
						return "Hello", nil
					},
					Depends: []string{},
				},
			},
			wantErr: false,
		},
		{
			name: "missing task name",
			task: struct {
				Task1 struct {
					Run     func(context.Context) (interface{}, error) `task:"name="`
					Depends []string                                   `task:"depends="`
				} `task:"name="`
			}{
				Task1: struct {
					Run     func(context.Context) (interface{}, error) `task:"name="`
					Depends []string                                   `task:"depends="`
				}{
					Run: func(ctx context.Context) (interface{}, error) {
						return "Hello", nil
					},
					Depends: []string{},
				},
			},
			wantErr: true,
		},
		{
			name: "nil run function",
			task: struct {
				Task1 struct {
					Run     func(context.Context) (interface{}, error) `task:"name=task1"`
					Depends []string                                   `task:"depends="`
				} `task:"name=task1"`
			}{
				Task1: struct {
					Run     func(context.Context) (interface{}, error) `task:"name=task1"`
					Depends []string                                   `task:"depends="`
				}{
					Run:     nil,
					Depends: []string{},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid run function signature",
			task: struct {
				Task1 struct {
					Run     func() error `task:"name=task1"`
					Depends []string     `task:"depends="`
				} `task:"name=task1"`
			}{
				Task1: struct {
					Run     func() error `task:"name=task1"`
					Depends []string     `task:"depends="`
				}{
					Run: func() error {
						return nil
					},
					Depends: []string{},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTaskStruct(tt.task)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTaskStruct() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
