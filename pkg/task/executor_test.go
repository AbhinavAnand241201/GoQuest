package task

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestExecutor_Run(t *testing.T) {
	tests := []struct {
		name         string
		tasks        []TaskRunner
		concurrency  int
		timeout      time.Duration
		wantResults  int
		wantErrors   bool
		wantDuration time.Duration
	}{
		{
			name: "single task success",
			tasks: []TaskRunner{
				NewTask(TaskSpec{
					Name: "task1",
					Run: func(ctx context.Context) (interface{}, error) {
						return "success", nil
					},
				}),
			},
			concurrency:  1,
			timeout:      time.Second,
			wantResults:  1,
			wantErrors:   false,
			wantDuration: 100 * time.Millisecond,
		},
		{
			name: "multiple tasks with errors",
			tasks: []TaskRunner{
				NewTask(TaskSpec{
					Name: "task1",
					Run: func(ctx context.Context) (interface{}, error) {
						return "success", nil
					},
				}),
				NewTask(TaskSpec{
					Name: "task2",
					Run: func(ctx context.Context) (interface{}, error) {
						return nil, errors.New("task failed")
					},
				}),
			},
			concurrency:  2,
			timeout:      time.Second,
			wantResults:  2,
			wantErrors:   true,
			wantDuration: 100 * time.Millisecond,
		},
		{
			name: "task timeout",
			tasks: []TaskRunner{
				NewTask(TaskSpec{
					Name: "task1",
					Run: func(ctx context.Context) (interface{}, error) {
						select {
						case <-ctx.Done():
							return nil, ctx.Err()
						case <-time.After(2 * time.Second):
							return "success", nil
						}
					},
				}),
			},
			concurrency:  1,
			timeout:      time.Second,
			wantResults:  1,
			wantErrors:   true,
			wantDuration: time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), tt.timeout)
			defer cancel()

			executor := NewExecutor(tt.tasks, tt.concurrency)
			results := executor.Run(ctx)

			if len(results) != tt.wantResults {
				t.Errorf("got %d results, want %d", len(results), tt.wantResults)
			}

			hasErrors := false
			for _, result := range results {
				if result.Error != nil {
					hasErrors = true
					break
				}
			}

			if hasErrors != tt.wantErrors {
				t.Errorf("got errors=%v, want errors=%v", hasErrors, tt.wantErrors)
			}

			// Check that all tasks completed within a reasonable time
			for _, result := range results {
				// Allow for some timing variance (20% margin)
				maxDuration := tt.timeout + (tt.timeout / 5)
				if result.Duration > maxDuration {
					t.Errorf("task %s took %v, want <= %v", result.Name, result.Duration, maxDuration)
				}
			}
		})
	}
}

func TestExecutor_AggregateErrors(t *testing.T) {
	results := []TaskResult{
		{
			Name:   "task1",
			Error:  errors.New("error 1"),
			Status: "failed",
		},
		{
			Name:   "task2",
			Error:  errors.New("error 2"),
			Status: "failed",
		},
		{
			Name:   "task3",
			Status: "success",
		},
	}

	executor := NewExecutor(nil, 1)
	err := executor.AggregateErrors(results)
	if err == nil {
		t.Error("expected error, got nil")
	}

	expected := "2 tasks failed"
	if err.Error()[:len(expected)] != expected {
		t.Errorf("got error %q, want prefix %q", err.Error(), expected)
	}
}

func TestExecutor_GetFailedTasks(t *testing.T) {
	results := []TaskResult{
		{
			Name:   "task1",
			Error:  errors.New("error 1"),
			Status: "failed",
		},
		{
			Name:   "task2",
			Status: "success",
		},
		{
			Name:   "task3",
			Error:  errors.New("error 3"),
			Status: "failed",
		},
	}

	executor := NewExecutor(nil, 1)
	failed := executor.GetFailedTasks(results)
	if len(failed) != 2 {
		t.Errorf("got %d failed tasks, want 2", len(failed))
	}

	expected := map[string]bool{"task1": true, "task3": true}
	for _, name := range failed {
		if !expected[name] {
			t.Errorf("unexpected failed task: %s", name)
		}
	}
}

func TestExecutor_GetSuccessfulTasks(t *testing.T) {
	results := []TaskResult{
		{
			Name:   "task1",
			Error:  errors.New("error 1"),
			Status: "failed",
		},
		{
			Name:   "task2",
			Status: "success",
		},
		{
			Name:   "task3",
			Status: "success",
		},
	}

	executor := NewExecutor(nil, 1)
	successful := executor.GetSuccessfulTasks(results)
	if len(successful) != 2 {
		t.Errorf("got %d successful tasks, want 2", len(successful))
	}

	expected := map[string]bool{"task2": true, "task3": true}
	for _, name := range successful {
		if !expected[name] {
			t.Errorf("unexpected successful task: %s", name)
		}
	}
}
