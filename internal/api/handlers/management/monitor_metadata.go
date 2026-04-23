package management

import (
	"os"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
)

type monitorYAMLRecord map[string]any

func asMonitorYAMLRecord(value any) monitorYAMLRecord {
	record, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	return record
}

func monitorYAMLString(record monitorYAMLRecord, keys ...string) string {
	if record == nil {
		return ""
	}
	for _, key := range keys {
		value, ok := record[key]
		if !ok {
			continue
		}
		if text, ok := value.(string); ok {
			trimmed := strings.TrimSpace(text)
			if trimmed != "" {
				return trimmed
			}
		}
	}
	return ""
}

func monitorYAMLSlice(record monitorYAMLRecord, keys ...string) []any {
	if record == nil {
		return nil
	}
	for _, key := range keys {
		value, ok := record[key]
		if !ok {
			continue
		}
		if items, ok := value.([]any); ok {
			return items
		}
	}
	return nil
}

func collectMonitorAPIKeyNames(entries []any) map[string]string {
	if len(entries) == 0 {
		return nil
	}

	names := make(map[string]string)
	for _, entry := range entries {
		record := asMonitorYAMLRecord(entry)
		if record == nil {
			continue
		}
		apiKey := monitorYAMLString(record, "api-key", "apiKey", "key", "Key")
		name := monitorYAMLString(record, "name")
		if apiKey == "" || name == "" {
			continue
		}
		names[apiKey] = name
	}
	if len(names) == 0 {
		return nil
	}
	return names
}

func parseMonitorAPIKeyNameMap(data []byte) map[string]string {
	if len(data) == 0 {
		return nil
	}

	var root map[string]any
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil
	}

	topLevelEntries := monitorYAMLSlice(root, "api-keys")
	authBlock := asMonitorYAMLRecord(root["auth"])
	providers := asMonitorYAMLRecord(authBlock["providers"])
	configAPIKeyProvider := asMonitorYAMLRecord(providers["config-api-key"])
	if configAPIKeyProvider != nil {
		providerEntries := monitorYAMLSlice(configAPIKeyProvider, "api-key-entries", "api-keys")
		if names := collectMonitorAPIKeyNames(providerEntries); len(names) > 0 {
			return names
		}
	}

	return collectMonitorAPIKeyNames(topLevelEntries)
}

func (h *Handler) monitorAPIKeyNameMap() map[string]string {
	if h == nil {
		return nil
	}
	configFilePath := strings.TrimSpace(h.configFilePath)
	if configFilePath == "" {
		return nil
	}
	data, err := os.ReadFile(configFilePath)
	if err != nil {
		return nil
	}
	return parseMonitorAPIKeyNameMap(data)
}

func lookupMonitorAPIKeysByName(query string, nameMap map[string]string) []string {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" || len(nameMap) == 0 {
		return nil
	}

	matches := make([]string, 0)
	for apiKey, name := range nameMap {
		if strings.Contains(strings.ToLower(strings.TrimSpace(name)), query) {
			matches = append(matches, apiKey)
		}
	}
	if len(matches) == 0 {
		return nil
	}
	sort.Strings(matches)
	return matches
}

func (h *Handler) buildMonitorRecordFilter(c *gin.Context, start, end *time.Time, status string) monitorRecordFilter {
	filter := monitorRecordFilter{
		APIKey:      firstQuery(c, "api", "api_key"),
		APIContains: firstQuery(c, "api_filter", "apiFilter", "api_like", "apiLike", "q"),
		Model:       firstQuery(c, "model"),
		Source:      firstQuery(c, "source", "channel"),
		Status:      status,
		Start:       start,
		End:         end,
	}
	filter.APIMatchedKeys = lookupMonitorAPIKeysByName(filter.APIContains, h.monitorAPIKeyNameMap())
	return filter
}
