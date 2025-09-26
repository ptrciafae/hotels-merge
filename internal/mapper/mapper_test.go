package mapper_test

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/ptrciafae/hotels-merge/internal/mapper"
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
		},
		"destination_id": {
			"src::source_1": "DestinationId",
			"src::source_2": "destination",
			"src::source_3": "destination_id"
		}
	}`

	engine, err := mapper.NewMappingEngine([]byte(mappingConfig))
	require.NoError(t, err)

	// sample source data
	sources := mapper.SupplierData{
		"source_1": json.RawMessage(`[{"Id": "123", "Name": "Hotel A", "DestinationId": 1}]`),
		"source_2": json.RawMessage(`[{"id": "123", "name": "Hotel A", "destination": 1}]`),
		"source_3": json.RawMessage(`[{"hotel_id": "123", "hotel_name": "Hotel A", "destination_id": 1}]`),
	}

	result, err := engine.Transform(sources)
	require.NoError(t, err)

	var transformed []map[string]interface{}
	err = json.Unmarshal(result, &transformed)
	require.NoError(t, err)

	assert.Equal(t, "123", transformed[0]["id"])
	assert.Equal(t, "Hotel A", transformed[0]["name"])

	assert.Equal(t, float64(1), transformed[0]["destination_id"]) // JSON numbers are float64
}

func TestMappingEngine_NestedJSONPath(t *testing.T) {
	mappingConfig := `{
		"id": {
			"src::source_1": "Id",
			"src::source_2": "id",
			"src::source_3": "hotel_id"
		},
		"location": {
			"country": {
				"src::source_3": "location.country"
			},
			"lat": {
				"src::source_1": "Latitude",
				"src::source_2": "lat",
				"src::source_3": null
			},
			"lng": {
				"src::source_1": "Longitude",
				"src::source_2": "lng",
				"src::source_3": null
			}
	}
	}`

	engine, err := mapper.NewMappingEngine([]byte(mappingConfig))
	require.NoError(t, err)

	sources := mapper.SupplierData{
		"source_1": json.RawMessage(`[{"Id": "123","Latitude":35.6926,"Longitude":139.690965}]`),
		"source_2": json.RawMessage(`[{"id": "123","lat":35.6926,"lng":139.690965}]`),
		"source_3": json.RawMessage(`[{"hotel_id": "123","location": {"country": "Singapore", "city": "Singapore"}}]`),
	}

	result, err := engine.Transform(sources)
	require.NoError(t, err)

	var transformed []map[string]interface{}
	err = json.Unmarshal(result, &transformed)
	require.NoError(t, err)

	location := transformed[0]["location"].(map[string]interface{})
	assert.Equal(t, 35.6926, location["lat"])
	assert.Equal(t, 139.690965, location["lng"])
	assert.Equal(t, "Singapore", location["country"])
}

func TestMappingEngine_DifferentDataTypes(t *testing.T) {
	mappingConfig := `{
		"id": {
			"src::source_1": "Id",
			"src::source_2": "id",
			"src::source_3": "hotel_id"
		},
		"location": {
			"lat": {
				"src::source_1": "Latitude",
				"src::source_2": "lat"
			},
			"lng": {
				"src::source_1": "Longitude",
				"src::source_2": "lng"
			}
		}
	}`

	engine, err := mapper.NewMappingEngine([]byte(mappingConfig))
	require.NoError(t, err)

	sources := mapper.SupplierData{
		"source_1": json.RawMessage(`[{"Id": "123","Latitude":"","Longitude":""}]`),
		"source_2": json.RawMessage(`[{"id": "123","lat":35.6926,"lng":139.690965}]`),
	}

	result, err := engine.Transform(sources)
	require.NoError(t, err)

	var transformed []map[string]interface{}
	err = json.Unmarshal(result, &transformed)
	require.NoError(t, err)

	location := transformed[0]["location"].(map[string]interface{})
	assert.Equal(t, 35.6926, location["lat"])
	assert.Equal(t, 139.690965, location["lng"])
}
func TestMappingEngine_GeneralAmenities(t *testing.T) {
	mappingConfig := `{
		"id": {
			"src::source_1": "Id",
			"src::source_2": "id",
			"src::source_3": "hotel_id"
		},
		"amenities": {
			"general": {
				"src::source_1": "Facilities",
				"src::source_2": null,
				"src::source_3": "amenities.general",
				"actions": ["normalize_general_amenities"]
			}
		}
	}`

	engine, err := mapper.NewMappingEngine([]byte(mappingConfig))
	require.NoError(t, err)

	sources := mapper.SupplierData{
		"source_1": json.RawMessage(`[{"Id": "123", "Facilities": ["WiFi", "BusinessCenter", "gym"]}]`),
		"source_3": json.RawMessage(`[{"hotel_id": "123","amenities": {"general": ["outdoor pool", "GYM"]}}]`),
	}

	result, err := engine.Transform(sources)
	require.NoError(t, err)

	var transformed []map[string]interface{}
	err = json.Unmarshal(result, &transformed)
	require.NoError(t, err)

	amenities := transformed[0]["amenities"].(map[string]interface{})
	general := amenities["general"].([]interface{})

	assert.Len(t, general, 4) // deduplicated length
	assert.Contains(t, general, "wifi")
	assert.Contains(t, general, "business center")
	assert.Contains(t, general, "outdoor pool")
	assert.Contains(t, general, "gym") // deduplicated and lowercased
}

func TestMappingEngine_RoomAmenities(t *testing.T) {
	mappingConfig := `{
		"id": {
			"src::source_1": "Id",
			"src::source_2": "id",
			"src::source_3": "hotel_id"
		},
		"amenities": {
			"room": {
				"src::source_1": "Facilities",
				"src::source_2": "amenities",
				"src::source_3": "amenities.room",
				"actions": ["normalize_room_amenities"]
			}
		}
	}`

	engine, err := mapper.NewMappingEngine([]byte(mappingConfig))
	require.NoError(t, err)

	sources := mapper.SupplierData{
		"source_1": json.RawMessage(`[{"Id": "123", "Facilities": ["Aircon", "Tv", "gym"]}]`),
		"source_2": json.RawMessage(`[{"id": "123", "amenities": ["Aircon", "Tv", "Tub"]}]`),
		"source_3": json.RawMessage(`[{"hotel_id": "123","amenities": {"room": ["outdoor pool", "BathTub"]}}]`),
	}

	result, err := engine.Transform(sources)
	require.NoError(t, err)

	var transformed []map[string]interface{}
	err = json.Unmarshal(result, &transformed)
	require.NoError(t, err)

	amenities := transformed[0]["amenities"].(map[string]interface{})
	room := amenities["room"].([]interface{})

	assert.Len(t, room, 3) // deduplicated length
	assert.Contains(t, room, "aircon")
	assert.Contains(t, room, "tv")
	assert.Contains(t, room, "bathtub") // deduplicated and lowercased
}

// Test function to verify the image merging works correctly
func TestMappingEngine_ImageArrayProcessing(t *testing.T) {
	mappingConfig := `{
		"id": {
			"src::source_1": "Id",
			"src::source_2": "id",
			"src::source_3": "hotel_id"
		},
		"images": {
			"rooms": {
				"src::source_1": null,
				"src::source_2": "images.rooms",
				"src::source_3": "images.rooms",
				"actions": ["merge_image_arrays"],
				"field_mapping": {
					"link": ["url", "link"],
					"description": ["description", "caption"]
				}
			}
		}
	}`

	engine, err := mapper.NewMappingEngine([]byte(mappingConfig))
	if err != nil {
		panic(err)
	}

	sources := mapper.SupplierData{
		"source_2": json.RawMessage(`[{
			"id": "123",
			"images": {
				"rooms": [
					{"url": "https://example.com/1.jpg", "description": "Room 1"},
					{"url": "https://example.com/2.jpg", "description": "Bathroom"}
				]
			}
		}]`),
		"source_3": json.RawMessage(`[{
			"hotel_id": "123",
			"images": {
				"rooms": [
					{"link": "https://example.com/1.jpg", "caption": "Room 1"},
					{"link": "https://example.com/3.jpg", "caption": "Double room"}
				]
			}
		}]`),
	}

	result, err := engine.Transform(sources)
	if err != nil {
		panic(err)
	}

	var transformed []map[string]interface{}
	err = json.Unmarshal(result, &transformed)
	if err != nil {
		panic(err)
	}

	images := transformed[0]["images"].(map[string]interface{})
	rooms := images["rooms"].([]interface{})

	expectedLinks := map[string]bool{
		"https://example.com/1.jpg": true,
		"https://example.com/2.jpg": true,
		"https://example.com/3.jpg": true,
	}

	for _, img := range rooms {
		image := img.(map[string]interface{})
		link := image["link"].(string)
		assert.True(t, expectedLinks[link])
	}
	assert.Len(t, rooms, 3) // deduplicated length
}

func TestMappingEngine_TemplateProcessing(t *testing.T) {
	mappingConfig := `{
		"id": {
			"src::source_1": "Id",
			"src::source_2": "id",
			"src::source_3": "hotel_id"
		},
		"location": {
			"address": {
				"src::source_1": "{{Address}}, {{PostalCode}}",
				"src::source_2": "address",
				"src::source_3": "location.address"
			}
		}
	}`

	engine, err := mapper.NewMappingEngine([]byte(mappingConfig))
	require.NoError(t, err)

	sources := mapper.SupplierData{
		"source_1": json.RawMessage(`[{"Id": "123", "Address": "123 Main St", "PostalCode": "12345"}]`),
		"source_2": json.RawMessage(`[{"id": "123", "address": "456 Oak Ave"}]`),
	}

	result, err := engine.Transform(sources)
	require.NoError(t, err)

	var transformed []map[string]interface{}
	err = json.Unmarshal(result, &transformed)
	require.NoError(t, err)

	location := transformed[0]["location"].(map[string]interface{})
	assert.Equal(t, "123 Main St, 12345", location["address"]) // source_1 value still longer than source_2
}

func TestMappingEngine_FromSampleFiles(t *testing.T) {
	// load mapping configuration from file
	file, err := os.Open("../../testdata/mapping.json")
	require.NoError(t, err)
	defer file.Close()
	mappingConfig, err := io.ReadAll(file)
	require.NoError(t, err)

	// load source files
	file1, err := os.Open("../../testdata/source_1.json")
	require.NoError(t, err)
	defer file1.Close()
	source1, err := io.ReadAll(file1)
	require.NoError(t, err)

	file2, err := os.Open("../../testdata/source_2.json")
	require.NoError(t, err)
	defer file2.Close()
	source2, err := io.ReadAll(file2)
	require.NoError(t, err)

	file3, err := os.Open("../../testdata/source_3.json")
	require.NoError(t, err)
	defer file3.Close()
	source3, err := io.ReadAll(file3)
	require.NoError(t, err)

	// perform test
	engine, err := mapper.NewMappingEngine(mappingConfig)
	require.NoError(t, err)

	sources := mapper.SupplierData{
		"source_1": source1,
		"source_2": source2,
		"source_3": source3,
	}
	result, err := engine.Transform(sources)
	require.NoError(t, err)

	fmt.Println(string(result))
	// NOTE: intentionally failing the test to output the result
	assert.NotEqual(t, result, result) // TODO: implement actual assertion logic
}
