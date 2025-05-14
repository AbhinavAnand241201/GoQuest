package output

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/AbhinavAnand241201/goquest/pkg/task"
)

// TaskResultJSON represents a task result in JSON format
type TaskResultJSON struct {
	Name     string      `json:"name"`
	Status   string      `json:"status"`
	Result   interface{} `json:"result,omitempty"`
	Error    string      `json:"error,omitempty"`
	Duration string      `json:"duration"`
}

// ConvertTaskResults converts task results to JSON format
func ConvertTaskResults(results []task.TaskResult) []TaskResultJSON {
	jsonResults := make([]TaskResultJSON, len(results))
	for i, result := range results {
		jsonResults[i] = TaskResultJSON{
			Name:     result.Name,
			Status:   result.Status,
			Result:   result.Result,
			Duration: result.Duration.String(),
		}
		if result.Error != nil {
			jsonResults[i].Error = result.Error.Error()
		}
	}
	return jsonResults
}

// PrintTaskResults prints task results in JSON format
func PrintTaskResults(results []task.TaskResult) error {
	jsonResults := ConvertTaskResults(results)
	jsonData, err := json.MarshalIndent(jsonResults, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling task results: %w", err)
	}
	fmt.Println(string(jsonData))
	return nil
}

// PrintError prints an error in JSON format
func PrintError(err error) error {
	errorJSON := struct {
		Error string `json:"error"`
	}{
		Error: err.Error(),
	}
	jsonData, err := json.MarshalIndent(errorJSON, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling error: %w", err)
	}
	fmt.Fprintln(os.Stderr, string(jsonData))
	return nil
}
