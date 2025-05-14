package builtins

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// FileTask represents a task that performs file operations
type FileTask struct {
	Operation string
	Source    string
	Target    string
	Content   string
	Mode      os.FileMode
}

// NewFileTask creates a new file task
func NewFileTask(operation, source string) *FileTask {
	return &FileTask{
		Operation: operation,
		Source:    source,
		Mode:      0644,
	}
}

// WithTarget sets the target path for operations like copy or move
func (f *FileTask) WithTarget(target string) *FileTask {
	f.Target = target
	return f
}

// WithContent sets the content for write operations
func (f *FileTask) WithContent(content string) *FileTask {
	f.Content = content
	return f
}

// WithMode sets the file mode
func (f *FileTask) WithMode(mode os.FileMode) *FileTask {
	f.Mode = mode
	return f
}

// Run executes the file operation
func (f *FileTask) Run(ctx context.Context) (interface{}, error) {
	switch f.Operation {
	case "read":
		return f.read()
	case "write":
		return f.write()
	case "copy":
		return f.copy()
	case "move":
		return f.move()
	case "delete":
		return f.delete()
	default:
		return nil, fmt.Errorf("unsupported file operation: %s", f.Operation)
	}
}

// read reads the contents of a file
func (f *FileTask) read() (interface{}, error) {
	data, err := os.ReadFile(f.Source)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	return string(data), nil
}

// write writes content to a file
func (f *FileTask) write() (interface{}, error) {
	if err := os.WriteFile(f.Source, []byte(f.Content), f.Mode); err != nil {
		return nil, fmt.Errorf("failed to write file: %w", err)
	}
	return fmt.Sprintf("Successfully wrote %d bytes to %s", len(f.Content), f.Source), nil
}

// copy copies a file from source to target
func (f *FileTask) copy() (interface{}, error) {
	src, err := os.Open(f.Source)
	if err != nil {
		return nil, fmt.Errorf("failed to open source file: %w", err)
	}
	defer src.Close()

	// Create target directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(f.Target), 0755); err != nil {
		return nil, fmt.Errorf("failed to create target directory: %w", err)
	}

	dst, err := os.Create(f.Target)
	if err != nil {
		return nil, fmt.Errorf("failed to create target file: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return nil, fmt.Errorf("failed to copy file: %w", err)
	}

	return fmt.Sprintf("Successfully copied %s to %s", f.Source, f.Target), nil
}

// move moves a file from source to target
func (f *FileTask) move() (interface{}, error) {
	if err := os.Rename(f.Source, f.Target); err != nil {
		return nil, fmt.Errorf("failed to move file: %w", err)
	}
	return fmt.Sprintf("Successfully moved %s to %s", f.Source, f.Target), nil
}

// delete deletes a file
func (f *FileTask) delete() (interface{}, error) {
	if err := os.Remove(f.Source); err != nil {
		return nil, fmt.Errorf("failed to delete file: %w", err)
	}
	return fmt.Sprintf("Successfully deleted %s", f.Source), nil
}

// Read creates a new file read task
func Read(path string) *FileTask {
	return NewFileTask("read", path)
}

// Write creates a new file write task
func Write(path string) *FileTask {
	return NewFileTask("write", path)
}

// Copy creates a new file copy task
func Copy(source string) *FileTask {
	return NewFileTask("copy", source)
}

// Move creates a new file move task
func Move(source string) *FileTask {
	return NewFileTask("move", source)
}

// Delete creates a new file delete task
func Delete(path string) *FileTask {
	return NewFileTask("delete", path)
}
