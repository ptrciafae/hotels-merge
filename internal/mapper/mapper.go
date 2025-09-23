package mapper

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/tidwall/gjson"
)

// MappingEngine handles data transformation based on mapping configuration
type MappingEngine struct {
	config MappingConfig
}

// MappingConfig represents the structure of mapping.json
type MappingConfig map[string]interface{}

// FieldMapping represents a field mapping with source paths and actions
type FieldMapping struct {
	SourcePaths map[string]interface{} // key: source_1, source_2, source_3 -> value: jsonpath or template
	Actions     []string               // actions to apply
}

// SourceData holds data from all sources
type SourceData map[string]json.RawMessage

// NewMappingEngine creates a new mapping engine
func NewMappingEngine(mappingJSON []byte) (*MappingEngine, error) {
	var config MappingConfig
	if err := json.Unmarshal(mappingJSON, &config); err != nil {
		return nil, fmt.Errorf("failed to parse mapping config: %w", err)
	}

	engine := &MappingEngine{
		config: config,
	}

	return engine, nil
}

// Transform applies the mapping to source data
func (m *MappingEngine) Transform(sources SourceData) (json.RawMessage, error) {
	result := make(map[string]interface{})

	err := m.processMapping("", m.config, sources, result)
	if err != nil {
		return nil, fmt.Errorf("transformation failed: %w", err)
	}

	output, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	return json.RawMessage(output), nil
}

// processMapping recursively processes the mapping configuration
func (me *MappingEngine) processMapping(currentPath string, config interface{}, sources SourceData, result map[string]interface{}) error {

	switch v := config.(type) {
	case map[string]interface{}:
		// Check if this is a leaf node with source mappings
		if me.isLeafMapping(v) {
			value, err := me.processLeafMapping(v, sources)
			if err != nil {
				return err
			}
			me.setNestedValue(result, currentPath, value)
		} else {
			// Recursive processing for nested objects
			for key, value := range v {
				newPath := key
				if currentPath != "" {
					newPath = currentPath + "." + key
				}
				err := me.processMapping(newPath, value, sources, result)
				if err != nil {
					return err
				}
			}
		}
	case MappingConfig: // unfortunately golang doesn't support type aliasing in type switches
		return me.processMapping(currentPath, map[string]interface{}(v), sources, result)
	}
	return nil
}

// isLeafMapping checks if a mapping object contains source definitions
func (me *MappingEngine) isLeafMapping(mapping map[string]interface{}) bool {
	for key := range mapping {
		if strings.HasPrefix(key, "src::") {
			return true
		}
	}
	return false
}

// processLeafMapping processes a leaf mapping with source paths
func (me *MappingEngine) processLeafMapping(mapping map[string]interface{}, sources SourceData) (interface{}, error) {
	fieldMapping := me.parseFieldMapping(mapping)

	// extract values from all sources
	values := me.extractValuesFromSources(fieldMapping.SourcePaths, sources)

	// apply actions if specified
	if len(fieldMapping.Actions) > 0 {
		return me.applyActions(values, fieldMapping.Actions)
	}

	return me.selectBestValue(values), nil
}

// parseFieldMapping converts raw mapping to FieldMapping struct
func (me *MappingEngine) parseFieldMapping(mapping map[string]interface{}) FieldMapping {
	fieldMapping := FieldMapping{
		SourcePaths: make(map[string]interface{}),
		Actions:     []string{},
	}

	for key, value := range mapping {
		if strings.HasPrefix(key, "src::") {
			fieldMapping.SourcePaths[key] = value
		} else if key == "action" {
			if actionStr, ok := value.(string); ok {
				fieldMapping.Actions = strings.Split(actionStr, ", ")
			}
		}
	}

	return fieldMapping
}

// extractValuesFromSources extracts values from all sources using JSONPath or templates
func (me *MappingEngine) extractValuesFromSources(sourcePaths map[string]interface{}, sources SourceData) map[string]interface{} {
	values := make(map[string]interface{})

	for sourceKey, pathOrTemplate := range sourcePaths {
		sourceName := strings.TrimPrefix(sourceKey, "src::")
		if sourceData, hasSource := sources[sourceName]; hasSource {
			value := me.extractValue(sourceData, pathOrTemplate)
			if value != nil {
				values[sourceKey] = value
			}
		}
	}

	return values
}

// extractValue extracts a value using JSONPath or template
func (me *MappingEngine) extractValue(sourceData json.RawMessage, pathOrTemplate interface{}) interface{} {
	if pathOrTemplate == nil {
		return nil
	}

	pathStr, ok := pathOrTemplate.(string)
	if !ok {
		return nil
	}

	// handle template strings (e.g., "{{Address}}, {{PostalCode}}")
	if me.isTemplate(pathStr) {
		return me.processTemplate(sourceData, pathStr)
	}

	// handle regular JSONPath
	result := gjson.Get(string(sourceData), pathStr)
	if !result.Exists() {
		return nil
	}

	switch result.Type {
	case gjson.String:
		return result.String()
	case gjson.Number:
		if result.Num == float64(int64(result.Num)) {
			return int64(result.Num)
		}
		return result.Num
	case gjson.True, gjson.False:
		return result.Bool()
	case gjson.JSON:
		if result.IsArray() {
			var arr []interface{}
			for _, item := range result.Array() {
				arr = append(arr, item.Value())
			}
			return arr
		}
		var obj map[string]interface{}
		json.Unmarshal([]byte(result.Raw), &obj)
		return obj
	default:
		return result.Value()
	}
}

// isTemplate checks if a string is a template (contains {{...}})
func (me *MappingEngine) isTemplate(str string) bool {
	return strings.Contains(str, "{{") && strings.Contains(str, "}}")
}

// processTemplate processes template strings like "{{Address}}, {{PostalCode}}"
func (me *MappingEngine) processTemplate(sourceData json.RawMessage, template string) interface{} {
	// find all template variables
	re := regexp.MustCompile(`\{\{([^}]+)\}\}`)
	matches := re.FindAllStringSubmatch(template, -1)

	result := template
	for _, match := range matches {
		if len(match) >= 2 {
			fieldName := match[1]
			placeholder := match[0]

			// extract value for this field
			value := gjson.Get(string(sourceData), fieldName)
			if value.Exists() {
				result = strings.ReplaceAll(result, placeholder, value.String())
			} else {
				result = strings.ReplaceAll(result, placeholder, "")
			}
		}
	}

	// clean up extra commas and spaces
	result = strings.TrimSpace(result)
	result = regexp.MustCompile(`\s*,\s*,\s*`).ReplaceAllString(result, ", ")
	result = regexp.MustCompile(`^,\s*|,\s*$`).ReplaceAllString(result, "")

	return result
}

// selectBestValue chooses the best value from available sources
func (m *MappingEngine) selectBestValue(values map[string]interface{}) interface{} {
	for _, value := range values {
		fmt.Println("Evaluating value:", value)
		if value == nil {
			continue
		}
		if str, ok := value.(string); ok && strings.TrimSpace(str) != "" {
			return m.selectStringBestValue(values)
		}
		if arr, ok := value.([]interface{}); ok && len(arr) > 0 {
			fmt.Println("Evaluating array value:", value)
			return m.mergeLists(values)
		}
	}

	return nil
}

// selectStringBestValue chooses the best value from available sources
func (me *MappingEngine) selectStringBestValue(values map[string]interface{}) interface{} {
	longestStr := ""
	for _, value := range values {
		// default behavior if string: return longest non-empty string
		if str, ok := value.(string); ok && strings.TrimSpace(str) != "" {
			trimmedVal := strings.TrimSpace(str)
			if len(trimmedVal) > len(longestStr) {
				longestStr = trimmedVal
			}
		}
	}

	return longestStr
}

// applyActions applies processing actions to values
func (me *MappingEngine) applyActions(values map[string]interface{}, actions []string) (interface{}, error) {

	var result interface{}
	// apply each action in sequence
	for _, action := range actions {
		action = strings.TrimSpace(action)
		switch action {
		case "normalize_general_amenities":
			fmt.Println("Normalizing general amenities with values:", values)
			result = me.normalizeGeneralAmenities(values)
		case "normalize_room_amenities":
			fmt.Println("Normalizing room amenities with values:", values)
			result = me.normalizeRoomAmenities(values)
		case "to_lowercase":
			result = me.toLowerCase(result)
		}
	}

	return result, nil
}

// normalizeAmenities normalizes amenities by mapping known variants to standard names
func (m *MappingEngine) normalizeGeneralAmenities(values map[string]interface{}) interface{} {

	// list of all known general amenities
	// if value from source is not in list, discard value
	// NOTE: should be a global constant or config
	generalAmenity := map[string]string{
		"businesscenter":  "business center",
		"business center": "business center",
		"gym":             "gym",
		"outdoor pool":    "outdoor pool",
		"indoor pool":     "indoor pool",
		"pool":            "outdoor pool", // NOTE: assuming pool means outdoor pool
		"airport shuttle": "airport shuttle",
		"childcare":       "childcare",
		"wifi":            "wifi",
		"drycleaning":     "dry cleaning",
		"dry cleaning":    "dry cleaning",
		"breakfast":       "breakfast",
		"bar":             "bar",       // NOTE: not in sample result.json but included for completeness, also assumed it's a general amenetity
		"parking":         "parking",   // NOTE: not in sample result.json but included for completeness, also assumed it's a general amenetity
		"concierge":       "concierge", // NOTE: not in sample result.json but included for completeness, also assumed it's a general amenetity
	}

	return m.normalizeAmenities(values, generalAmenity)
}

// normalizeRoomAmenities normalizes amenities by mapping known variants to standard names
func (m *MappingEngine) normalizeRoomAmenities(values map[string]interface{}) interface{} {
	// list of all known room amenities
	// if value from source is not in list, discard value
	// NOTE: should be a global constant or config
	roomAmenity := map[string]string{
		"aircon":         "aircon",
		"tv":             "tv",
		"coffee machine": "coffee machine",
		"kettle":         "kettle",
		"hair dryer":     "hair dryer",
		"iron":           "iron",
		"bathtub":        "bathtub",
		"tub":            "bathtub",
		"minibar":        "minibar", // NOTE: not in sample result.json but included for completeness, kept in room amenity as in sample_3.json
	}

	return m.normalizeAmenities(values, roomAmenity)
}

// normalizeAmenities normalizes amenities by mapping known variants to standard names
func (m *MappingEngine) normalizeAmenities(values map[string]interface{}, amenityMap map[string]string) interface{} {
	seenValue := make(map[string]bool)

	for _, value := range values {
		if arr, ok := value.([]interface{}); ok && len(arr) > 0 {
			merged := m.mergeLists(values)
			lowered := m.toLowerCase(merged)

			for _, v := range lowered.([]interface{}) {
				if str, ok := v.(string); ok {
					if norm, exists := amenityMap[str]; exists {
						seenValue[norm] = true
					} else {
						// NOTE: discard value if not in map of amenities
					}
				}
			}
		}
	}
	deduplicated := []string{}
	for k := range seenValue {
		deduplicated = append(deduplicated, k)
	}

	return deduplicated
}

// setNestedValue sets a value at a nested path in the result map
func (me *MappingEngine) setNestedValue(result map[string]interface{}, path string, value interface{}) {
	if value == nil {
		return
	}

	if path == "" {
		return
	}

	parts := strings.Split(path, ".")
	current := result

	// navigate to the parent of the target
	for _, part := range parts[:len(parts)-1] {
		if _, exists := current[part]; !exists {
			current[part] = make(map[string]interface{})
		}
		if next, ok := current[part].(map[string]interface{}); ok {
			current = next
		} else {
			// path conflict, can't proceed
			return
		}
	}

	finalKey := parts[len(parts)-1]
	current[finalKey] = value
}

// mergeLists merges multiple lists into one, removing duplicates
func (me *MappingEngine) mergeLists(values map[string]interface{}) interface{} {
	var allValues []interface{}
	for _, value := range values {
		if value != nil {
			allValues = append(allValues, value)
		}
	}

	var merged []interface{}
	seenItems := make(map[string]bool)

	for _, value := range allValues {
		if arr, ok := value.([]interface{}); ok {
			for _, item := range arr {
				itemStr := fmt.Sprintf("%v", item)
				if !seenItems[itemStr] {
					merged = append(merged, item)
					seenItems[itemStr] = true
				}
			}
		} else if value != nil {
			itemStr := fmt.Sprintf("%v", value)
			if !seenItems[itemStr] {
				merged = append(merged, value)
				seenItems[itemStr] = true
			}
		}
	}

	return merged
}

func (me *MappingEngine) toLowerCase(value interface{}) interface{} {
	switch v := value.(type) {
	case string:
		return strings.ToLower(v)
	case []interface{}:
		var result []interface{}
		for _, item := range v {
			result = append(result, me.toLowerCase(item))
		}
		return result
	default:
		return value
	}
}
