package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tbernacchi/datadog-monitor-manager/internal/datadog"
)

var deleteAllCmd = &cobra.Command{
	Use:   "delete-all",
	Short: "Delete all monitors matching filters",
	Long:  `Delete all monitors matching the specified filters (interactive confirmation)`,
	RunE:  runDeleteAll,
}

var (
	deleteAllService   string
	deleteAllEnv       string
	deleteAllNamespace string
	deleteAllTags      string
)

func init() {
	rootCmd.AddCommand(deleteAllCmd)
	deleteAllCmd.Flags().StringVar(&deleteAllService, "service", "", "Filter by service")
	deleteAllCmd.Flags().StringVar(&deleteAllEnv, "env", "", "Filter by environment")
	deleteAllCmd.Flags().StringVar(&deleteAllNamespace, "namespace", "", "Filter by namespace")
	deleteAllCmd.Flags().StringVar(&deleteAllTags, "tags", "", "Filter by tags (comma-separated)")
}

func runDeleteAll(cmd *cobra.Command, args []string) error {
	client, err := datadog.NewClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error: %v\n", err)
		return err
	}

	fmt.Println("\n🔍 Finding monitors to delete with filters:")
	if deleteAllService != "" {
		fmt.Printf("📦 Service: %s\n", deleteAllService)
	}
	if deleteAllEnv != "" {
		fmt.Printf("🌍 Environment: %s\n", deleteAllEnv)
	}
	if deleteAllNamespace != "" {
		fmt.Printf("🏷️  Namespace: %s\n", deleteAllNamespace)
	}

	var tags []string
	if deleteAllTags != "" {
		tags = strings.Split(deleteAllTags, ",")
		for i := range tags {
			tags[i] = strings.TrimSpace(tags[i])
		}
		if len(tags) > 0 {
			fmt.Printf("🏷️  Tags: %s\n", strings.Join(tags, ", "))
		}
	}
	fmt.Println(strings.Repeat("=", 80))

	// Find monitors to delete
	monitors, err := client.ListMonitors(tags, "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error listing monitors: %v\n", err)
		return err
	}

	// Filter monitors by service, env, namespace
	var filteredMonitors []datadog.Monitor
	for _, monitor := range monitors {
		matches := true
		monitorTags := monitor.Tags

		if deleteAllService != "" {
			found := false
			for _, tag := range monitorTags {
				if tag == fmt.Sprintf("service:%s", deleteAllService) {
					found = true
					break
				}
			}
			if !found {
				matches = false
			}
		}

		if deleteAllEnv != "" {
			found := false
			for _, tag := range monitorTags {
				if tag == fmt.Sprintf("env:%s", deleteAllEnv) {
					found = true
					break
				}
			}
			if !found {
				matches = false
			}
		}

		if deleteAllNamespace != "" {
			found := false
			for _, tag := range monitorTags {
				if tag == fmt.Sprintf("namespace:%s", deleteAllNamespace) {
					found = true
					break
				}
			}
			if !found {
				matches = false
			}
		}

		if matches {
			filteredMonitors = append(filteredMonitors, monitor)
		}
	}

	if len(filteredMonitors) == 0 {
		fmt.Println("ℹ️  No monitors found matching the specified filters")
		return nil
	}

	// Show monitors that will be deleted
	fmt.Printf("\n📋 Found %d monitors to delete:\n", len(filteredMonitors))
	for _, monitor := range filteredMonitors {
		status := "🟢 Enabled"
		if monitor.OverallState == "muted" {
			status = "🔴 Disabled"
		}
		fmt.Printf("   ID %d: %s (%s)\n", monitor.ID, monitor.Name, status)
	}

	// Interactive confirmation
	fmt.Printf("\n⚠️  WARNING: This will permanently delete %d monitors!\n", len(filteredMonitors))
	fmt.Print("Type 'yes' to confirm deletion: ")

	reader := bufio.NewReader(os.Stdin)
	confirm, _ := reader.ReadString('\n')
	confirm = strings.TrimSpace(strings.ToLower(confirm))

	if confirm != "yes" {
		fmt.Println("❌ Deletion cancelled")
		return nil
	}

	fmt.Println("\n🗑️  Deleting monitors...")

	// Delete monitors
	results, err := client.DeleteMonitorsByFilter(deleteAllService, deleteAllEnv, deleteAllNamespace, tags)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error deleting monitors: %v\n", err)
		return err
	}

	var successfulDeletions []map[string]interface{}
	var failedDeletions []map[string]interface{}

	for _, result := range results {
		if status, ok := result["status"].(string); ok && status == "deleted" {
			successfulDeletions = append(successfulDeletions, result)
		} else {
			failedDeletions = append(failedDeletions, result)
		}
	}

	fmt.Printf("\n📊 Deletion Results:\n")
	fmt.Printf("✅ Successfully deleted: %d\n", len(successfulDeletions))
	fmt.Printf("❌ Failed to delete: %d\n", len(failedDeletions))

	if len(successfulDeletions) > 0 {
		fmt.Println("\n✅ Successfully deleted monitors:")
		for _, monitor := range successfulDeletions {
			id, _ := monitor["id"].(int)
			name, _ := monitor["name"].(string)
			fmt.Printf("   🗑️  ID %d: %s\n", id, name)
		}
	}

	if len(failedDeletions) > 0 {
		fmt.Println("\n❌ Failed to delete monitors:")
		for _, monitor := range failedDeletions {
			id, _ := monitor["id"].(int)
			name, _ := monitor["name"].(string)
			status, _ := monitor["status"].(string)
			fmt.Printf("   ⚠️  ID %d: %s - %s\n", id, name, status)
		}
	}

	return nil
}
