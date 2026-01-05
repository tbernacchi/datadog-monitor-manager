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
	addTagsTags       []string
)

func init() {
	rootCmd.AddCommand(addTagsCmd)
	addTagsCmd.Flags().IntVar(&addTagsMonitorID, "monitor-id", 0, "Monitor ID (for single monitor)")
	addTagsCmd.Flags().StringVar(&addTagsService, "service", "", "Filter by service (for multiple monitors)")
	addTagsCmd.Flags().StringVar(&addTagsEnv, "env", "", "Filter by environment (for multiple monitors)")
	addTagsCmd.Flags().StringVar(&addTagsNamespace, "namespace", "", "Filter by namespace (for multiple monitors)")
	addTagsCmd.Flags().StringVar(&addTagsFilterTags, "filter-tags", "", "Filter by tags (comma-separated, for multiple monitors)")
	addTagsCmd.Flags().StringArrayVar(&addTagsTags, "tag", []string{}, "Tags to add (required, can be used multiple times)")
	addTagsCmd.MarkFlagRequired("tag")
}

func runAddTags(cmd *cobra.Command, args []string) error {
	if len(addTagsTags) == 0 {
		return fmt.Errorf("at least one --tag is required")
	}

	// Validate: either monitor-id or filters must be provided
	if addTagsMonitorID == 0 && addTagsService == "" && addTagsEnv == "" && addTagsNamespace == "" && addTagsFilterTags == "" {
		return fmt.Errorf("either --monitor-id or filter flags (--service, --env, --namespace, --filter-tags) must be provided")
	}

	// Cannot use both monitor-id and filters
	if addTagsMonitorID > 0 && (addTagsService != "" || addTagsEnv != "" || addTagsNamespace != "" || addTagsFilterTags != "") {
		return fmt.Errorf("cannot use --monitor-id together with filter flags")
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

		results, err := client.AddTagsToMonitors(addTagsService, addTagsEnv, addTagsNamespace, filterTags, addTagsTags)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Error adding tags: %v\n", err)
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
