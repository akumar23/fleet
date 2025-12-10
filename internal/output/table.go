package output

import (
	"fmt"
	"io"
	"strings"

	"github.com/aryankumar/fleet/internal/executor"
	"github.com/olekukonko/tablewriter"
)

// TableFormatter formats output as a table (kubectl-style)
type TableFormatter struct {
	options *Options
}

// NewTableFormatter creates a new table formatter
func NewTableFormatter(opts *Options) *TableFormatter {
	if opts == nil {
		opts = &Options{}
	}
	return &TableFormatter{
		options: opts,
	}
}

// Format outputs a single data item as a table
func (f *TableFormatter) Format(w io.Writer, data interface{}) error {
	table := f.createTable(w)

	// Handle different data types
	switch v := data.(type) {
	case map[string]interface{}:
		return f.formatMap(table, v)
	case []map[string]interface{}:
		return f.formatMapSlice(table, v)
	case string:
		fmt.Fprintln(w, v)
		return nil
	default:
		// Fallback to simple string representation
		fmt.Fprintln(w, v)
		return nil
	}
}

// FormatMultiCluster outputs multiple cluster results as a table
func (f *TableFormatter) FormatMultiCluster(w io.Writer, results []executor.Result) error {
	if len(results) == 0 {
		fmt.Fprintln(w, "No results")
		return nil
	}

	// Create color scheme
	colors := NewColorScheme(w, f.options.NoColor)

	// Group results by cluster
	table := f.createTable(w)

	// Set headers
	headers := []string{"CLUSTER", "STATUS", "DURATION"}
	if f.options.Wide {
		headers = append(headers, "DATA")
	}

	if !f.options.NoHeaders {
		if colors.Disabled {
			table.SetHeader(headers)
		} else {
			coloredHeaders := make([]string, len(headers))
			for i, h := range headers {
				coloredHeaders[i] = colors.Header(h)
			}
			table.SetHeader(coloredHeaders)
		}
	}

	// Add rows for each result
	for _, result := range results {
		row := f.formatResultRow(result, colors)
		table.Append(row)
	}

	table.Render()

	// Print summary
	f.printSummary(w, results, colors)

	return nil
}

// formatResultRow formats a single result as a table row
func (f *TableFormatter) formatResultRow(result executor.Result, colors *ColorScheme) []string {
	// Cluster name
	clusterName := result.ClusterName
	if !colors.Disabled {
		clusterName = colors.ClusterName(clusterName)
	}

	// Status
	status := "Success"
	if result.Error != nil {
		status = "Failed"
	}
	if !colors.Disabled {
		status = colors.StatusColor(result.Error != nil)(status)
	}

	// Duration
	duration := result.Duration.String()
	if !colors.Disabled {
		duration = colors.Duration(duration)
	}

	row := []string{clusterName, status, duration}

	// Add data column if wide mode
	if f.options.Wide {
		dataStr := ""
		if result.Error != nil {
			dataStr = result.Error.Error()
		} else if result.Data != nil {
			dataStr = fmt.Sprintf("%v", result.Data)
			// Truncate long data
			if len(dataStr) > 50 {
				dataStr = dataStr[:47] + "..."
			}
		}
		row = append(row, dataStr)
	}

	return row
}

// formatMap formats a map as a two-column table (key-value pairs)
func (f *TableFormatter) formatMap(table *tablewriter.Table, data map[string]interface{}) error {
	if !f.options.NoHeaders {
		table.SetHeader([]string{"KEY", "VALUE"})
	}

	for k, v := range data {
		table.Append([]string{k, fmt.Sprintf("%v", v)})
	}

	table.Render()
	return nil
}

// formatMapSlice formats a slice of maps as a table
func (f *TableFormatter) formatMapSlice(table *tablewriter.Table, data []map[string]interface{}) error {
	if len(data) == 0 {
		return nil
	}

	// Extract headers from the first map
	var headers []string
	for k := range data[0] {
		headers = append(headers, strings.ToUpper(k))
	}

	if !f.options.NoHeaders {
		table.SetHeader(headers)
	}

	// Add rows
	for _, item := range data {
		var row []string
		for _, h := range headers {
			key := strings.ToLower(h)
			row = append(row, fmt.Sprintf("%v", item[key]))
		}
		table.Append(row)
	}

	table.Render()
	return nil
}

// createTable creates a new table with kubectl-style configuration
func (f *TableFormatter) createTable(w io.Writer) *tablewriter.Table {
	table := tablewriter.NewWriter(w)

	// kubectl-style configuration
	table.SetAutoWrapText(false)
	table.SetAutoFormatHeaders(true)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetCenterSeparator("")
	table.SetColumnSeparator("")
	table.SetRowSeparator("")
	table.SetHeaderLine(false)
	table.SetBorder(false)
	table.SetTablePadding("\t") // Tab-separated like kubectl
	table.SetNoWhiteSpace(true)

	return table
}

// printSummary prints a summary of the results
func (f *TableFormatter) printSummary(w io.Writer, results []executor.Result, colors *ColorScheme) {
	summary := executor.Summarize(results)

	fmt.Fprintln(w, "")
	fmt.Fprintf(w, "Summary: ")

	successText := fmt.Sprintf("%d successful", summary.Successful)
	if !colors.Disabled {
		successText = colors.Success(successText)
	}

	failedText := fmt.Sprintf("%d failed", summary.Failed)
	if !colors.Disabled && summary.Failed > 0 {
		failedText = colors.Error(failedText)
	}

	durationText := fmt.Sprintf("avg=%s", summary.AvgDuration.Round(1000))
	if !colors.Disabled {
		durationText = colors.Duration(durationText)
	}

	fmt.Fprintf(w, "%s, %s, %s\n", successText, failedText, durationText)
}
