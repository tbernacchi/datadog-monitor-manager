# Datadog Monitor Manager

Datadog monitor manager for Kubernetes - Go CLI tool.

Version: 1.0.0

## Description

CLI tool to manage Datadog monitors/alerts via API. Built specifically for Kubernetes monitors, with template support and auto-detection capabilities ready for pipelines.

## Requirements

- Go 1.21 or higher
- Environment variables:
  - `DD_API_KEY` or `DATADOG_API_KEY` - Datadog API key
  - `DD_APP_KEY` or `DATADOG_APP_KEY` - Datadog Application key

## Installation

```bash
# Clone the repository and navigate to the directory
cd usable-tools/datadog-monitor-manager

# Download dependencies
go mod download

# Build
go build -o datadog-monitor-manager

# Or use Makefile
make build
```

## Configuration

Set up environment variables:

```bash
export DD_API_KEY='your-api-key'
export DD_APP_KEY='your-app-key'
```

## Usage

### List Monitors

```bash
# List all monitors
./datadog-monitor-manager list

# Filter by service
./datadog-monitor-manager list --service bff-whatsapp

# Filter by namespace
./datadog-monitor-manager list --namespace bffwhatsappapi

# Filter by environment and namespace
./datadog-monitor-manager list --env prd --namespace bffwhatsappapi

# Search by tags
./datadog-monitor-manager list --tags whatsapp

# Show only tags from all monitors
./datadog-monitor-manager list --tags-only

# Show only tags from monitors with a specific service
./datadog-monitor-manager list --service myapp --tags-only

# Show only tags from a specific monitor
./datadog-monitor-manager list --monitor-id 12345 --tags-only

# List monitors with complex query
./datadog-monitor-manager list --query "service:(service1 OR service2 OR service3)"
```

### Describe Monitor

```bash
# Show details of a monitor
./datadog-monitor-manager describe --monitor-id 12345

# Output in JSON format
./datadog-monitor-manager describe --monitor-id 12345 --json
```

### Delete Monitor

```bash
# Delete a specific monitor
./datadog-monitor-manager delete --monitor-id 12345 --confirm

# Delete all monitors matching filters (interactive confirmation)
./datadog-monitor-manager delete-all --service partners-caixa-api --env hml --namespace partners-caixa-api
```

### Apply Templates

```bash
# Apply template from a specific file
./datadog-monitor-manager template \
  --service bff-whatsapp-tbernacchi-api \
  --env corp \
  --namespace bffwhatsappapi \
  --file templates/kubernetes-monitors.json

# Apply all templates from a directory
./datadog-monitor-manager template \
  --service myapp \
  --env hml \
  --namespace myapp \
  --template-dir templates

# Only create new monitors (fail if already exists)
./datadog-monitor-manager template \
  --service myapp \
  --env hml \
  --namespace myapp \
  --file templates/kubernetes-monitors.json \
  --no-upsert

# Add additional tags
./datadog-monitor-manager template \
  --service myapp \
  --env hml \
  --namespace myapp \
  --file templates/kubernetes-monitors.json \
  --tag team:backend \
  --tag priority:high
```

### Add Tags

```bash
# Add tags to a single monitor
./datadog-monitor-manager add-tags \
  --monitor-id 12345 \
  --tag team:backend \
  --tag priority:high

# Add tags to multiple monitors matching filters
./datadog-monitor-manager add-tags \
  --service myapp \
  --env hml \
  --namespace myapp \
  --tag team:backend \
  --tag priority:high

# Add tags with additional filter tags
./datadog-monitor-manager add-tags \
  --service myapp \
  --env hml \
  --filter-tags "kubernetes,production" \
  --tag team:backend

# Add tags to monitors matching a complex query
./datadog-monitor-manager add-tags \
  --query "service:(service1 OR service2 OR service3)" \
  --tag squad:parcerias

# Add tags to monitors in a specific state (e.g., Alert/Warn/No Data/OK)
./datadog-monitor-manager add-tags \
  --query "service:(service1 OR service2 OR service3)" \
  --status "Alert" \
  --tag squad:parcerias
```

### Remove Tags

```bash
# Remove tags from a single monitor
./datadog-monitor-manager remove-tags \
  --monitor-id 12345 \
  --tag team:backend \
  --tag priority:high

# Remove tags from multiple monitors matching filters
./datadog-monitor-manager remove-tags \
  --service myapp \
  --env hml \
  --namespace myapp \
  --tag team:backend \
  --tag priority:high

# Remove tags with additional filter tags
./datadog-monitor-manager remove-tags \
  --service myapp \
  --env hml \
  --filter-tags "kubernetes,production" \
  --tag team:backend

# Remove tags from monitors matching a complex query and state
./datadog-monitor-manager remove-tags \
  --query "service:(service1 OR service2 OR service3)" \
  --status "No Data" \
  --tag squad:parcerias
```

## Project Structure

```
datadog-monitor-manager/
├── cmd/
│   ├── root.go          # Root command
│   ├── list.go          # List command
│   ├── describe.go      # Describe command
│   ├── delete.go        # Delete command
│   ├── delete_all.go    # Delete-all command
│   ├── template.go      # Template command
│   ├── add_tags.go      # Add-tags command
│   └── remove_tags.go   # Remove-tags command
├── internal/
│   └── datadog/
│       └── client.go    # Datadog API client
├── main.go              # Entry point
├── go.mod               # Dependencies
├── Makefile             # Build and tasks
└── README.md            # This file
```

## Template Format

Templates must be in JSON format. They can be:

1. **Single template:**
```json
{
  "name": "Monitor {service} - Error Rate",
  "type": "query alert",
  "query": "sum(last_5m):sum:http.requests{service:{service},env:{env}}",
  "message": "High error rate for {service}",
  "tags": ["env:{env}", "service:{service}"]
}
```

2. **Multiple templates:**
```json
{
  "templates": [
    {
      "name": "Error Rate",
      "config": {
        "name": "Monitor {service} - Error Rate",
        "type": "query alert",
        "query": "sum(last_5m):sum:http.requests{service:{service}}"
      }
    },
    {
      "name": "Latency",
      "config": {
        "name": "Monitor {service} - Latency",
        "type": "query alert",
        "query": "avg(last_5m):avg:http.request.duration{service:{service}}"
      }
    }
  ]
}
```

## Supported Placeholders

The following placeholders can be used in templates:

- `{service}` - Service name
- `{env}` - Environment (dev, hml, prd, corp)
- `{namespace}` - Kubernetes namespace

**Note:** The placeholder `by {service}` in the query is preserved literally (not replaced), as the Datadog API needs it as-is.

## Valid Environments

- `dev` - Development
- `hml` - Staging/Homologation
- `prd` - Production
- `corp` - Corporate

## Complete Examples

### Create monitors for a new service

```bash
# 1. Check which monitors already exist
./datadog-monitor-manager list --service my-new-service --env hml

# 2. Apply templates
./datadog-monitor-manager template \
  --service my-new-service \
  --env hml \
  --namespace my-new-service \
  --template-dir templates

# 3. Verify created monitors
./datadog-monitor-manager list --service my-new-service --env hml
```

### Clean up monitors for a service

```bash
# List first to see what will be deleted
./datadog-monitor-manager list --service old-service --env hml

# Delete all monitors for the service
./datadog-monitor-manager delete-all \
  --service old-service \
  --env hml \
  --namespace old-service
```

### Manage tags for monitors

```bash
# Add tags to all monitors for a service
./datadog-monitor-manager add-tags \
  --service myapp \
  --env hml \
  --tag team:backend \
  --tag priority:high

# Remove specific tags from monitors
./datadog-monitor-manager remove-tags \
  --service myapp \
  --env hml \
  --tag old-tag:value

# Add tags to a single monitor
./datadog-monitor-manager add-tags \
  --monitor-id 12345 \
  --tag team:backend
```

## Development

```bash
# Install dependencies
go mod download

# Build
make build

# Run tests (when available)
go test ./...

# Clean binaries
make clean
```

## Commands Reference

### `list`
List existing monitors with optional filters.

**Flags:**
- `--service` - Filter by service name
- `--env` - Filter by environment
- `--namespace` - Filter by namespace
- `--tags` - Search in all tags (like UI search box)
- `--query` - Complex search query (e.g., service:(service1 OR service2))
- `--tags-only` - Show only tags from monitors (one per line, sorted)
- `--monitor-id` - Get tags from a specific monitor (use with --tags-only)
- `--simple` - Simple output format (ID and name only)
- `--limit` - Limit number of monitors to show

### `describe`
Show detailed information about a specific monitor.

**Flags:**
- `--monitor-id` (required) - Monitor ID
- `--json` - Output in JSON format

### `delete`
Delete a single monitor by ID.

**Flags:**
- `--monitor-id` (required) - Monitor ID to delete
- `--confirm` (required) - Confirm deletion

### `delete-all`
Delete all monitors matching the specified filters (interactive confirmation).

**Flags:**
- `--service` - Filter by service name
- `--env` - Filter by environment
- `--namespace` - Filter by namespace
- `--tags` - Filter by tags (comma-separated)

### `template`
Apply monitor templates from JSON files.

**Flags:**
- `--service` (required) - Service name
- `--env` (required) - Environment: dev, hml, prd, corp
- `--namespace` (required) - Kubernetes namespace
- `--file` / `-f` - Path to JSON template file
- `--template-dir` - Directory containing JSON templates (default: templates/)
- `--no-upsert` - Only create new monitors (fail if exists). Default is to update existing monitors.
- `--tag` - Additional tags to add to monitors (can be used multiple times)

### `add-tags`
Add tags to a single monitor or multiple monitors matching filters.

**Flags:**
- `--monitor-id` - Monitor ID (for single monitor)
- `--service` - Filter by service (for multiple monitors)
- `--env` - Filter by environment (for multiple monitors)
- `--namespace` - Filter by namespace (for multiple monitors)
- `--filter-tags` - Filter by tags (comma-separated, for multiple monitors)
- `--query` - Complex search query (e.g., service:(service1 OR service2)) for multiple monitors
- `--status` - Filter by monitor state (e.g., No Data, Alert, Warn, OK) for multiple monitors
- `--tag` (required) - Tags to add (can be used multiple times)

**Note:** Either `--monitor-id` or filter flags (`--service`, `--env`, `--namespace`, `--filter-tags`, `--query`) must be provided. Cannot use `--query` together with other filter flags.

### `remove-tags`
Remove tags from a single monitor or multiple monitors matching filters.

**Flags:**
- `--monitor-id` - Monitor ID (for single monitor)
- `--service` - Filter by service (for multiple monitors)
- `--env` - Filter by environment (for multiple monitors)
- `--namespace` - Filter by namespace (for multiple monitors)
- `--filter-tags` - Filter by tags (comma-separated, for multiple monitors)
- `--query` - Complex search query (e.g., service:(service1 OR service2)) for multiple monitors
- `--status` - Filter by monitor state (e.g., No Data, Alert, Warn, OK) for multiple monitors
- `--tag` (required) - Tags to remove (can be used multiple times)

**Note:** Either `--monitor-id` or filter flags (`--service`, `--env`, `--namespace`, `--filter-tags`, `--query`) must be provided. Cannot use `--query` together with other filter flags.

## License

This project is part of the sre-tbernacchi-bkp repository.
