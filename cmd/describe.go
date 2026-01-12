package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tbernacchi/datadog-monitor-manager/internal/datadog"
)

var describeCmd = &cobra.Command{
	Use:   "describe",
	Short: "Show detailed monitor information",
	Long:  `Show detailed information about a specific monitor`,
	RunE:  runDescribe,
}

var (
	describeMonitorID int
	describeJSON      bool
)

func init() {
	rootCmd.AddCommand(describeCmd)
	describeCmd.Flags().IntVar(&describeMonitorID, "monitor-id", 0, "Monitor ID (required)")
	describeCmd.MarkFlagRequired("monitor-id")
	describeCmd.Flags().BoolVar(&describeJSON, "json", false, "Output in JSON format")
}

func runDescribe(cmd *cobra.Command, args []string) error {
	client, err := datadog.NewClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error: %v\n", err)
		return err
	}

	monitor, err := client.GetMonitor(describeMonitorID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error getting monitor: %v\n", err)
		return err
	}

	if describeJSON {
		jsonData, err := json.MarshalIndent(monitor, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(jsonData))
		return nil
	}

	// Human-readable format
	fmt.Println("\n📊 Monitor Details:")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("ID: %d\n", monitor.ID)
	fmt.Printf("Name: %s\n", monitor.Name)
	fmt.Printf("Type: %s\n", monitor.Type)
	fmt.Printf("Query: %s\n", monitor.Query)
	fmt.Printf("Message: %s\n", monitor.Message)
	fmt.Printf("Overall State: %s\n", monitor.OverallState)

	status := "🟢 Enabled"
	if monitor.OverallState == "muted" {
		status = "🔴 Disabled"
	}
	fmt.Printf("Status: %s\n", status)

	if len(monitor.Tags) > 0 {
		fmt.Printf("Tags: %s\n", strings.Join(monitor.Tags, ", "))
	}

	if monitor.Options != nil {
		if thresholds, ok := monitor.Options["thresholds"].(map[string]interface{}); ok {
			thresholdsJSON, _ := json.Marshal(thresholds)
			fmt.Printf("Thresholds: %s\n", string(thresholdsJSON))
		}
		if notifyNoData, ok := monitor.Options["notify_no_data"].(bool); ok {
			fmt.Printf("Notify No Data: %v\n", notifyNoData)
		}
		if notifyAudit, ok := monitor.Options["notify_audit"].(bool); ok {
			fmt.Printf("Notify Audit: %v\n", notifyAudit)
		}
	}

	if monitor.CreatedAt.Int64() > 0 {
		fmt.Printf("Created: %d\n", monitor.CreatedAt.Int64())
	}
	if monitor.Modified.Int64() > 0 {
		fmt.Printf("Modified: %d\n", monitor.Modified.Int64())
	}

	fmt.Println(strings.Repeat("=", 80))
	return nil
}
