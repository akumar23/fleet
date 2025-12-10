package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/aryankumar/fleet/pkg/version"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// newVersionCmd creates the version command
func newVersionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Long:  "Display detailed version information for the Fleet CLI",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVersion(cmd)
		},
	}

	return cmd
}

func runVersion(cmd *cobra.Command) error {
	info := version.Get()
	outputFormat, _ := cmd.Flags().GetString("output")

	switch outputFormat {
	case "json":
		return outputJSON(info)
	case "yaml":
		return outputYAML(info)
	case "table":
		return outputTable(info)
	default:
		// Default to human-readable format
		fmt.Println(info.String())
		return nil
	}
}

func outputJSON(info version.Info) error {
	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal version info to JSON: %w", err)
	}
	fmt.Println(string(data))
	return nil
}

func outputYAML(info version.Info) error {
	data, err := yaml.Marshal(info)
	if err != nil {
		return fmt.Errorf("failed to marshal version info to YAML: %w", err)
	}
	fmt.Print(string(data))
	return nil
}

func outputTable(info version.Info) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "COMPONENT\tVALUE")
	fmt.Fprintf(w, "Version\t%s\n", info.Version)
	fmt.Fprintf(w, "Commit\t%s\n", info.Commit)
	fmt.Fprintf(w, "Build Time\t%s\n", info.BuildTime)
	fmt.Fprintf(w, "Go Version\t%s\n", info.GoVersion)
	fmt.Fprintf(w, "Platform\t%s\n", info.Platform)
	return w.Flush()
}
