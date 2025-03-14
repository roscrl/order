package order

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// SchemaProperty represents a property in a JSON schema, which may contain nested properties
type SchemaProperty struct {
	Name       string
	Properties []*SchemaProperty
}

// Lint validates that a YAML or JSON file follows the property order specified in a JSON schema
func Lint(yamlOrJsonPath, jsonSchemaPath string) error {
	content, err := os.ReadFile(yamlOrJsonPath)
	if err != nil {
		return err
	}

	var yamlRoot yaml.Node
	if strings.HasSuffix(yamlOrJsonPath, ".yaml") || strings.HasSuffix(yamlOrJsonPath, ".yml") {
		err = yaml.Unmarshal(content, &yamlRoot)
		if err != nil {
			return err
		}
	} else if strings.HasSuffix(yamlOrJsonPath, ".json") {
		// For JSON, we need to parse it in a way that preserves property order
		jsonReader := strings.NewReader(string(content))
		jsonNode, err := parseJSONWithOrder(jsonReader)
		if err != nil {
			return err
		}

		yamlRoot = *jsonNode
	} else {
		return errors.New("file must have .yaml, .yml, or .json extension")
	}

	// Extract schema properties in their original order
	schemaProperties, err := extractNestedSchemaOrder(jsonSchemaPath)
	if err != nil {
		return err
	}

	// Validate the YAML document against the schema properties
	// We start by validating the root level
	if yamlRoot.Kind == yaml.DocumentNode && len(yamlRoot.Content) > 0 {
		docNode := yamlRoot.Content[0]
		if docNode.Kind == yaml.MappingNode {
			return validateNodeAgainstSchema(docNode, schemaProperties)
		}
	}

	return nil
}

// validateNodeAgainstSchema checks if a YAML node's properties are in the correct order according to the schema
func validateNodeAgainstSchema(node *yaml.Node, schemaProperties []*SchemaProperty) error {
	if node.Kind != yaml.MappingNode {
		return nil // Not a mapping, nothing to validate
	}

	// Build a map of property names to their positions in the schema
	propertyPositions := make(map[string]int)
	for i, prop := range schemaProperties {
		propertyPositions[prop.Name] = i
	}

	// Extract the keys from the YAML mapping in order
	var keys []string
	var keyPositions = make(map[string]int) // Track position of each key in the actual document

	for i := 0; i < len(node.Content); i += 2 {
		key := node.Content[i].Value
		keys = append(keys, key)
		keyPositions[key] = i / 2
	}

	// Check if the properties are in the correct order
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			keyI := keys[i]
			keyJ := keys[j]

			// Skip keys that aren't in the schema
			posI, inSchemaI := propertyPositions[keyI]
			posJ, inSchemaJ := propertyPositions[keyJ]

			// If both keys are in the schema, check their order
			if inSchemaI && inSchemaJ && posI > posJ {
				return errors.New(
					"properties out of order: '" + keyI + "' should come after '" + keyJ +
						"' according to the schema")
			}
		}
	}

	// Now recursively validate nested properties
	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valueNode := node.Content[i+1]

		// Skip if this property isn't in the schema
		prop, ok := findPropertyByName(schemaProperties, keyNode.Value)
		if !ok || len(prop.Properties) == 0 || valueNode.Kind != yaml.MappingNode {
			continue
		}

		// Validate nested properties
		err := validateNodeAgainstSchema(valueNode, prop.Properties)
		if err != nil {
			return errors.New("in property '" + keyNode.Value + "': " + err.Error())
		}
	}

	return nil
}

// findPropertyByName finds a property in a slice of properties by its name
func findPropertyByName(properties []*SchemaProperty, name string) (*SchemaProperty, bool) {
	for _, prop := range properties {
		if prop.Name == name {
			return prop, true
		}
	}
	return nil, false
}

// extractSchemaOrderFromJsonSchemaPath extracts properties names in the order they appear in the original YAML/JSON file
func extractSchemaOrderFromJsonSchemaPath(jsonSchemaPath string) ([]string, error) {
	properties, err := extractNestedSchemaOrder(jsonSchemaPath)
	if err != nil {
		return nil, err
	}

	// Extract top-level property names
	var propertyNames []string
	for _, prop := range properties {
		propertyNames = append(propertyNames, prop.Name)
	}

	return propertyNames, nil
}

// extractNestedSchemaOrder extracts properties names in the order they appear in the original YAML/JSON file,
// including nested properties
func extractNestedSchemaOrder(jsonSchemaPath string) ([]*SchemaProperty, error) {
	file, err := os.Open(jsonSchemaPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return parseJSONSchema(file)
}

// parseJSONSchema parses a JSON schema from an io.Reader and extracts properties in order
func parseJSONSchema(r io.Reader) ([]*SchemaProperty, error) {
	decoder := json.NewDecoder(r)

	// Ensure we're at the start of the JSON object
	if t, err := decoder.Token(); err != nil {
		return nil, err
	} else if t != json.Delim('{') {
		return nil, errors.New("expected JSON object")
	}

	// Look for the "properties" field
	for {
		t, err := decoder.Token()
		if err != nil {
			return nil, err
		}

		// Check if we've reached the end of the object
		if t == json.Delim('}') {
			return nil, errors.New("properties not found")
		}

		// Check if we found the properties key
		if key, ok := t.(string); ok && key == "properties" {
			// Parse the properties object
			return parsePropertiesObject(decoder)
		}

		// Skip the value of this field since it's not "properties"
		if err := skipJSONValue(decoder); err != nil {
			return nil, err
		}
	}
}

// parsePropertiesObject parses a JSON object that represents schema properties
func parsePropertiesObject(decoder *json.Decoder) ([]*SchemaProperty, error) {
	// Ensure we're at the start of the properties object
	if t, err := decoder.Token(); err != nil {
		return nil, err
	} else if t != json.Delim('{') {
		return nil, errors.New("expected properties object")
	}

	var properties []*SchemaProperty

	// Parse each property
	for {
		t, err := decoder.Token()
		if err != nil {
			return nil, err
		}

		// Check if we've reached the end of the properties object
		if t == json.Delim('}') {
			break
		}

		// Get property name
		propertyName, ok := t.(string)
		if !ok {
			return nil, errors.New("expected property name string")
		}

		// Create the property
		property := &SchemaProperty{
			Name: propertyName,
		}

		// Parse the property object
		if t, err := decoder.Token(); err != nil {
			return nil, err
		} else if t != json.Delim('{') {
			return nil, errors.New("expected property object")
		}

		// Look for nested "properties" in this property
		for {
			t, err := decoder.Token()
			if err != nil {
				return nil, err
			}

			// Check if we've reached the end of this property
			if t == json.Delim('}') {
				break
			}

			// Check if this is a nested "properties" field
			if key, ok := t.(string); ok && key == "properties" {
				// Parse nested properties
				nestedProperties, err := parsePropertiesObject(decoder)
				if err != nil {
					return nil, err
				}
				property.Properties = nestedProperties
			} else {
				// Skip the value of this field
				if err := skipJSONValue(decoder); err != nil {
					return nil, err
				}
			}
		}

		properties = append(properties, property)
	}

	return properties, nil
}

// skipJSONValue skips over a JSON value (object, array, or primitive)
func skipJSONValue(decoder *json.Decoder) error {
	t, err := decoder.Token()
	if err != nil {
		return err
	}

	switch t {
	case json.Delim('{'):
		// Skip object
		depth := 1
		for depth > 0 {
			t, err := decoder.Token()
			if err != nil {
				return err
			}
			if t == json.Delim('{') {
				depth++
			} else if t == json.Delim('}') {
				depth--
			}
		}
	case json.Delim('['):
		// Skip array
		depth := 1
		for depth > 0 {
			t, err := decoder.Token()
			if err != nil {
				return err
			}
			if t == json.Delim('[') {
				depth++
			} else if t == json.Delim(']') {
				depth--
			}
		}
	default:
		// Primitive value, already consumed
	}

	return nil
}

// parseJSONWithOrder parses JSON content while preserving property order
func parseJSONWithOrder(r io.Reader) (*yaml.Node, error) {
	// Create a document node as the root
	doc := &yaml.Node{
		Kind: yaml.DocumentNode,
	}

	// Parse the JSON content
	obj, err := parseJSONObject(json.NewDecoder(r))
	if err != nil {
		return nil, err
	}

	// Add the parsed object as content of the document
	doc.Content = append(doc.Content, obj)

	return doc, nil
}

// parseJSONObject parses a JSON object into a YAML mapping node
func parseJSONObject(decoder *json.Decoder) (*yaml.Node, error) {
	// Ensure we're at the start of an object
	t, err := decoder.Token()
	if err != nil {
		return nil, err
	}
	if t != json.Delim('{') {
		return nil, errors.New("expected JSON object")
	}

	// Create a mapping node for the object
	obj := &yaml.Node{
		Kind: yaml.MappingNode,
	}

	// Parse key-value pairs
	for {
		// Read the next token, which should be a key or closing brace
		t, err := decoder.Token()
		if err != nil {
			return nil, err
		}

		// Check if we've reached the end of the object
		if t == json.Delim('}') {
			break
		}

		// Get the key name
		key, ok := t.(string)
		if !ok {
			return nil, errors.New("expected string key in JSON object")
		}

		// Create a scalar node for the key
		keyNode := &yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: key,
		}

		// Parse the value
		valueNode, err := parseJSONValue(decoder)
		if err != nil {
			return nil, err
		}

		// Add the key-value pair to the mapping
		obj.Content = append(obj.Content, keyNode, valueNode)
	}

	return obj, nil
}

// parseJSONValue parses a JSON value into a YAML node
func parseJSONValue(decoder *json.Decoder) (*yaml.Node, error) {
	t, err := decoder.Token()
	if err != nil {
		return nil, err
	}

	switch v := t.(type) {
	case string:
		return &yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: v,
		}, nil
	case float64:
		return &yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: fmt.Sprintf("%g", v),
		}, nil
	case bool:
		return &yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: fmt.Sprintf("%t", v),
		}, nil
	case nil:
		return &yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: "null",
		}, nil
	case json.Delim:
		if v == '{' {
			// For object, we need special handling as we've already consumed the opening brace
			objNode := &yaml.Node{
				Kind: yaml.MappingNode,
			}

			// Parse key-value pairs
			for {
				// Read key or closing brace
				keyToken, err := decoder.Token()
				if err != nil {
					return nil, err
				}

				// Check if we've reached the end of the object
				if keyToken == json.Delim('}') {
					break
				}

				// Get the key name
				key, ok := keyToken.(string)
				if !ok {
					return nil, errors.New("expected string key in JSON object")
				}

				// Create a scalar node for the key
				keyNode := &yaml.Node{
					Kind:  yaml.ScalarNode,
					Value: key,
				}

				// Parse the value
				valNode, err := parseJSONValue(decoder)
				if err != nil {
					return nil, err
				}

				// Add the key-value pair to the mapping
				objNode.Content = append(objNode.Content, keyNode, valNode)
			}

			return objNode, nil
		} else if v == '[' {
			return parseJSONArray(decoder)
		}
	}

	return nil, errors.New("unexpected JSON value")
}

// parseJSONArray parses a JSON array into a YAML sequence node
func parseJSONArray(decoder *json.Decoder) (*yaml.Node, error) {
	// Create a sequence node for the array
	arr := &yaml.Node{
		Kind: yaml.SequenceNode,
	}

	// Parse array elements
	for {
		// Peek at the next token
		t, err := decoder.Token()
		if err != nil {
			return nil, err
		}

		// Check if we've reached the end of the array
		if t == json.Delim(']') {
			break
		}

		// Handle the token based on its type
		var valueNode *yaml.Node

		switch v := t.(type) {
		case string:
			valueNode = &yaml.Node{
				Kind:  yaml.ScalarNode,
				Value: v,
			}
		case float64:
			valueNode = &yaml.Node{
				Kind:  yaml.ScalarNode,
				Value: fmt.Sprintf("%g", v),
			}
		case bool:
			valueNode = &yaml.Node{
				Kind:  yaml.ScalarNode,
				Value: fmt.Sprintf("%t", v),
			}
		case nil:
			valueNode = &yaml.Node{
				Kind:  yaml.ScalarNode,
				Value: "null",
			}
		case json.Delim:
			if v == '{' {
				// For nested objects in arrays, we need special handling
				// because we've already consumed the opening brace
				objNode := &yaml.Node{
					Kind: yaml.MappingNode,
				}

				// Parse key-value pairs
				for {
					// Read key or closing brace
					keyToken, err := decoder.Token()
					if err != nil {
						return nil, err
					}

					// Check if we've reached the end of the object
					if keyToken == json.Delim('}') {
						break
					}

					// Get the key name
					key, ok := keyToken.(string)
					if !ok {
						return nil, errors.New("expected string key in JSON object")
					}

					// Create a scalar node for the key
					keyNode := &yaml.Node{
						Kind:  yaml.ScalarNode,
						Value: key,
					}

					// Parse the value
					valNode, err := parseJSONValue(decoder)
					if err != nil {
						return nil, err
					}

					// Add the key-value pair to the mapping
					objNode.Content = append(objNode.Content, keyNode, valNode)
				}

				valueNode = objNode
			} else if v == '[' {
				// For nested arrays, recursively parse
				nestedArr, err := parseJSONArray(decoder)
				if err != nil {
					return nil, err
				}
				valueNode = nestedArr
			} else {
				return nil, errors.New("unexpected JSON delimiter")
			}
		default:
			return nil, errors.New("unexpected JSON value type in array")
		}

		// Add the value to the array
		if valueNode != nil {
			arr.Content = append(arr.Content, valueNode)
		}
	}

	return arr, nil
}
