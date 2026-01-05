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
	removeTagsTags       []string
)

func init() {
	rootCmd.AddCommand(removeTagsCmd)
	removeTagsCmd.Flags().IntVar(&removeTagsMonitorID, "monitor-id", 0, "Monitor ID (for single monitor)")
	removeTagsCmd.Flags().StringVar(&removeTagsService, "service", "", "Filter by service (for multiple monitors)")
	removeTagsCmd.Flags().StringVar(&removeTagsEnv, "env", "", "Filter by environment (for multiple monitors)")
	removeTagsCmd.Flags().StringVar(&removeTagsNamespace, "namespace", "", "Filter by namespace (for multiple monitors)")
	removeTagsCmd.Flags().StringVar(&removeTagsFilterTags, "filter-tags", "", "Filter by tags (comma-separated, for multiple monitors)")
	removeTagsCmd.Flags().StringArrayVar(&removeTagsTags, "tag", []string{}, "Tags to remove (required, can be used multiple times)")
	removeTagsCmd.MarkFlagRequired("tag")
}

func runRemoveTags(cmd *cobra.Command, args []string) error {
	if len(removeTagsTags) == 0 {
		return fmt.Errorf("at least one --tag is required")
	}

	// Validate: either monitor-id or filters must be provided
	if removeTagsMonitorID == 0 && removeTagsService == "" && removeTagsEnv == "" && removeTagsNamespace == "" && removeTagsFilterTags == "" {
		return fmt.Errorf("either --monitor-id or filter flags (--service, --env, --namespace, --filter-tags) must be provided")
	}

	// Cannot use both monitor-id and filters
	if removeTagsMonitorID > 0 && (removeTagsService != "" || removeTagsEnv != "" || removeTagsNamespace != "" || removeTagsFilterTags != "") {
		return fmt.Errorf("cannot use --monitor-id together with filter flags")
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

		results, err := client.RemoveTagsFromMonitors(removeTagsService, removeTagsEnv, removeTagsNamespace, filterTags, removeTagsTags)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Error removing tags: %v\n", err)
			return err
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
