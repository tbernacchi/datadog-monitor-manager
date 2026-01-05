package datadog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
)

// Config holds Datadog API configuration
type Config struct {
	APIKey  string
	AppKey  string
	APIURL  string
	Headers map[string]string
}

// Timestamp is a custom type that can unmarshal both string and int64 timestamps
type Timestamp int64

// UnmarshalJSON implements custom unmarshaling for Timestamp
func (t *Timestamp) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as string first
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		// Try to parse as int64 string
		if val, err := strconv.ParseInt(str, 10, 64); err == nil {
			*t = Timestamp(val)
			return nil
		}
		// If it's not a numeric string, return 0
		*t = 0
		return nil
	}

	// Try to unmarshal as int64
	var val int64
	if err := json.Unmarshal(data, &val); err != nil {
		*t = 0
		return nil
	}
	*t = Timestamp(val)
	return nil
}

// Int64 returns the timestamp as int64
func (t Timestamp) Int64() int64 {
	return int64(t)
}

// Monitor represents a Datadog monitor
type Monitor struct {
	ID           int                    `json:"id,omitempty"`
	Name         string                 `json:"name"`
	Type         string                 `json:"type,omitempty"`
	Query        string                 `json:"query,omitempty"`
	Message      string                 `json:"message,omitempty"`
	Tags         []string               `json:"tags,omitempty"`
	Options      map[string]interface{} `json:"options,omitempty"`
	OverallState string                 `json:"overall_state,omitempty"`
	CreatedAt    Timestamp              `json:"created_at,omitempty"`
	Modified     Timestamp              `json:"modified,omitempty"`
}

// TemplateData represents a template structure
type TemplateData struct {
	Name   string                 `json:"name"`
	Config map[string]interface{} `json:"config"`
}

// TemplateFile represents a template file structure
type TemplateFile struct {
	Templates []TemplateData         `json:"templates,omitempty"`
	Config    map[string]interface{} `json:"-"`
}

// Client is the Datadog API client
type Client struct {
	config *Config
	client *http.Client
}

// NewClient creates a new Datadog API client
func NewClient() (*Client, error) {
	apiKey := os.Getenv("DD_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("DATADOG_API_KEY")
	}

	appKey := os.Getenv("DD_APP_KEY")
	if appKey == "" {
		appKey = os.Getenv("DATADOG_APP_KEY")
	}

	if apiKey == "" || appKey == "" {
		return nil, fmt.Errorf("DD_API_KEY and DD_APP_KEY environment variables required\n\nSet them with:\n  export DD_API_KEY='your-api-key'\n  export DD_APP_KEY='your-app-key'")
	}

	config := &Config{
		APIKey: apiKey,
		AppKey: appKey,
		APIURL: "https://api.datadoghq.com/api/v1",
		Headers: map[string]string{
			"DD-API-KEY":         apiKey,
			"DD-APPLICATION-KEY": appKey,
			"Content-Type":       "application/json",
		},
	}

	return &Client{
		config: config,
		client: &http.Client{},
	}, nil
}

// makeRequest performs an HTTP request to the Datadog API
func (c *Client) makeRequest(method, endpoint string, body interface{}) (*http.Response, error) {
	url := fmt.Sprintf("%s%s", c.config.APIURL, endpoint)

	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, err
	}

	for key, value := range c.config.Headers {
		req.Header.Set(key, value)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// CreateMonitor creates a new monitor
func (c *Client) CreateMonitor(monitor *Monitor) (*Monitor, error) {
	resp, err := c.makeRequest("POST", "/monitor", monitor)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to create monitor: status %d, body: %s", resp.StatusCode, string(body))
	}

	var result Monitor
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

// UpdateMonitor updates an existing monitor
func (c *Client) UpdateMonitor(monitorID int, monitor *Monitor) (*Monitor, error) {
	endpoint := fmt.Sprintf("/monitor/%d", monitorID)
	resp, err := c.makeRequest("PUT", endpoint, monitor)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to update monitor: status %d, body: %s", resp.StatusCode, string(body))
	}

	var result Monitor
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

// FindMonitorByName finds a monitor by its exact name
func (c *Client) FindMonitorByName(name string) (*Monitor, error) {
	monitors, err := c.ListMonitors(nil, "")
	if err != nil {
		return nil, err
	}

	for _, monitor := range monitors {
		if monitor.Name == name {
			return &monitor, nil
		}
	}

	return nil, nil
}

// UpsertMonitor creates or updates a monitor
func (c *Client) UpsertMonitor(monitor *Monitor) (*Monitor, bool, error) {
	existing, err := c.FindMonitorByName(monitor.Name)
	if err != nil {
		return nil, false, err
	}

	if existing != nil {
		updated, err := c.UpdateMonitor(existing.ID, monitor)
		return updated, false, err
	}

	created, err := c.CreateMonitor(monitor)
	return created, true, err
}

// ListMonitors lists existing monitors
func (c *Client) ListMonitors(tags []string, searchText string) ([]Monitor, error) {
	endpoint := "/monitor"
	req, err := http.NewRequest("GET", c.config.APIURL+endpoint, nil)
	if err != nil {
		return nil, err
	}

	for key, value := range c.config.Headers {
		req.Header.Set(key, value)
	}

	q := req.URL.Query()
	if len(tags) > 0 {
		var tagList []string
		for _, tag := range tags {
			if strings.HasPrefix(tag, "!=") {
				tagList = append(tagList, "!"+strings.TrimPrefix(tag, "!="))
			} else {
				tagList = append(tagList, tag)
			}
		}
		q.Set("monitor_tags", strings.Join(tagList, ","))
	}
	if searchText != "" {
		q.Set("query", searchText)
	}
	req.URL.RawQuery = q.Encode()

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list monitors: status %d, body: %s", resp.StatusCode, string(body))
	}

	var monitors []Monitor
	if err := json.NewDecoder(resp.Body).Decode(&monitors); err != nil {
		return nil, err
	}

	return monitors, nil
}

// GetMonitor gets detailed monitor information
func (c *Client) GetMonitor(monitorID int) (*Monitor, error) {
	endpoint := fmt.Sprintf("/monitor/%d", monitorID)
	resp, err := c.makeRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get monitor: status %d, body: %s", resp.StatusCode, string(body))
	}

	var monitor Monitor
	if err := json.NewDecoder(resp.Body).Decode(&monitor); err != nil {
		return nil, err
	}

	return &monitor, nil
}

// DeleteMonitor deletes a monitor
func (c *Client) DeleteMonitor(monitorID int) error {
	endpoint := fmt.Sprintf("/monitor/%d", monitorID)
	resp, err := c.makeRequest("DELETE", endpoint, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete monitor: status %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

// DeleteMonitorsByFilter deletes all monitors matching the specified filters
func (c *Client) DeleteMonitorsByFilter(service, env, namespace string, tags []string) ([]map[string]interface{}, error) {
	monitors, err := c.ListMonitors(tags, "")
	if err != nil {
		return nil, err
	}

	// Filter monitors by service, env, namespace
	var filteredMonitors []Monitor
	for _, monitor := range monitors {
		matches := true
		monitorTags := monitor.Tags

		if service != "" {
			found := false
			for _, tag := range monitorTags {
				if tag == fmt.Sprintf("service:%s", service) {
					found = true
					break
				}
			}
			if !found {
				matches = false
			}
		}

		if env != "" {
			found := false
			for _, tag := range monitorTags {
				if tag == fmt.Sprintf("env:%s", env) {
					found = true
					break
				}
			}
			if !found {
				matches = false
			}
		}

		if namespace != "" {
			found := false
			for _, tag := range monitorTags {
				if tag == fmt.Sprintf("namespace:%s", namespace) {
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

	// Delete each matching monitor
	var results []map[string]interface{}
	for _, monitor := range filteredMonitors {
		err := c.DeleteMonitor(monitor.ID)
		if err != nil {
			results = append(results, map[string]interface{}{
				"id":     monitor.ID,
				"name":   monitor.Name,
				"status": fmt.Sprintf("failed: %v", err),
			})
		} else {
			results = append(results, map[string]interface{}{
				"id":     monitor.ID,
				"name":   monitor.Name,
				"status": "deleted",
			})
		}
	}

	return results, nil
}

// LoadTemplateFromJSON loads monitor templates from JSON file
func LoadTemplateFromJSON(templateFile string) ([]TemplateData, error) {
	data, err := os.ReadFile(templateFile)
	if err != nil {
		return nil, fmt.Errorf("template file not found: %s", templateFile)
	}

	var templateFileData TemplateFile
	if err := json.Unmarshal(data, &templateFileData); err != nil {
		// Try as single template
		var singleTemplate map[string]interface{}
		if err := json.Unmarshal(data, &singleTemplate); err != nil {
			return nil, fmt.Errorf("invalid JSON in template file %s: %v", templateFile, err)
		}
		return []TemplateData{
			{Name: "Single Template", Config: singleTemplate},
		}, nil
	}

	if len(templateFileData.Templates) > 0 {
		return templateFileData.Templates, nil
	}

	// If no templates array, treat the whole file as a single template
	var singleTemplate map[string]interface{}
	if err := json.Unmarshal(data, &singleTemplate); err != nil {
		return nil, fmt.Errorf("invalid JSON in template file %s: %v", templateFile, err)
	}
	return []TemplateData{
		{Name: "Single Template", Config: singleTemplate},
	}, nil
}

// CustomizeTemplate customizes a template with service-specific values
func CustomizeTemplate(template map[string]interface{}, service, env, namespace string, additionalTags []string) map[string]interface{} {
	customized := make(map[string]interface{})
	for k, v := range template {
		customized[k] = v
	}

	// Replace placeholders in name
	if name, ok := customized["name"].(string); ok {
		customized["name"] = strings.ReplaceAll(
			strings.ReplaceAll(
				strings.ReplaceAll(name, "{service}", service),
				"{env}", strings.ToUpper(env)),
			"{namespace}", namespace)
	}

	// Replace placeholders in query
	if query, ok := customized["query"].(string); ok {
		// Preserve "by {service}" literally
		query = strings.ReplaceAll(query, "by {service}", "by __SERVICE_PRESERVE__")
		query = strings.ReplaceAll(query, "{service}", service)
		query = strings.ReplaceAll(query, "__SERVICE_PRESERVE__", "{service}")
		query = strings.ReplaceAll(query, "{env}", env)
		query = strings.ReplaceAll(query, "{namespace}", namespace)
		customized["query"] = query
	}

	// Replace placeholders in message
	if message, ok := customized["message"].(string); ok {
		customized["message"] = strings.ReplaceAll(
			strings.ReplaceAll(
				strings.ReplaceAll(message, "{service}", service),
				"{env}", env),
			"{namespace}", namespace)
	}

	// Add/update tags
	var tags []string
	if existingTags, ok := customized["tags"].([]interface{}); ok {
		for _, tag := range existingTags {
			if tagStr, ok := tag.(string); ok {
				tags = append(tags, tagStr)
			}
		}
	}

	// Add service-specific tags
	serviceTags := []string{
		fmt.Sprintf("service:%s", service),
		fmt.Sprintf("env:%s", env),
		fmt.Sprintf("namespace:%s", namespace),
	}

	for _, tag := range serviceTags {
		found := false
		for _, existingTag := range tags {
			if existingTag == tag {
				found = true
				break
			}
		}
		if !found {
			tags = append(tags, tag)
		}
	}

	// Add additional tags
	for _, tag := range additionalTags {
		found := false
		for _, existingTag := range tags {
			if existingTag == tag {
				found = true
				break
			}
		}
		if !found {
			tags = append(tags, tag)
		}
	}

	customized["tags"] = tags
	return customized
}

// ApplyTemplate applies monitor templates from JSON file
func (c *Client) ApplyTemplate(templateFile, service, env, namespace string, upsert bool, additionalTags []string) ([]map[string]interface{}, error) {
	templates, err := LoadTemplateFromJSON(templateFile)
	if err != nil {
		return nil, err
	}

	var results []map[string]interface{}
	for _, templateData := range templates {
		templateName := templateData.Name
		if templateName == "" {
			templateName = "Unknown Template"
		}

		templateConfig := templateData.Config
		if templateConfig == nil {
			// Try to use the whole templateData as config
			templateBytes, _ := json.Marshal(templateData)
			json.Unmarshal(templateBytes, &templateConfig)
		}

		// Customize the template
		customizedTemplate := CustomizeTemplate(templateConfig, service, env, namespace, additionalTags)

		// Convert to Monitor
		monitorBytes, err := json.Marshal(customizedTemplate)
		if err != nil {
			return nil, err
		}

		var monitor Monitor
		if err := json.Unmarshal(monitorBytes, &monitor); err != nil {
			return nil, err
		}

		// Create or update the monitor
		var result *Monitor
		var wasCreated bool
		if upsert {
			result, wasCreated, err = c.UpsertMonitor(&monitor)
		} else {
			result, err = c.CreateMonitor(&monitor)
			wasCreated = true
		}

		if err != nil {
			return nil, fmt.Errorf("failed to apply %s: %v", templateName, err)
		}

		resultMap := map[string]interface{}{
			"template_name": templateName,
			"id":            result.ID,
			"was_created":   wasCreated,
		}
		results = append(results, resultMap)
	}

	return results, nil
}

// CheckMonitorsExist checks which monitors from template already exist
func (c *Client) CheckMonitorsExist(templateFile, service, env, namespace string) (map[string]interface{}, error) {
	templates, err := LoadTemplateFromJSON(templateFile)
	if err != nil {
		return nil, err
	}

	if len(templates) == 0 {
		return map[string]interface{}{
			"total":    0,
			"existing": []interface{}{},
			"missing":  []interface{}{},
		}, nil
	}

	var existing []map[string]interface{}
	var missing []map[string]interface{}

	for _, templateData := range templates {
		templateName := templateData.Name
		if templateName == "" {
			templateName = "Unknown Template"
		}

		templateConfig := templateData.Config
		if templateConfig == nil {
			templateBytes, _ := json.Marshal(templateData)
			json.Unmarshal(templateBytes, &templateConfig)
		}

		customizedTemplate := CustomizeTemplate(templateConfig, service, env, namespace, nil)
		monitorName, _ := customizedTemplate["name"].(string)

		existingMonitor, err := c.FindMonitorByName(monitorName)
		if err != nil {
			return nil, err
		}

		if existingMonitor != nil {
			status := "enabled"
			if existingMonitor.OverallState == "No Data" {
				status = "no_data"
			}
			existing = append(existing, map[string]interface{}{
				"template_name": templateName,
				"monitor_name":  monitorName,
				"monitor_id":    existingMonitor.ID,
				"status":        status,
			})
		} else {
			missing = append(missing, map[string]interface{}{
				"template_name": templateName,
				"monitor_name":  monitorName,
			})
		}
	}

	return map[string]interface{}{
		"total":    len(templates),
		"existing": existing,
		"missing":  missing,
	}, nil
}

// AddTagsToMonitor adds tags to a monitor
func (c *Client) AddTagsToMonitor(monitorID int, tagsToAdd []string) (*Monitor, error) {
	// Get current monitor
	monitor, err := c.GetMonitor(monitorID)
	if err != nil {
		return nil, err
	}

	// Merge tags (avoid duplicates)
	existingTags := make(map[string]bool)
	for _, tag := range monitor.Tags {
		existingTags[tag] = true
	}

	for _, tag := range tagsToAdd {
		if !existingTags[tag] {
			monitor.Tags = append(monitor.Tags, tag)
		}
	}

	// Update monitor
	updatedMonitor, err := c.UpdateMonitor(monitorID, monitor)
	if err != nil {
		return nil, err
	}

	return updatedMonitor, nil
}

// RemoveTagsFromMonitor removes tags from a monitor
func (c *Client) RemoveTagsFromMonitor(monitorID int, tagsToRemove []string) (*Monitor, error) {
	// Get current monitor
	monitor, err := c.GetMonitor(monitorID)
	if err != nil {
		return nil, err
	}

	// Create a map of tags to remove for quick lookup
	tagsToRemoveMap := make(map[string]bool)
	for _, tag := range tagsToRemove {
		tagsToRemoveMap[tag] = true
	}

	// Filter out tags to remove
	var newTags []string
	for _, tag := range monitor.Tags {
		if !tagsToRemoveMap[tag] {
			newTags = append(newTags, tag)
		}
	}

	monitor.Tags = newTags

	// Update monitor
	updatedMonitor, err := c.UpdateMonitor(monitorID, monitor)
	if err != nil {
		return nil, err
	}

	return updatedMonitor, nil
}

// AddTagsToMonitors adds tags to multiple monitors matching filters
func (c *Client) AddTagsToMonitors(service, env, namespace string, tags []string, tagsToAdd []string) ([]map[string]interface{}, error) {
	// Find monitors matching filters
	monitors, err := c.ListMonitors(tags, "")
	if err != nil {
		return nil, err
	}

	// Filter monitors by service, env, namespace
	var filteredMonitors []Monitor
	for _, monitor := range monitors {
		matches := true
		monitorTags := monitor.Tags

		if service != "" {
			found := false
			for _, tag := range monitorTags {
				if tag == fmt.Sprintf("service:%s", service) {
					found = true
					break
				}
			}
			if !found {
				matches = false
			}
		}

		if env != "" {
			found := false
			for _, tag := range monitorTags {
				if tag == fmt.Sprintf("env:%s", env) {
					found = true
					break
				}
			}
			if !found {
				matches = false
			}
		}

		if namespace != "" {
			found := false
			for _, tag := range monitorTags {
				if tag == fmt.Sprintf("namespace:%s", namespace) {
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

	// Add tags to each monitor
	var results []map[string]interface{}
	for _, monitor := range filteredMonitors {
		updated, err := c.AddTagsToMonitor(monitor.ID, tagsToAdd)
		if err != nil {
			results = append(results, map[string]interface{}{
				"id":     monitor.ID,
				"name":   monitor.Name,
				"status": fmt.Sprintf("failed: %v", err),
			})
		} else {
			results = append(results, map[string]interface{}{
				"id":     updated.ID,
				"name":   updated.Name,
				"status": "updated",
				"tags":   updated.Tags,
			})
		}
	}

	return results, nil
}

// RemoveTagsFromMonitors removes tags from multiple monitors matching filters
func (c *Client) RemoveTagsFromMonitors(service, env, namespace string, tags []string, tagsToRemove []string) ([]map[string]interface{}, error) {
	// Find monitors matching filters
	monitors, err := c.ListMonitors(tags, "")
	if err != nil {
		return nil, err
	}

	// Filter monitors by service, env, namespace
	var filteredMonitors []Monitor
	for _, monitor := range monitors {
		matches := true
		monitorTags := monitor.Tags

		if service != "" {
			found := false
			for _, tag := range monitorTags {
				if tag == fmt.Sprintf("service:%s", service) {
					found = true
					break
				}
			}
			if !found {
				matches = false
			}
		}

		if env != "" {
			found := false
			for _, tag := range monitorTags {
				if tag == fmt.Sprintf("env:%s", env) {
					found = true
					break
				}
			}
			if !found {
				matches = false
			}
		}

		if namespace != "" {
			found := false
			for _, tag := range monitorTags {
				if tag == fmt.Sprintf("namespace:%s", namespace) {
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

	// Remove tags from each monitor
	var results []map[string]interface{}
	for _, monitor := range filteredMonitors {
		updated, err := c.RemoveTagsFromMonitor(monitor.ID, tagsToRemove)
		if err != nil {
			results = append(results, map[string]interface{}{
				"id":     monitor.ID,
				"name":   monitor.Name,
				"status": fmt.Sprintf("failed: %v", err),
			})
		} else {
			results = append(results, map[string]interface{}{
				"id":     updated.ID,
				"name":   updated.Name,
				"status": "updated",
				"tags":   updated.Tags,
			})
		}
	}

	return results, nil
}
