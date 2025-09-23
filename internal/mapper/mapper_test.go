package mapper

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMappingEngine_BasicTransformation(t *testing.T) {
	// sample mapping configuration
	mappingConfig := `{
		"id": {
			"src::source_1": "Id",
			"src::source_2": "id",
			"src::source_3": "hotel_id"
		},
		"name": {
			"src::source_1": "Name",
			"src::source_2": "name",
			"src::source_3": "hotel_name"
		}
	}`

	engine, err := NewMappingEngine([]byte(mappingConfig))
	require.NoError(t, err)

	// sample source data
	sources := SourceData{
		"source_1": json.RawMessage(`{"Id": "123", "Name": "Hotel A"}`),
		"source_2": json.RawMessage(`{"id": "123", "name": "Hotel A"}`),
		"source_3": json.RawMessage(`{"hotel_id": "123", "hotel_name": "Hotel A"}`),
	}

	result, err := engine.Transform(sources)
	require.NoError(t, err)

	var transformed map[string]interface{}
	err = json.Unmarshal(result, &transformed)
	require.NoError(t, err)

	assert.Equal(t, "123", transformed["id"])
	assert.Equal(t, "Hotel A", transformed["name"])
}

func TestMappingEngine_NestedJSONPath(t *testing.T) {
	mappingConfig := `{
		"location": {
			"country": {
				"src::source_3": "location.country"
			}
		}
	}`

	engine, err := NewMappingEngine([]byte(mappingConfig))
	require.NoError(t, err)

	sources := SourceData{
		"source_3": json.RawMessage(`{"location": {"country": "Singapore", "city": "Singapore"}}`),
	}

	result, err := engine.Transform(sources)
	require.NoError(t, err)

	var transformed map[string]interface{}
	err = json.Unmarshal(result, &transformed)
	require.NoError(t, err)

	location := transformed["location"].(map[string]interface{})
	assert.Equal(t, "Singapore", location["country"])
}

func TestMappingEngine_GeneralAmenities(t *testing.T) {
	mappingConfig := `{
		"amenities": {
			"general": {
				"src::source_1": "Facilities",
				"src::source_2": null,
				"src::source_3": "amenities.general",
				"action": "normalize_general_amenities"
			}
		}
	}`

	engine, err := NewMappingEngine([]byte(mappingConfig))
	require.NoError(t, err)

	sources := SourceData{
		"source_1": json.RawMessage(`{"Facilities": ["WiFi", "BusinessCenter", "gym"]}`),
		"source_3": json.RawMessage(`{"amenities": {"general": ["outdoor pool", "GYM"]}}`),
	}

	result, err := engine.Transform(sources)
	require.NoError(t, err)

	var transformed map[string]interface{}
	err = json.Unmarshal(result, &transformed)
	require.NoError(t, err)

	amenities := transformed["amenities"].(map[string]interface{})
	general := amenities["general"].([]interface{})

	assert.Len(t, general, 4) // deduplicated length
	assert.Contains(t, general, "wifi")
	assert.Contains(t, general, "business center")
	assert.Contains(t, general, "outdoor pool")
	assert.Contains(t, general, "gym") // deduplicated and lowercased
}

func TestMappingEngine_RoomAmenities(t *testing.T) {
	mappingConfig := `{
		"amenities": {
			"room": {
				"src::source_1": "Facilities",
				"src::source_2": "amenities",
				"src::source_3": "amenities.room",
				"action": "normalize_room_amenities"
			}
		}
	}`

	engine, err := NewMappingEngine([]byte(mappingConfig))
	require.NoError(t, err)

	sources := SourceData{
		"source_1": json.RawMessage(`{"Facilities": ["Aircon", "Tv", "gym"]}`),
		"source_2": json.RawMessage(`{"amenities": ["Aircon", "Tv", "Tub"]}`),
		"source_3": json.RawMessage(`{"amenities": {"room": ["outdoor pool", "BathTub"]}}`),
	}

	result, err := engine.Transform(sources)
	require.NoError(t, err)

	var transformed map[string]interface{}
	err = json.Unmarshal(result, &transformed)
	require.NoError(t, err)

	amenities := transformed["amenities"].(map[string]interface{})
	room := amenities["room"].([]interface{})

	assert.Len(t, room, 3) // deduplicated length
	assert.Contains(t, room, "aircon")
	assert.Contains(t, room, "tv")
	assert.Contains(t, room, "bathtub") // deduplicated and lowercased
}

func TestMappingEngine_TemplateProcessing(t *testing.T) {
	mappingConfig := `{
		"location": {
			"address": {
				"src::source_1": "{{Address}}, {{PostalCode}}",
				"src::source_2": "address",
				"src::source_3": "location.address"
			}
		}
	}`

	engine, err := NewMappingEngine([]byte(mappingConfig))
	require.NoError(t, err)

	sources := SourceData{
		"source_1": json.RawMessage(`{"Address": "123 Main St", "PostalCode": "12345"}`),
		"source_2": json.RawMessage(`{"address": "456 Oak Ave"}`),
	}

	result, err := engine.Transform(sources)
	require.NoError(t, err)

	var transformed map[string]interface{}
	err = json.Unmarshal(result, &transformed)
	require.NoError(t, err)

	location := transformed["location"].(map[string]interface{})
	assert.Equal(t, "123 Main St, 12345", location["address"]) // source_1 value still longer than source_2
}
