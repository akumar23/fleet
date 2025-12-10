package output

import (
	"encoding/json"
	"io"

	"github.com/aryankumar/fleet/internal/executor"
)

// JSONFormatter formats output as JSON
type JSONFormatter struct {
	options *Options
}

// NewJSONFormatter creates a new JSON formatter
func NewJSONFormatter(opts *Options) *JSONFormatter {
	if opts == nil {
		opts = &Options{}
	}
	return &JSONFormatter{
		options: opts,
	}
}

// Format outputs a single data item as JSON
func (f *JSONFormatter) Format(w io.Writer, data interface{}) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// FormatMultiCluster outputs multiple cluster results as JSON
func (f *JSONFormatter) FormatMultiCluster(w io.Writer, results []executor.Result) error {
	// Convert results to a more JSON-friendly structure
	output := make([]map[string]interface{}, len(results))

	for i, result := range results {
		item := map[string]interface{}{
			"cluster":  result.ClusterName,
			"duration": result.Duration.String(),
		}

		if result.Error != nil {
			item["status"] = "failed"
			item["error"] = result.Error.Error()
		} else {
			item["status"] = "success"
			item["data"] = result.Data
		}

		output[i] = item
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}
