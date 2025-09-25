# Description

Web server taking data from various suppliers with non-uniform format and with duplicated value (example:
[acme](https://5f2be0b4ffc88500167b85a0.mockapi.io/suppliers/acme),
[patagonia](https://5f2be0b4ffc88500167b85a0.mockapi.io/suppliers/patagonia),
[paperflies](https://5f2be0b4ffc88500167b85a0.mockapi.io/suppliers/paperflies))
transform it into a unified response ([samples/result.json](https://github.com/ptrciafae/hotels-merge/blob/16d923e012b0a52608df31faac4a51c56cdb6e69/samples/result.json)).

# Running the application

1. Install go in local machine, follow the [instructions for your operating system](https://www.bytesizego.com/blog/installing-golang).
2. Download and build the solution in your workspace

```bash
$ git clone https://github.com/ptrciafae/hotels-merge.git
$ cd hotels-merge
$ go mod tidy
$ go run cmd/main.go
```

3. Service runs on: `127.0.0.1:8085` which you can reach from your browser, through curl, or via Postman

# APIs

## Endpints

### /

home page retrieves all hotels merged from all the sources already normalized and deduplicated

### /hotel

retrieves hotels filtered with supplied params

| Params         | Description                 |
| -------------- | --------------------------- |
| id             | filters by `id` of hotel    |
| destination_id | filters by `destination_id` |

_NOTE_: endpoint only accepts either `id` or `destination_id` at a time. Status `400 - BadRequest` is returned if both are supplied at the same time.

_ASSUMPTION_: multiple hotels can share the same `destination_id`

## Response

[As struct](https://github.com/ptrciafae/hotels-merge/blob/16d923e012b0a52608df31faac4a51c56cdb6e69/internal/hotels/hotels.go)

# Design

The core of my approach rests on two principles: the Principle of Least Astonishment and extensibility. To uphold the first, I use a configuration-driven method for handling normalization. At the same time, I’ve designed the configuration to be straightforward to update and flexible enough to accommodate new rules to satisfy the second priciple.

The configuration lives in [mapping.json](https://github.com/ptrciafae/hotels-merge/blob/16d923e012b0a52608df31faac4a51c56cdb6e69/mapping.json). As the name implies it maps the fields from the supplier response to the intended response fields returned by the API. If we onboard more suppliers, it would be easy to add mapping from their unique response for the API to consider.

## Selecting the Best Data

The logic is a bit naive. Since the variations between hotel data from different suppliers in the example is little: for strings, best value is the longest string. For non-strings, the first non-empty value.

## Actions

### Special Fields

Some fields, require special manipulation, example: `ameneties` where normalization is non-standard. (I initially opted to do a series of actions: `lower_case`, `split_words`, `merge_list`. But the transformation is unpredicable, example: `WiFi` is being rendered as `wi fi`) So specific actions that abstract the implementation is added.

- `normalize_general_amenities` / `normalize_room_amenities` - since there's a single `ameneties` field mapped to both general and room amenities, a list of allowable values for each is introduced. It also is a map of values seen in the wild and their normalized values.

- `merge_image_arrays` - the result are in the format `[{"link": "", "description": ""}]`, supported by additional configuration `field_mapping` to achieve mapping of nested fields for each item in the array. This also abstracts a logic of enforcing uniqueness based on `link`.

# Mapping JSON spec

```json
{
    "{{response_field_name}}": {
        "src::{{source_name}}": "{{path_in_source}}"
    },
    "{{parent_response_field_name}}": {
        "{{nested_response_field_name}}": {
            "src::{{source_name}}": "{{path_in_source}}", // same as above
            "src::{{source_name}}": "{{path_in_source}}, {{another_field}}" // combination of multiple fields
            "actions": ["{{action_option}}"] // applied sequentially
        }
    },
    "rooms": {
        "src::patagonia": "images.rooms",
        "src::paperflies": "images.rooms",
        "actions": [
            "merge_image_arrays"
        ],
        "field_mapping": { // "url" and "link" fields from sources map to "link" in response
            "link": [
                "url",
                "link"
            ]
        }
    }
}
```

## Keys in spec

- `src::` prefix to indicate if key is source and source name
- reserved keys

1. `actions` for custom logic on merging values and other normalization logic
1. `field_mapping` particularly for `merge_image_arrays` where the array items are objects and mappings from supplier format to response is required

- no prefix - key is a field name in the result

## Values in spec

- _JSONPath where to extract value_: `Description`
- Template of JSONPaths: `{{FieldOne}} and {{FieldTwo}}`

# Development

1. Started with the response format (didn't change anything, it seemed straightforward).
2. Added the mapping engine but AI (Claude code free tier) did the heavily lifting on the logic of the mapping engine, especially with the formation of the response based on the mapping rules. So, parsing the mapping.json file, extracting the correct values from the supplier data, and transforming into its counterpart field in the response. What I tweaked were the custom actions, decisions on best value. To ensure I get my desired outcome, mapper logic is the only one developed with TDD.

NOTE: the last unit test is intentionally failing to allow viewing of the logged values

3. Server was added with fetching the data from suppliers and allowing queries on the data via the endpoints

# Future optimization

- I've tested only on the sample data, I'm sure there are multiple ways the logic might fail. More testin needs to be done.
- Clean up some of the AI-generated code I haven't touched.
