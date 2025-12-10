package output

import (
	"io"

	"github.com/aryankumar/fleet/internal/executor"
	"gopkg.in/yaml.v3"
)

// YAMLFormatter formats output as YAML
type YAMLFormatter struct {
	options *Options
}

// NewYAMLFormatter creates a new YAML formatter
func NewYAMLFormatter(opts *Options) *YAMLFormatter {
	if opts == nil {
		opts = &Options{}
	}
	return &YAMLFormatter{
		options: opts,
	}
}

// Format outputs a single data item as YAML
func (f *YAMLFormatter) Format(w io.Writer, data interface{}) error {
	encoder := yaml.NewEncoder(w)
	encoder.SetIndent(2)
	defer encoder.Close()

	return encoder.Encode(data)
}

// FormatMultiCluster outputs multiple cluster results as YAML
func (f *YAMLFormatter) FormatMultiCluster(w io.Writer, results []executor.Result) error {
	// Convert results to a more YAML-friendly structure
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

	encoder := yaml.NewEncoder(w)
	encoder.SetIndent(2)
	defer encoder.Close()

	return encoder.Encode(output)
}
