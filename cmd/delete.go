package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/tbernacchi/datadog-monitor-manager/internal/datadog"
)

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a monitor",
	Long:  `Delete a single monitor by ID`,
	RunE:  runDelete,
}

var (
	deleteMonitorID int
	deleteConfirm   bool
)

func init() {
	rootCmd.AddCommand(deleteCmd)
	deleteCmd.Flags().IntVar(&deleteMonitorID, "monitor-id", 0, "Monitor ID (required)")
	deleteCmd.MarkFlagRequired("monitor-id")
	deleteCmd.Flags().BoolVar(&deleteConfirm, "confirm", false, "Confirm deletion")
}

func runDelete(cmd *cobra.Command, args []string) error {
	if !deleteConfirm {
		fmt.Fprintf(os.Stderr, "❌ Please use --confirm to confirm deletion\n")
		return fmt.Errorf("confirmation required")
	}

	client, err := datadog.NewClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error: %v\n", err)
		return err
	}

	err = client.DeleteMonitor(deleteMonitorID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error deleting monitor: %v\n", err)
		return err
	}

	fmt.Printf("✅ Monitor %d deleted successfully!\n", deleteMonitorID)
	return nil
}
