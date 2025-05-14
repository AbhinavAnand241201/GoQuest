package task

import (
	"context"
	"testing"
	"time"
)

func TestTaskSpec_Validate(t *testing.T) {
	tests := []struct {
		name    string
		spec    TaskSpec
		wantErr bool
	}{
		{
			name: "valid task spec",
			spec: TaskSpec{
				Name: "test-task",
				Run: func(ctx context.Context) (interface{}, error) {
					return "test", nil
				},
				Depends: []string{"dep1"},
			},
			wantErr: false,
		},
		{
			name: "empty name",
			spec: TaskSpec{
				Name: "",
				Run: func(ctx context.Context) (interface{}, error) {
					return "test", nil
				},
			},
			wantErr: true,
		},
		{
			name: "nil run function",
			spec: TaskSpec{
				Name: "test-task",
				Run:  nil,
			},
			wantErr: true,
		},
		{
			name: "empty dependency name",
			spec: TaskSpec{
				Name: "test-task",
				Run: func(ctx context.Context) (interface{}, error) {
					return "test", nil
				},
				Depends: []string{""},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.spec.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("TaskSpec.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTask_Run(t *testing.T) {
	ctx := context.Background()
	expected := "test result"

	task := NewTask(TaskSpec{
		Name: "test-task",
		Run: func(ctx context.Context) (interface{}, error) {
			return expected, nil
		},
	})

	result, err := task.Run(ctx)
	if err != nil {
		t.Errorf("Task.Run() error = %v", err)
	}
	if result != expected {
		t.Errorf("Task.Run() = %v, want %v", result, expected)
	}
}

func TestTask_RunWithContextCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	task := NewTask(TaskSpec{
		Name: "test-task",
		Run: func(ctx context.Context) (interface{}, error) {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(200 * time.Millisecond):
				return "test", nil
			}
		},
	})

	result, err := task.Run(ctx)
	if err != context.DeadlineExceeded {
		t.Errorf("Task.Run() error = %v, want %v", err, context.DeadlineExceeded)
	}
	if result != nil {
		t.Errorf("Task.Run() = %v, want nil", result)
	}
}
