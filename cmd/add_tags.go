package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/tbernacchi/datadog-monitor-manager/internal/datadog"
	"github.com/spf13/cobra"
)

var addTagsCmd = &cobra.Command{
	Use:   "add-tags",
	Short: "Add tags to monitors",
	Long:  `Add tags to a single monitor or multiple monitors matching filters`,
	RunE:  runAddTags,
}

var (
	addTagsMonitorID  int
	addTagsService    string
	addTagsEnv        string
	addTagsNamespace  string
	addTagsFilterTags string
	addTagsQuery      string
	addTagsStatus     string
	addTagsTags       []string
)

func init() {
	rootCmd.AddCommand(addTagsCmd)
	addTagsCmd.Flags().IntVar(&addTagsMonitorID, "monitor-id", 0, "Monitor ID (for single monitor)")
	addTagsCmd.Flags().StringVar(&addTagsService, "service", "", "Filter by service (for multiple monitors)")
	addTagsCmd.Flags().StringVar(&addTagsEnv, "env", "", "Filter by environment (for multiple monitors)")
	addTagsCmd.Flags().StringVar(&addTagsNamespace, "namespace", "", "Filter by namespace (for multiple monitors)")
	addTagsCmd.Flags().StringVar(&addTagsFilterTags, "filter-tags", "", "Filter by tags (comma-separated, for multiple monitors)")
	addTagsCmd.Flags().StringVar(&addTagsQuery, "query", "", "Complex search query (e.g., service:(service1 OR service2))")
	addTagsCmd.Flags().StringVar(&addTagsStatus, "status", "", "Filter by monitor state (e.g., No Data, Alert, Warn, OK) when updating multiple monitors")
	addTagsCmd.Flags().StringArrayVar(&addTagsTags, "tag", []string{}, "Tags to add (required, can be used multiple times)")
	addTagsCmd.MarkFlagRequired("tag")
}

func runAddTags(cmd *cobra.Command, args []string) error {
	if len(addTagsTags) == 0 {
		return fmt.Errorf("at least one --tag is required")
	}

	// Validate: either monitor-id or filters must be provided
	if addTagsMonitorID == 0 && addTagsService == "" && addTagsEnv == "" && addTagsNamespace == "" && addTagsFilterTags == "" && addTagsQuery == "" {
		return fmt.Errorf("either --monitor-id or filter flags (--service, --env, --namespace, --filter-tags, --query) must be provided")
	}

	// Cannot use both monitor-id and filters
	if addTagsMonitorID > 0 && (addTagsService != "" || addTagsEnv != "" || addTagsNamespace != "" || addTagsFilterTags != "" || addTagsQuery != "" || addTagsStatus != "") {
		return fmt.Errorf("cannot use --monitor-id together with filter flags")
	}
	
	// Cannot use --query together with other filter flags
	if addTagsQuery != "" && (addTagsService != "" || addTagsEnv != "" || addTagsNamespace != "" || addTagsFilterTags != "") {
		return fmt.Errorf("cannot use --query together with other filter flags (--service, --env, --namespace, --filter-tags)")
	}

	client, err := datadog.NewClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error: %v\n", err)
		return err
	}

	if addTagsMonitorID > 0 {
		// Single monitor
		updated, err := client.AddTagsToMonitor(addTagsMonitorID, addTagsTags)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Error adding tags: %v\n", err)
			return err
		}

		fmt.Printf("✅ Tags added to monitor %d\n", addTagsMonitorID)
		fmt.Printf("Monitor: %s\n", updated.Name)
		fmt.Printf("Tags: %s\n", strings.Join(updated.Tags, ", "))
	} else if addTagsQuery != "" {
		// Use query to find monitors
		fmt.Println("\n🔍 Finding monitors with query:")
		fmt.Printf("🔎 Query: %s\n", addTagsQuery)
		if addTagsStatus != "" {
			fmt.Printf("🚦 Status: %s\n", addTagsStatus)
		}
		fmt.Println(strings.Repeat("=", 80))
		
		monitors, err := client.ListMonitors(nil, addTagsQuery)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Error listing monitors: %v\n", err)
			return err
		}

		if addTagsStatus != "" {
			monitors = filterMonitorsByState(monitors, addTagsStatus)
		}
		
		if len(monitors) == 0 {
			fmt.Println("ℹ️  No monitors found matching the specified query/status")
			return nil
		}
		
		fmt.Printf("📊 Found %d monitor(s) matching the query\n\n", len(monitors))
		
		// Add tags to each monitor
		var results []map[string]interface{}
		for _, monitor := range monitors {
			updated, err := client.AddTagsToMonitor(monitor.ID, addTagsTags)
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

		fmt.Printf("📊 Results:\n")
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
		if addTagsService != "" {
			fmt.Printf("📦 Service: %s\n", addTagsService)
		}
		if addTagsEnv != "" {
			fmt.Printf("🌍 Environment: %s\n", addTagsEnv)
		}
		if addTagsNamespace != "" {
			fmt.Printf("🏷️  Namespace: %s\n", addTagsNamespace)
		}
		if addTagsStatus != "" {
			fmt.Printf("🚦 Status: %s\n", addTagsStatus)
		}

		var filterTags []string
		if addTagsFilterTags != "" {
			filterTags = strings.Split(addTagsFilterTags, ",")
			for i := range filterTags {
				filterTags[i] = strings.TrimSpace(filterTags[i])
			}
			if len(filterTags) > 0 {
				fmt.Printf("🏷️  Filter Tags: %s\n", strings.Join(filterTags, ", "))
			}
		}
		fmt.Println(strings.Repeat("=", 80))

		var results []map[string]interface{}
		if addTagsStatus == "" {
			// Keep existing behavior (more efficient) when status filter is not requested
			results, err = client.AddTagsToMonitors(addTagsService, addTagsEnv, addTagsNamespace, filterTags, addTagsTags)
			if err != nil {
				fmt.Fprintf(os.Stderr, "❌ Error adding tags: %v\n", err)
				return err
			}
		} else {
			// When filtering by status, we need to list and filter locally
			monitors, err := client.ListMonitors(filterTags, "")
			if err != nil {
				fmt.Fprintf(os.Stderr, "❌ Error listing monitors: %v\n", err)
				return err
			}

			monitors = filterMonitorsByServiceEnvNamespace(monitors, addTagsService, addTagsEnv, addTagsNamespace)
			monitors = filterMonitorsByState(monitors, addTagsStatus)

			if len(monitors) == 0 {
				fmt.Println("ℹ️  No monitors found matching the specified filters/status")
				return nil
			}

			for _, monitor := range monitors {
				updated, err := client.AddTagsToMonitor(monitor.ID, addTagsTags)
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

func canonicalMonitorState(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "-", " ")
	s = strings.ReplaceAll(s, "_", " ")
	// collapse repeated spaces
	for strings.Contains(s, "  ") {
		s = strings.ReplaceAll(s, "  ", " ")
	}
	return strings.ToLower(s)
}

func filterMonitorsByState(monitors []datadog.Monitor, desiredState string) []datadog.Monitor {
	want := canonicalMonitorState(desiredState)
	if want == "" {
		return monitors
	}
	var filtered []datadog.Monitor
	for _, m := range monitors {
		if canonicalMonitorState(m.OverallState) == want {
			filtered = append(filtered, m)
		}
	}
	return filtered
}

func filterMonitorsByServiceEnvNamespace(monitors []datadog.Monitor, service, env, namespace string) []datadog.Monitor {
	if service == "" && env == "" && namespace == "" {
		return monitors
	}

	var filtered []datadog.Monitor
	for _, monitor := range monitors {
		matches := true
		if service != "" && !hasExactTag(monitor.Tags, fmt.Sprintf("service:%s", service)) {
			matches = false
		}
		if env != "" && !hasExactTag(monitor.Tags, fmt.Sprintf("env:%s", env)) {
			matches = false
		}
		if namespace != "" && !hasExactTag(monitor.Tags, fmt.Sprintf("namespace:%s", namespace)) {
			matches = false
		}
		if matches {
			filtered = append(filtered, monitor)
		}
	}
	return filtered
}

func hasExactTag(tags []string, want string) bool {
	for _, t := range tags {
		if t == want {
			return true
		}
	}
	return false
}
