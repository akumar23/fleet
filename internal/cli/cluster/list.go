package cluster

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"sort"
	"strings"

	"github.com/aryankumar/fleet/internal/config"
	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// newListCmd creates the cluster list command
func newListCmd() *cobra.Command {
	var (
		showLabels bool
		outputFormat string
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all available Kubernetes clusters",
		Long: `List all Kubernetes clusters from your kubeconfig file(s).

This command displays all available contexts, showing the current context,
cluster names, servers, namespaces, and users. It supports multiple kubeconfig
sources including the KUBECONFIG environment variable.`,
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(cmd, showLabels, outputFormat)
		},
	}

	cmd.Flags().BoolVar(&showLabels, "show-labels", false, "show cluster labels from fleet config")
	cmd.Flags().StringVarP(&outputFormat, "output", "o", "", "output format (table, json, yaml)")

	return cmd
}

func runList(cmd *cobra.Command, showLabels bool, outputFormat string) error {
	logger := slog.Default()

	// Get kubeconfig path from flag or viper
	kubeconfigPath := viper.GetString("kubeconfig")

	logger.Debug("loading kubeconfig", "path", kubeconfigPath)

	// Create kubeconfig loader
	loader := config.NewKubeconfigLoader(kubeconfigPath)

	// Log the paths being used
	paths := loader.GetPaths()
	logger.Debug("using kubeconfig paths", "paths", strings.Join(paths, ", "))

	// Load cluster information
	clusters, err := loader.GetClusters()
	if err != nil {
		return fmt.Errorf("failed to load clusters: %w", err)
	}

	if len(clusters) == 0 {
		fmt.Fprintf(os.Stderr, "No clusters found in kubeconfig\n")
		return nil
	}

	// Load fleet configuration to merge cluster metadata
	cfgPath := viper.GetString("config")
	configManager := config.NewManager(cfgPath)
	if fleetConfig, err := configManager.Load(); err == nil {
		logger.Debug("loaded fleet config", "clusters", len(fleetConfig.Clusters))
		clusters = configManager.MergeClusterInfo(clusters)
	} else {
		logger.Debug("no fleet config loaded, using kubeconfig only")
	}

	// Sort clusters by context name for consistent output
	sort.Slice(clusters, func(i, j int) bool {
		// Current context always comes first
		if clusters[i].Current {
			return true
		}
		if clusters[j].Current {
			return false
		}
		return clusters[i].Context < clusters[j].Context
	})

	// Determine output format
	if outputFormat == "" {
		outputFormat = viper.GetString("output")
	}
	if outputFormat == "" {
		outputFormat = "table"
	}

	// Generate output based on format
	switch outputFormat {
	case "json":
		return outputJSON(clusters)
	case "yaml":
		return outputYAML(clusters)
	case "table":
		return outputTable(clusters, showLabels, viper.GetBool("no-color"))
	default:
		return fmt.Errorf("unsupported output format: %s (supported: table, json, yaml)", outputFormat)
	}
}

func outputTable(clusters []config.ClusterInfo, showLabels bool, noColor bool) error {
	table := tablewriter.NewWriter(os.Stdout)

	// Set up headers
	headers := []string{"Current", "Context", "Cluster", "Server", "Namespace", "User"}
	if showLabels {
		headers = append(headers, "Labels")
	}
	table.SetHeader(headers)

	// Configure table style
	table.SetAutoWrapText(false)
	table.SetAutoFormatHeaders(true)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetCenterSeparator("")
	table.SetColumnSeparator("")
	table.SetRowSeparator("")
	table.SetHeaderLine(false)
	table.SetBorder(false)
	table.SetTablePadding("\t")
	table.SetNoWhiteSpace(true)

	// Color setup
	var (
		currentMarker = "*"
		greenBold     = color.New(color.FgGreen, color.Bold)
		cyan          = color.New(color.FgCyan)
		yellow        = color.New(color.FgYellow)
	)

	if noColor {
		color.NoColor = true
	}

	// Add rows
	for _, cluster := range clusters {
		row := make([]string, 0, len(headers))

		// Current indicator
		current := ""
		if cluster.Current {
			current = currentMarker
		}
		row = append(row, current)

		// Context (bold green if current)
		contextStr := cluster.Context
		if cluster.Current && !noColor {
			contextStr = greenBold.Sprint(contextStr)
		}
		row = append(row, contextStr)

		// Cluster name (with alias if available)
		clusterName := cluster.Name
		if cluster.Alias != "" && cluster.Alias != cluster.Context {
			if !noColor {
				clusterName = fmt.Sprintf("%s (%s)", clusterName, cyan.Sprint(cluster.Alias))
			} else {
				clusterName = fmt.Sprintf("%s (%s)", clusterName, cluster.Alias)
			}
		}
		row = append(row, clusterName)

		// Server URL (truncate if too long)
		server := cluster.Server
		if len(server) > 50 {
			server = server[:47] + "..."
		}
		row = append(row, server)

		// Namespace
		namespace := cluster.Namespace
		if namespace == "" {
			namespace = "default"
		}
		row = append(row, namespace)

		// User
		user := cluster.User
		if len(user) > 30 {
			user = user[:27] + "..."
		}
		row = append(row, user)

		// Labels
		if showLabels {
			labelStr := formatLabels(cluster.Labels)
			if !noColor && labelStr != "" {
				labelStr = yellow.Sprint(labelStr)
			}
			row = append(row, labelStr)
		}

		table.Append(row)
	}

	table.Render()

	// Print summary
	fmt.Fprintf(os.Stdout, "\nTotal clusters: %d\n", len(clusters))

	return nil
}

func outputJSON(clusters []config.ClusterInfo) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(clusters)
}

func outputYAML(clusters []config.ClusterInfo) error {
	encoder := yaml.NewEncoder(os.Stdout)
	defer encoder.Close()
	return encoder.Encode(clusters)
}

func formatLabels(labels map[string]string) string {
	if len(labels) == 0 {
		return ""
	}

	pairs := make([]string, 0, len(labels))
	for k, v := range labels {
		pairs = append(pairs, fmt.Sprintf("%s=%s", k, v))
	}
	sort.Strings(pairs)
	return strings.Join(pairs, ",")
}
