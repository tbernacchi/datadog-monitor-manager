package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/tbernacchi/datadog-monitor-manager/internal/datadog"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list [tag]",
	Short: "List existing monitors",
	Long: `List existing monitors with optional filters.

You can pass a tag as a positional argument (e.g., "service:myapp") or use flags.
Examples:
  list                                    # List all monitors
  list service:myapp                      # List monitors with exact tag
  list --service myapp                    # List monitors with service tag
  list --tags service:myapp --limit 5    # List with tag filter and limit
  list --tags-only                        # Show only unique tags from all monitors
  list --service myapp --tags-only       # Show only tags from monitors with service tag
  list --monitor-id 12345 --tags-only    # Show only tags from monitor ID 12345`,
	RunE: runList,
}

var (
	listService   string
	listEnv       string
	listNamespace string
	listTags      string
	listSimple    bool
	listTagsOnly  bool
	listMonitorID int
	listLimit     int
)

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().StringVar(&listService, "service", "", "Filter by service")
	listCmd.Flags().StringVar(&listEnv, "env", "", "Filter by environment")
	listCmd.Flags().StringVar(&listNamespace, "namespace", "", "Filter by namespace")
	listCmd.Flags().StringVar(&listTags, "tags", "", "Search in all tags (like UI search box)")
	listCmd.Flags().BoolVar(&listSimple, "simple", false, "Simple output format (ID and name only)")
	listCmd.Flags().BoolVar(&listTagsOnly, "tags-only", false, "Show only tags from monitors")
	listCmd.Flags().IntVar(&listMonitorID, "monitor-id", 0, "Get tags from a specific monitor (use with --tags-only)")
	listCmd.Flags().IntVar(&listLimit, "limit", 0, "Limit number of monitors to show (e.g., --limit 1 for one example)")
}

func runList(cmd *cobra.Command, args []string) error {
	client, err := datadog.NewClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error: %v\n", err)
		return err
	}

	// If monitor-id is specified with tags-only, get that specific monitor
	if listMonitorID > 0 && listTagsOnly {
		monitor, err := client.GetMonitor(listMonitorID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Error getting monitor: %v\n", err)
			return err
		}

		// Print tags, one per line, sorted
		tags := make([]string, len(monitor.Tags))
		copy(tags, monitor.Tags)
		sort.Strings(tags)

		for _, tag := range tags {
			fmt.Println(tag)
		}
		return nil
	}

	// If tags flag is empty but we have positional args that look like tags, use them
	if listTags == "" && len(args) > 0 {
		for _, arg := range args {
			// If argument contains ':', treat it as a tag
			if strings.Contains(arg, ":") {
				listTags = arg
				break
			}
		}
	}

	var monitors []datadog.Monitor
	if listTags != "" {
		// If the search text looks like a tag (contains ':'), use tag filter directly
		if strings.Contains(listTags, ":") {
			// Use exact tag filter via API
			exactTag := listTags
			tags := []string{exactTag}
			monitors, err = client.ListMonitors(tags, "")
		} else {
			// Use search text for flexible search (no ':')
			monitors, err = client.ListMonitors(nil, listTags)
		}
	} else {
		var tags []string
		if listService != "" {
			tags = append(tags, fmt.Sprintf("service:%s", listService))
		}
		if listEnv != "" {
			tags = append(tags, fmt.Sprintf("env:%s", listEnv))
		}
		if listNamespace != "" {
			tags = append(tags, fmt.Sprintf("namespace:%s", listNamespace))
		}

		if len(tags) > 0 {
			monitors, err = client.ListMonitors(tags, "")
		} else {
			monitors, err = client.ListMonitors(nil, "")
		}
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error listing monitors: %v\n", err)
		return err
	}

	// Apply limit if specified
	if listLimit > 0 && len(monitors) > listLimit {
		monitors = monitors[:listLimit]
	}

	if listTagsOnly {
		// Collect all unique tags
		tagSet := make(map[string]bool)
		for _, monitor := range monitors {
			for _, tag := range monitor.Tags {
				tagSet[tag] = true
			}
		}
		
		// Convert to slice and sort for consistent output
		var tags []string
		for tag := range tagSet {
			tags = append(tags, tag)
		}
		sort.Strings(tags)
		
		// Print tags, one per line
		for _, tag := range tags {
			fmt.Println(tag)
		}
		return nil
	}

	if listSimple {
		// Simple format: just ID and name
		for _, monitor := range monitors {
			fmt.Printf("%d\t%s\n", monitor.ID, monitor.Name)
		}
		return nil
	}

	totalCount := len(monitors)
	if listLimit > 0 {
		fmt.Printf("\n📊 Showing %d monitor(s) (limited):\n", totalCount)
	} else {
		fmt.Printf("\n📊 Found %d monitors:\n", totalCount)
	}
	if totalCount == 0 {
		return nil
	}
	fmt.Println(strings.Repeat("-", 80))

	for _, monitor := range monitors {
		status := "🟢 Enabled"
		if monitor.OverallState == "muted" {
			status = "🔴 Disabled"
		}
		fmt.Printf("\nID: %d\n", monitor.ID)
		fmt.Printf("Name: %s\n", monitor.Name)
		fmt.Printf("Type: %s\n", monitor.Type)
		fmt.Printf("Status: %s\n", status)
		if len(monitor.Tags) > 0 {
			fmt.Printf("Tags: %s\n", strings.Join(monitor.Tags, ", "))
		} else {
			fmt.Printf("Tags: (none)\n")
		}
	}

	return nil
}
