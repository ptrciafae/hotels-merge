# Mapping JSON spec
```
{
    "{{result-field-name}}": {
        "src::{{source-name}}": "{{path-in-source}}"
    },
    "{{nested-result-field-name}}": {
        "{{result-field-name}}": {
            "src::{{source-name}}": "{{path-in-source}}", // same as above
            "src::{{source-name}}": "{{path-in-source}}, {{another-field}}" // combination of multiple fields
            "action": "{{action-option}}"
        }
    },
}
```

## Keys in spec
* `src::` prefix to indicate if key is source
* no prefix - key is a field name in the result

## Values
Path in source: `path.in.source`

# Notes
When a field is a combination of 2 or more fields in other 

# Logic
Get field values based on `mapping.json`
if values between source are all equal: copy as-is to destination field.

## Actions

`merge_lists` - deduplicate on combination of values
// TODO: make default on list after all actions are applied
no action - all equal

`merge_image_arrays`: `merge_object_arrays` with `link` as unique identifier
// TODO: make identifier configurable and accept multiple identifiers as well


* order matters: `merge_lists, add_spaces, to_lowercase`, first action is applied first

TODO: With regards to ameneties, there is a special, hardcoded mapping (WiFi & Business Center)
# To think about