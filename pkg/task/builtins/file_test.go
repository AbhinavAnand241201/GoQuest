package builtins

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestFileTask(t *testing.T) {
	// Create test directory
	testDir := "testdata"
	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	// Test file paths
	sourceFile := filepath.Join(testDir, "source.txt")
	targetFile := filepath.Join(testDir, "target.txt")
	moveTarget := filepath.Join(testDir, "moved.txt")
	content := "test content"

	// Test write
	t.Run("Write", func(t *testing.T) {
		task := Write(sourceFile).
			WithContent(content).
			WithMode(0644)

		_, err := task.Run(context.Background())
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		// Verify file exists and has correct content
		data, err := os.ReadFile(sourceFile)
		if err != nil {
			t.Errorf("Failed to read written file: %v", err)
		}
		if string(data) != content {
			t.Errorf("Expected content %s, got %s", content, string(data))
		}
	})

	// Test read
	t.Run("Read", func(t *testing.T) {
		task := Read(sourceFile)

		result, err := task.Run(context.Background())
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if result != content {
			t.Errorf("Expected content %s, got %v", content, result)
		}
	})

	// Test copy
	t.Run("Copy", func(t *testing.T) {
		task := Copy(sourceFile).
			WithTarget(targetFile)

		_, err := task.Run(context.Background())
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		// Verify target file exists and has correct content
		data, err := os.ReadFile(targetFile)
		if err != nil {
			t.Errorf("Failed to read copied file: %v", err)
		}
		if string(data) != content {
			t.Errorf("Expected content %s, got %s", content, string(data))
		}
	})

	// Test move
	t.Run("Move", func(t *testing.T) {
		task := Move(targetFile).
			WithTarget(moveTarget)

		_, err := task.Run(context.Background())
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		// Verify source file is gone and target exists
		if _, err := os.Stat(targetFile); !os.IsNotExist(err) {
			t.Error("Source file still exists after move")
		}
		if _, err := os.Stat(moveTarget); err != nil {
			t.Errorf("Target file does not exist after move: %v", err)
		}
	})

	// Test delete
	t.Run("Delete", func(t *testing.T) {
		task := Delete(moveTarget)

		_, err := task.Run(context.Background())
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		// Verify file is deleted
		if _, err := os.Stat(moveTarget); !os.IsNotExist(err) {
			t.Error("File still exists after delete")
		}
	})

	// Test error cases
	t.Run("Error cases", func(t *testing.T) {
		// Test read non-existent file
		task := Read("non-existent.txt")
		_, err := task.Run(context.Background())
		if err == nil {
			t.Error("Expected error for non-existent file, got nil")
		}

		// Test invalid operation
		task = &FileTask{
			Operation: "invalid",
			Source:    "test.txt",
		}
		_, err = task.Run(context.Background())
		if err == nil {
			t.Error("Expected error for invalid operation, got nil")
		}
	})
}
