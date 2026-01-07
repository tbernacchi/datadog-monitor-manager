package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/tbernacchi/datadog-monitor-manager/internal/datadog"
	"github.com/spf13/cobra"
)

var removeTagsCmd = &cobra.Command{
	Use:   "remove-tags",
	Short: "Remove tags from monitors",
	Long:  `Remove tags from a single monitor or multiple monitors matching filters`,
	RunE:  runRemoveTags,
}

var (
	removeTagsMonitorID  int
	removeTagsService    string
	removeTagsEnv        string
	removeTagsNamespace  string
	removeTagsFilterTags string
	removeTagsQuery      string
	removeTagsStatus     string
	removeTagsTags       []string
)

func init() {
	rootCmd.AddCommand(removeTagsCmd)
	removeTagsCmd.Flags().IntVar(&removeTagsMonitorID, "monitor-id", 0, "Monitor ID (for single monitor)")
	removeTagsCmd.Flags().StringVar(&removeTagsService, "service", "", "Filter by service (for multiple monitors)")
	removeTagsCmd.Flags().StringVar(&removeTagsEnv, "env", "", "Filter by environment (for multiple monitors)")
	removeTagsCmd.Flags().StringVar(&removeTagsNamespace, "namespace", "", "Filter by namespace (for multiple monitors)")
	removeTagsCmd.Flags().StringVar(&removeTagsFilterTags, "filter-tags", "", "Filter by tags (comma-separated, for multiple monitors)")
	removeTagsCmd.Flags().StringVar(&removeTagsQuery, "query", "", "Complex search query (e.g., service:(service1 OR service2))")
	removeTagsCmd.Flags().StringVar(&removeTagsStatus, "status", "", "Filter by monitor state (e.g., No Data, Alert, Warn, OK) when updating multiple monitors")
	removeTagsCmd.Flags().StringArrayVar(&removeTagsTags, "tag", []string{}, "Tags to remove (required, can be used multiple times)")
	removeTagsCmd.MarkFlagRequired("tag")
}

func runRemoveTags(cmd *cobra.Command, args []string) error {
	if len(removeTagsTags) == 0 {
		return fmt.Errorf("at least one --tag is required")
	}

	// Validate: either monitor-id or filters must be provided
	if removeTagsMonitorID == 0 && removeTagsService == "" && removeTagsEnv == "" && removeTagsNamespace == "" && removeTagsFilterTags == "" && removeTagsQuery == "" {
		return fmt.Errorf("either --monitor-id or filter flags (--service, --env, --namespace, --filter-tags, --query) must be provided")
	}

	// Cannot use both monitor-id and filters
	if removeTagsMonitorID > 0 && (removeTagsService != "" || removeTagsEnv != "" || removeTagsNamespace != "" || removeTagsFilterTags != "" || removeTagsQuery != "" || removeTagsStatus != "") {
		return fmt.Errorf("cannot use --monitor-id together with filter flags")
	}

	// Cannot use --query together with other filter flags
	if removeTagsQuery != "" && (removeTagsService != "" || removeTagsEnv != "" || removeTagsNamespace != "" || removeTagsFilterTags != "") {
		return fmt.Errorf("cannot use --query together with other filter flags (--service, --env, --namespace, --filter-tags)")
	}

	client, err := datadog.NewClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error: %v\n", err)
		return err
	}

	if removeTagsMonitorID > 0 {
		// Single monitor
		updated, err := client.RemoveTagsFromMonitor(removeTagsMonitorID, removeTagsTags)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Error removing tags: %v\n", err)
			return err
		}

		fmt.Printf("✅ Tags removed from monitor %d\n", removeTagsMonitorID)
		fmt.Printf("Monitor: %s\n", updated.Name)
		fmt.Printf("Tags: %s\n", strings.Join(updated.Tags, ", "))
	} else if removeTagsQuery != "" {
		// Use query to find monitors
		fmt.Println("\n🔍 Finding monitors with query:")
		fmt.Printf("🔎 Query: %s\n", removeTagsQuery)
		if removeTagsStatus != "" {
			fmt.Printf("🚦 Status: %s\n", removeTagsStatus)
		}
		fmt.Println(strings.Repeat("=", 80))

		monitors, err := client.ListMonitors(nil, removeTagsQuery)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Error listing monitors: %v\n", err)
			return err
		}

		if removeTagsStatus != "" {
			monitors = filterMonitorsByState(monitors, removeTagsStatus)
		}

		if len(monitors) == 0 {
			fmt.Println("ℹ️  No monitors found matching the specified query/status")
			return nil
		}

		fmt.Printf("📊 Found %d monitor(s) matching the query\n\n", len(monitors))

		// Remove tags from each monitor
		var results []map[string]interface{}
		for _, monitor := range monitors {
			updated, err := client.RemoveTagsFromMonitor(monitor.ID, removeTagsTags)
			if err != nil {
				results = append(results, map[string]interface{}{
					"id":     monitor.ID,
					"name":   monitor.Name,
					"status": fmt.Sprintf("failed: %v", err),
				})
			} else {
				results = append(results, map[string]interface{}{
					"id":     updated.ID,
					"name":   updated.Name,
					"status": "updated",
					"tags":   updated.Tags,
				})
			}
		}

		var successful []map[string]interface{}
		var failed []map[string]interface{}

		for _, result := range results {
			if status, ok := result["status"].(string); ok && status == "updated" {
				successful = append(successful, result)
			} else {
				failed = append(failed, result)
			}
		}

		fmt.Printf("\n📊 Results:\n")
		fmt.Printf("✅ Successfully updated: %d\n", len(successful))
		fmt.Printf("❌ Failed: %d\n", len(failed))

		if len(successful) > 0 {
			fmt.Println("\n✅ Successfully updated monitors:")
			for _, result := range successful {
				id, _ := result["id"].(int)
				name, _ := result["name"].(string)
				var tags []string
				if tagsInterface, ok := result["tags"].([]interface{}); ok {
					for _, tag := range tagsInterface {
						if tagStr, ok := tag.(string); ok {
							tags = append(tags, tagStr)
						}
					}
				} else if tagsStr, ok := result["tags"].([]string); ok {
					tags = tagsStr
				}
				fmt.Printf("   ✅ ID %d: %s\n", id, name)
				if len(tags) > 0 {
					fmt.Printf("      Tags: %s\n", strings.Join(tags, ", "))
				}
			}
		}

		if len(failed) > 0 {
			fmt.Println("\n❌ Failed to update monitors:")
			for _, result := range failed {
				id, _ := result["id"].(int)
				name, _ := result["name"].(string)
				status, _ := result["status"].(string)
				fmt.Printf("   ⚠️  ID %d: %s - %s\n", id, name, status)
			}
		}
	} else {
		// Multiple monitors
		fmt.Println("\n🔍 Finding monitors to update with filters:")
		if removeTagsService != "" {
			fmt.Printf("📦 Service: %s\n", removeTagsService)
		}
		if removeTagsEnv != "" {
			fmt.Printf("🌍 Environment: %s\n", removeTagsEnv)
		}
		if removeTagsNamespace != "" {
			fmt.Printf("🏷️  Namespace: %s\n", removeTagsNamespace)
		}
		if removeTagsStatus != "" {
			fmt.Printf("🚦 Status: %s\n", removeTagsStatus)
		}

		var filterTags []string
		if removeTagsFilterTags != "" {
			filterTags = strings.Split(removeTagsFilterTags, ",")
			for i := range filterTags {
				filterTags[i] = strings.TrimSpace(filterTags[i])
			}
			if len(filterTags) > 0 {
				fmt.Printf("🏷️  Filter Tags: %s\n", strings.Join(filterTags, ", "))
			}
		}
		fmt.Println(strings.Repeat("=", 80))

		var results []map[string]interface{}
		if removeTagsStatus == "" {
			// Keep existing behavior (more efficient) when status filter is not requested
			results, err = client.RemoveTagsFromMonitors(removeTagsService, removeTagsEnv, removeTagsNamespace, filterTags, removeTagsTags)
			if err != nil {
				fmt.Fprintf(os.Stderr, "❌ Error removing tags: %v\n", err)
				return err
			}
		} else {
			// When filtering by status, we need to list and filter locally
			monitors, err := client.ListMonitors(filterTags, "")
			if err != nil {
				fmt.Fprintf(os.Stderr, "❌ Error listing monitors: %v\n", err)
				return err
			}

			monitors = filterMonitorsByServiceEnvNamespace(monitors, removeTagsService, removeTagsEnv, removeTagsNamespace)
			monitors = filterMonitorsByState(monitors, removeTagsStatus)

			if len(monitors) == 0 {
				fmt.Println("ℹ️  No monitors found matching the specified filters/status")
				return nil
			}

			for _, monitor := range monitors {
				updated, err := client.RemoveTagsFromMonitor(monitor.ID, removeTagsTags)
				if err != nil {
					results = append(results, map[string]interface{}{
						"id":     monitor.ID,
						"name":   monitor.Name,
						"status": fmt.Sprintf("failed: %v", err),
					})
				} else {
					results = append(results, map[string]interface{}{
						"id":     updated.ID,
						"name":   updated.Name,
						"status": "updated",
						"tags":   updated.Tags,
					})
				}
			}
		}

		if len(results) == 0 {
			fmt.Println("ℹ️  No monitors found matching the specified filters")
			return nil
		}

		var successful []map[string]interface{}
		var failed []map[string]interface{}

		for _, result := range results {
			if status, ok := result["status"].(string); ok && status == "updated" {
				successful = append(successful, result)
			} else {
				failed = append(failed, result)
			}
		}

		fmt.Printf("\n📊 Results:\n")
		fmt.Printf("✅ Successfully updated: %d\n", len(successful))
		fmt.Printf("❌ Failed: %d\n", len(failed))

		if len(successful) > 0 {
			fmt.Println("\n✅ Successfully updated monitors:")
			for _, result := range successful {
				id, _ := result["id"].(int)
				name, _ := result["name"].(string)
				var tags []string
				if tagsInterface, ok := result["tags"].([]interface{}); ok {
					for _, tag := range tagsInterface {
						if tagStr, ok := tag.(string); ok {
							tags = append(tags, tagStr)
						}
					}
				} else if tagsStr, ok := result["tags"].([]string); ok {
					tags = tagsStr
				}
				fmt.Printf("   ✅ ID %d: %s\n", id, name)
				if len(tags) > 0 {
					fmt.Printf("      Tags: %s\n", strings.Join(tags, ", "))
				}
			}
		}

		if len(failed) > 0 {
			fmt.Println("\n❌ Failed to update monitors:")
			for _, result := range failed {
				id, _ := result["id"].(int)
				name, _ := result["name"].(string)
				status, _ := result["status"].(string)
				fmt.Printf("   ⚠️  ID %d: %s - %s\n", id, name, status)
			}
		}
	}

	return nil
}
