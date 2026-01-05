package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "datadog-monitor-manager",
	Short: "Datadog Kubernetes Monitor Manager CLI",
	Long: `Datadog Monitor Manager - Kubernetes Monitors Only
Creates Kubernetes-specific monitors/alerts in Datadog via API
Pipeline-ready with auto-detection capabilities

Version: 1.0.0`,
	Version: "1.0.0",
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.SetVersionTemplate("{{.Version}}\n")
	cobra.OnInitialize()
}

