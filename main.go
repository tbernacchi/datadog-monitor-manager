package main

import (
	"os"

	"github.com/tbernacchi/datadog-monitor-manager/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
