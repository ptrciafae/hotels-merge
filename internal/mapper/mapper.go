package mapper

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/tidwall/gjson"
)

const dataSupplierPrefix = "src::"

// list of all known general amenities
// if value from supplier is not in list, discard value
var generalAmenity = map[string]string{
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

// list of all known room amenities
// if value from supplier is not in list, discard value
var roomAmenity = map[string]string{
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

// MappingEngine handles data transformation based on mapping configuration
type MappingEngine struct {
	config MappingConfig
}

// MappingConfig represents the structure of mapping.json
type MappingConfig map[string]interface{}

// FieldMapping represents a field mapping with supplier paths and actions
type FieldMapping struct {
	SupplierPaths           map[string]interface{} // key: supplier_1, supplier_2, supplier_3, value: jsonpath or template
	Actions                 []string               // actions to apply
	ObjectArrayFieldMapping map[string][]string    // key: field name, value: possible supplier field names, used for merging object arrays

}

// SupplierData holds data from all suppliers
type SupplierData map[string]json.RawMessage

// HotelSupplierData represents the processed hotel data from all suppliers for a single hotel
type HotelSupplierData map[string]json.RawMessage

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

// Transform applies the mapping to supplier data
func (m *MappingEngine) Transform(suppliers SupplierData) (json.RawMessage, error) {
	// parse each supplier's array and group by hotel id
	// result: key: hotel id, value: hotel data from all suppliers
	hotelGroups, err := m.groupHotelsById(suppliers)
	if err != nil {
		return nil, fmt.Errorf("failed to group hotels: %w", err)
	}

	// transform each hotel group
	var results []map[string]interface{}
	for hotelId, hotelSuppliers := range hotelGroups {
		result := make(map[string]interface{})
		err := m.processMapping("", m.config, hotelSuppliers, result)

		// skip hotel if mapping cannot be processed
		if err != nil {
			fmt.Printf("Failed to process hotel %s: %v\n", hotelId, err)
			continue
		}

		results = append(results, result)
	}

	// marshal the results array
	output, err := json.Marshal(results)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal results: %w", err)
	}

	return json.RawMessage(output), nil
}

// groupHotelsById processes supplier arrays and groups hotels by their Ids
func (m *MappingEngine) groupHotelsById(suppliers SupplierData) (map[string]HotelSupplierData, error) {
	hotelGroups := make(map[string]HotelSupplierData)

	// id field mappings for each supplier
	idFieldMappings := m.extractIdFieldMapping()

	// process each supplier
	for supplierKey, supplierData := range suppliers {
		// parse the JSON array
		supplierArray := gjson.Get(string(supplierData), "@this")
		if !supplierArray.IsArray() {
			return nil, fmt.Errorf("supplier response %s is not an array", supplierKey)
		}

		// get the id field name for this supplier
		idField, exists := idFieldMappings[supplierKey]
		if !exists {
			return nil, fmt.Errorf("no id field mapping for supplier %s", supplierKey)
		}

		// process each hotel in the array
		for _, hotelItem := range supplierArray.Array() {
			// Extract hotel id
			hotelId := gjson.Get(hotelItem.Raw, idField)
			if !hotelId.Exists() || hotelId.String() == "" {
				fmt.Printf("Warning: No id found for hotel in supplier %s\n", supplierKey)
				continue
			}

			hotelIdStr := hotelId.String()

			// initialize hotel group if it doesn't exist
			if hotelGroups[hotelIdStr] == nil {
				hotelGroups[hotelIdStr] = make(HotelSupplierData)
			}

			// store this hotel's data for this supplier
			hotelGroups[hotelIdStr][supplierKey] = json.RawMessage(hotelItem.Raw)
		}
	}

	return hotelGroups, nil
}

func (m *MappingEngine) extractIdFieldMapping() map[string]string {
	// extracts from existing mapping.json:
	// "id": {
	//     "src::supplier_1": "Id",
	//     "src::supplier_2": "id",
	//     "src::supplier_3": "hotel_id"
	// }

	idMappings := make(map[string]string)
	idConfig := m.config["id"] // gets the "id" mapping, update key if result id field changes

	for key, value := range idConfig.(map[string]interface{}) {
		if strings.HasPrefix(key, dataSupplierPrefix) {
			supplierName := strings.TrimPrefix(key, dataSupplierPrefix) // "supplier_1", "supplier_2", etc.
			idMappings[supplierName] = value.(string)                   // "Id", "id", "hotel_id"
		}
	}

	return idMappings
}

// processMapping recursively processes the mapping configuration
func (m *MappingEngine) processMapping(currentPath string, config interface{}, suppliers HotelSupplierData, result map[string]interface{}) error {

	switch v := config.(type) {
	case map[string]interface{}:
		if m.isLeafMapping(v) {
			value, err := m.processLeafMapping(v, suppliers)
			if err != nil {
				return err
			}
			m.setNestedValue(result, currentPath, value)
		} else { // recursive processing for nested objects
			for key, value := range v {
				newPath := key
				if currentPath != "" {
					newPath = currentPath + "." + key
				}
				if err := m.processMapping(newPath, value, suppliers, result); err != nil {
					return err
				}
			}
		}
	case MappingConfig: // unfortunately golang doesn't support type aliasing in type switches
		return m.processMapping(currentPath, map[string]interface{}(v), suppliers, result)
	}
	return nil
}

// isLeafMapping checks if a mapping object contains supplier definitions
func (*MappingEngine) isLeafMapping(mapping map[string]interface{}) bool {
	for key := range mapping {
		if strings.HasPrefix(key, dataSupplierPrefix) {
			return true
		}
	}
	return false
}

// processLeafMapping processes a leaf mapping with supplier paths
func (m *MappingEngine) processLeafMapping(mapping map[string]interface{}, suppliers HotelSupplierData) (interface{}, error) {
	fieldMapping := m.parseFieldMapping(mapping)

	// extract values from all suppliers
	values := m.extractValuesFromSuppliers(fieldMapping.SupplierPaths, suppliers)
	// apply actions if specified
	if len(fieldMapping.Actions) > 0 {
		return m.applyActions(values, fieldMapping.Actions, fieldMapping)
	}

	return m.selectBestValue(values), nil
}

// parseFieldMapping converts raw mapping to FieldMapping struct
func (*MappingEngine) parseFieldMapping(mapping map[string]interface{}) FieldMapping {
	fieldMapping := FieldMapping{
		SupplierPaths:           make(map[string]interface{}),
		Actions:                 []string{},
		ObjectArrayFieldMapping: make(map[string][]string),
	}

	for key, value := range mapping {
		if strings.HasPrefix(key, dataSupplierPrefix) {
			fieldMapping.SupplierPaths[key] = value
		} else if key == "actions" {
			if actions, ok := value.([]interface{}); ok {
				for _, action := range actions {
					if actionStr, ok := action.(string); ok {
						fieldMapping.Actions = append(fieldMapping.Actions, actionStr)
					}
				}
			}
		} else if key == "field_mapping" {
			if fieldMapConfig, ok := value.(map[string]interface{}); ok {
				for fieldName, possibleFields := range fieldMapConfig {
					if fieldList, ok := possibleFields.([]interface{}); ok {
						var stringList []string
						for _, field := range fieldList {
							if fieldStr, ok := field.(string); ok {
								stringList = append(stringList, fieldStr)
							}
						}
						fieldMapping.ObjectArrayFieldMapping[fieldName] = stringList
					}
				}
			}
		}

	}

	return fieldMapping
}

// extractValuesFromSuppliers extracts values from all suppliers using JSONPath or templates
func (m *MappingEngine) extractValuesFromSuppliers(supplierPaths map[string]interface{}, suppliers HotelSupplierData) map[string]interface{} {
	values := make(map[string]interface{})

	for supplierKey, pathOrTemplate := range supplierPaths {
		supplierName := strings.TrimPrefix(supplierKey, dataSupplierPrefix)
		if supplierData, hasSupplier := suppliers[supplierName]; hasSupplier {
			value := m.extractValue(supplierData, pathOrTemplate)
			if value != nil {
				values[supplierKey] = value
			}
		}
	}

	return values
}

// extractValue extracts a value using JSONPath or template
func (m *MappingEngine) extractValue(supplierData json.RawMessage, pathOrTemplate interface{}) interface{} {
	if pathOrTemplate == nil {
		return nil
	}

	pathStr, ok := pathOrTemplate.(string)
	if !ok {
		return nil
	}

	// handle template strings (e.g., "{{Address}}, {{PostalCode}}")
	if m.isTemplate(pathStr) {
		return m.processTemplate(supplierData, pathStr)
	}

	// handle regular JSONPath
	result := gjson.Get(string(supplierData), pathStr)
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
func (*MappingEngine) isTemplate(str string) bool {
	return strings.Contains(str, "{{") && strings.Contains(str, "}}")
}

// processTemplate processes template strings like "{{Address}}, {{PostalCode}}"
func (*MappingEngine) processTemplate(supplierData json.RawMessage, template string) interface{} {
	// find all template variables
	re := regexp.MustCompile(`\{\{([^}]+)\}\}`)
	matches := re.FindAllStringSubmatch(template, -1)

	result := template
	for _, match := range matches {
		if len(match) >= 2 {
			fieldName := match[1]
			placeholder := match[0]

			// extract value for this field
			value := gjson.Get(string(supplierData), fieldName)
			if value.Exists() {
				result = strings.ReplaceAll(result, placeholder, strings.TrimSpace(value.String()))
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

// selectBestValue chooses the best value from available suppliers
// for non-strings: returns the first non-nil value
// for strings: delegates to selectStringBestValue for custom logic
// NOTE: can be configurable
func (m *MappingEngine) selectBestValue(values map[string]interface{}) interface{} {
	for _, value := range values {
		if value == nil {
			continue
		}

		switch v := value.(type) {
		case string:
			if strings.TrimSpace(v) != "" {
				return m.selectStringBestValue(values)
			}
		default:
			return v // first non-nil, non-string value
		}
	}
	return nil
}

// selectStringBestValue chooses the best value from available suppliers
func (*MappingEngine) selectStringBestValue(values map[string]interface{}) interface{} {
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
func (m *MappingEngine) applyActions(values map[string]interface{}, actions []string, fieldMapping FieldMapping) (interface{}, error) {
	var result interface{}

	// apply each action in sequence
	for _, action := range actions {
		action = strings.TrimSpace(action)
		switch action {
		case "normalize_general_amenities":
			result = m.normalizeGeneralAmenities(values)
		case "normalize_room_amenities":
			result = m.normalizeRoomAmenities(values)
		case "merge_image_arrays":
			result = m.mergeObjectArrays(values, fieldMapping.ObjectArrayFieldMapping, "link") // "link" is the unique identifier for the object array
		case "to_lowercase":
			result = m.toLowerCase(result)
		}
	}

	return result, nil
}

// normalizeAmenities normalizes amenities by mapping known variants to standard names
func (m *MappingEngine) normalizeGeneralAmenities(values map[string]interface{}) interface{} {
	return m.normalizeAmenities(values, generalAmenity)
}

// normalizeRoomAmenities normalizes amenities by mapping known variants to standard names
func (m *MappingEngine) normalizeRoomAmenities(values map[string]interface{}) interface{} {
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
		break // break after first non-empty as we already merged all lists
	}

	deduplicated := []string{}
	for k := range seenValue {
		deduplicated = append(deduplicated, k)
	}

	return deduplicated
}

// mergeObjectArrays merges arrays of objects from multiple suppliers
func (m *MappingEngine) mergeObjectArrays(values map[string]interface{}, fieldMapping map[string][]string, uniqueIdentifier string) interface{} {
	var uniqueObjects []map[string]interface{}
	seenObject := make(map[string]bool) // track by link to avoid duplicates

	// process each supplier
	for _, value := range values {
		if value == nil {
			continue
		}

		// handle array of objects
		if arr, ok := value.([]interface{}); ok {
			for _, objInterface := range arr {
				if obj, ok := objInterface.(map[string]interface{}); ok {
					normalizedObject := m.normalizeObject(obj, fieldMapping)

					if identifier, hasIdentifier := normalizedObject[uniqueIdentifier].(string); hasIdentifier && identifier != "" {
						if !seenObject[identifier] {
							uniqueObjects = append(uniqueObjects, normalizedObject)
							seenObject[identifier] = true
						}
					}
				}
			}
		}
	}

	deduplicated := []map[string]interface{}{}
	deduplicated = append(deduplicated, uniqueObjects...)
	return deduplicated
}

// normalizeObject substitutes mapped field names in given object
func (*MappingEngine) normalizeObject(obj map[string]interface{}, fieldMapping map[string][]string) map[string]interface{} {
	result := make(map[string]interface{})

	for targetField, possibleSupplierFields := range fieldMapping {
		for _, supplierField := range possibleSupplierFields { // try to find the value using possible supplier field names, unfortunately since it's not a mapped by supplier field name
			if value, exists := obj[supplierField]; exists && value != nil {
				if str, ok := value.(string); ok && strings.TrimSpace(str) != "" {
					result[targetField] = strings.TrimSpace(str)
					break // use first valid value found
				}
			}
		}
	}

	return result
}

// setNestedValue sets a value at a nested path in the result map
func (*MappingEngine) setNestedValue(result map[string]interface{}, path string, value interface{}) {
	if value == nil || path == "" {
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
		} else { // path conflict, can't proceed
			return
		}
	}

	finalKey := parts[len(parts)-1]
	current[finalKey] = value
}

// mergeLists merges multiple lists into one, removing duplicates
func (*MappingEngine) mergeLists(values map[string]interface{}) interface{} {
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

func (m *MappingEngine) toLowerCase(value interface{}) interface{} {
	switch v := value.(type) {
	case string:
		return strings.ToLower(v)
	case []interface{}:
		var result []interface{}
		for _, item := range v {
			result = append(result, m.toLowerCase(item))
		}
		return result
	default:
		return value
	}
}
