package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tbernacchi/datadog-monitor-manager/internal/datadog"
	"github.com/spf13/cobra"
)

var templateCmd = &cobra.Command{
	Use:   "template",
	Short: "Apply monitor templates from JSON files",
	Long:  `Apply monitor templates from JSON files`,
	RunE:  runTemplate,
}

var (
	templateService    string
	templateEnv        string
	templateNamespace  string
	templateFile       string
	templateDir        string
	templateNoUpsert   bool
	templateTags       []string
)

func init() {
	rootCmd.AddCommand(templateCmd)
	templateCmd.Flags().StringVar(&templateService, "service", "", "Service name (required)")
	templateCmd.MarkFlagRequired("service")
	templateCmd.Flags().StringVar(&templateEnv, "env", "", "Environment: dev, hml, prd, corp (required)")
	templateCmd.MarkFlagRequired("env")
	templateCmd.Flags().StringVar(&templateNamespace, "namespace", "", "Kubernetes namespace (required)")
	templateCmd.MarkFlagRequired("namespace")
	templateCmd.Flags().StringVarP(&templateFile, "file", "f", "", "Path to JSON template file")
	templateCmd.Flags().StringVar(&templateDir, "template-dir", "templates", "Directory containing JSON templates (default: templates/)")
	templateCmd.Flags().BoolVar(&templateNoUpsert, "no-upsert", false, "Only create new monitors (fail if exists). Default is to update existing monitors.")
	templateCmd.Flags().StringArrayVar(&templateTags, "tag", []string{}, "Additional tags to add to monitors (can be used multiple times)")
}

func runTemplate(cmd *cobra.Command, args []string) error {
	client, err := datadog.NewClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error: %v\n", err)
		return err
	}

	service := templateService
	env := templateEnv
	namespace := templateNamespace

	// Validate env
	validEnvs := map[string]bool{"dev": true, "hml": true, "prd": true, "corp": true}
	if !validEnvs[env] {
		return fmt.Errorf("invalid environment: %s (must be dev, hml, prd, or corp)", env)
	}

	fmt.Println("\n🚀 Applying monitor templates for:")
	fmt.Printf("📦 Service: %s\n", service)
	fmt.Printf("🌍 Environment: %s\n", env)
	fmt.Printf("🏷️  Namespace: %s\n", namespace)
	fmt.Println(strings.Repeat("=", 80))

	upsert := !templateNoUpsert

	if templateFile != "" {
		// Apply template file
		results, err := client.ApplyTemplate(templateFile, service, env, namespace, upsert, templateTags)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Error applying template: %v\n", err)
			return err
		}

		if len(results) > 0 {
			createdCount := 0
			updatedCount := 0
			for _, result := range results {
				if wasCreated, ok := result["was_created"].(bool); ok && wasCreated {
					createdCount++
				} else {
					updatedCount++
				}
			}

			if createdCount > 0 && updatedCount > 0 {
				fmt.Printf("✅ Applied %d monitors: %d created, %d updated\n", len(results), createdCount, updatedCount)
			} else if createdCount > 0 {
				fmt.Printf("✅ Created %d new monitors\n", createdCount)
			} else {
				fmt.Printf("✅ Updated %d existing monitors\n", updatedCount)
			}

			for _, result := range results {
				templateName, _ := result["template_name"].(string)
				monitorID, _ := result["id"].(int)
				wasCreated, _ := result["was_created"].(bool)
				action := "🆕 Created"
				if !wasCreated {
					action = "🔄 Updated"
				}
				fmt.Printf("   %s %s: Monitor ID %d\n", action, templateName, monitorID)
			}
		} else {
			fmt.Printf("❌ Failed to apply template: %s\n", templateFile)
		}
	} else {
		// Apply all templates from directory
		if _, err := os.Stat(templateDir); os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "❌ Template directory not found: %s\n", templateDir)
			fmt.Fprintf(os.Stderr, "💡 Create the directory and add JSON template files:\n")
			fmt.Fprintf(os.Stderr, "   mkdir %s\n", templateDir)
			fmt.Fprintf(os.Stderr, "   # Export templates from Datadog UI and save as .json files\n")
			return err
		}

		// Find all JSON files in template directory
		matches, err := filepath.Glob(filepath.Join(templateDir, "*.json"))
		if err != nil {
			return err
		}

		if len(matches) == 0 {
			fmt.Fprintf(os.Stderr, "❌ No JSON template files found in: %s\n", templateDir)
			fmt.Fprintf(os.Stderr, "💡 Add JSON template files exported from Datadog UI\n")
			return fmt.Errorf("no template files found")
		}

		fmt.Printf("📁 Found %d template files in %s\n", len(matches), templateDir)

		totalCreated := 0
		totalUpdated := 0

		for _, templateFile := range matches {
			templateName := filepath.Base(templateFile)
			fmt.Printf("\n📄 Applying template: %s\n", templateName)

			results, err := client.ApplyTemplate(templateFile, service, env, namespace, upsert, templateTags)
			if err != nil {
				fmt.Fprintf(os.Stderr, "   ❌ Failed to apply template: %v\n", err)
				continue
			}

			if len(results) > 0 {
				for _, result := range results {
					monitorName, _ := result["template_name"].(string)
					monitorID, _ := result["id"].(int)
					wasCreated, _ := result["was_created"].(bool)
					action := "🆕 Created"
					if !wasCreated {
						action = "🔄 Updated"
					}
					fmt.Printf("   %s %s: Monitor ID %d\n", action, monitorName, monitorID)

					if wasCreated {
						totalCreated++
					} else {
						totalUpdated++
					}
				}
			} else {
				fmt.Println("   ❌ Failed to apply template")
			}
		}

		fmt.Printf("\n✅ Successfully applied monitors:\n")
		fmt.Printf("   🆕 Created: %d\n", totalCreated)
		fmt.Printf("   🔄 Updated: %d\n", totalUpdated)
		fmt.Printf("   📊 Total: %d\n", totalCreated+totalUpdated)
	}

	return nil
}

