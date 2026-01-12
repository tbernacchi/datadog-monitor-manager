package cmd

import (
	"fmt"
	"strings"

	"github.com/tbernacchi/datadog-monitor-manager/internal/datadog"
)

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

func filterMonitorsByServices(monitors []datadog.Monitor, services []string) []datadog.Monitor {
	if len(services) == 0 {
		return monitors
	}

	var filtered []datadog.Monitor
	for _, monitor := range monitors {
		for _, service := range services {
			if hasExactTag(monitor.Tags, fmt.Sprintf("service:%s", service)) {
				filtered = append(filtered, monitor)
				break // Found a match, move to next monitor
			}
		}
	}
	return filtered
}
