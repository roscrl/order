package order

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestLint(t *testing.T) {
	// Create temporary directory for test files
	tempDir := t.TempDir()

	t.Run("Valid files", func(t *testing.T) {
		t.Parallel()

		// Create valid YAML file
		validYamlPath := filepath.Join(tempDir, "valid.yaml")
		validYamlContent := []byte(`---
first: value1
second: value2
`)
		err := os.WriteFile(validYamlPath, validYamlContent, 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Create valid schema file
		validSchemaPath := filepath.Join(tempDir, "schema.json")
		validSchemaContent := []byte(`{"properties": {"first": {}, "second": {}}}`)
		err = os.WriteFile(validSchemaPath, validSchemaContent, 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		err = Lint(validYamlPath, validSchemaPath)
		if err != nil {
			t.Errorf("Lint() returned an error for valid files: %v", err)
		}
	})

	t.Run("Invalid schema without properties", func(t *testing.T) {
		t.Parallel()

		// Create valid YAML file
		validYamlPath := filepath.Join(tempDir, "valid2.yaml")
		validYamlContent := []byte(`---
first: value1
second: value2
`)
		err := os.WriteFile(validYamlPath, validYamlContent, 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Create invalid schema file
		invalidSchemaPath := filepath.Join(tempDir, "invalid_schema.json")
		invalidSchemaContent := []byte(`{"type": "string"}`)
		err = os.WriteFile(invalidSchemaPath, invalidSchemaContent, 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		err = Lint(validYamlPath, invalidSchemaPath)
		if err == nil {
			t.Errorf("Lint() didn't return an error for schema without properties")
		}
	})

	t.Run("Nested properties validation", func(t *testing.T) {
		// Create a JSON schema with nested properties
		nestedSchemaPath := filepath.Join(tempDir, "nested_schema.json")
		nestedSchemaContent := []byte(`{
  "properties": {
    "personal": {
      "properties": {
        "name": {},
        "email": {},
        "phone": {}
      }
    },
    "address": {
      "properties": {
        "street": {},
        "city": {},
        "zipCode": {}
      }
    },
    "preferences": {}
  }
}`)
		err := os.WriteFile(nestedSchemaPath, nestedSchemaContent, 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Create a valid YAML file that follows the schema order
		validNestedYamlPath := filepath.Join(tempDir, "valid_nested.yaml")
		validNestedYamlContent := []byte(`---
personal:
  name: John Doe
  email: john@example.com
  phone: 555-123-4567
address:
  street: 123 Main St
  city: Anytown
  zipCode: 12345
preferences:
  theme: dark
`)
		err = os.WriteFile(validNestedYamlPath, validNestedYamlContent, 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Create an invalid YAML file that violates nested property order
		invalidNestedYamlPath := filepath.Join(tempDir, "invalid_nested.yaml")
		invalidNestedYamlContent := []byte(`---
personal:
  email: john@example.com
  name: John Doe  # name should come before email
  phone: 555-123-4567
address:
  street: 123 Main St
  city: Anytown
  zipCode: 12345
preferences:
  theme: dark
`)
		err = os.WriteFile(invalidNestedYamlPath, invalidNestedYamlContent, 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Create another invalid YAML file that violates top-level property order
		invalidOrderYamlPath := filepath.Join(tempDir, "invalid_order.yaml")
		invalidOrderYamlContent := []byte(`---
address:  # address should come after personal
  street: 123 Main St
  city: Anytown
  zipCode: 12345
personal:
  name: John Doe
  email: john@example.com
  phone: 555-123-4567
preferences:
  theme: dark
`)
		err = os.WriteFile(invalidOrderYamlPath, invalidOrderYamlContent, 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Test valid nested YAML
		err = Lint(validNestedYamlPath, nestedSchemaPath)
		if err != nil {
			t.Errorf("Lint() returned an error for valid nested YAML: %v", err)
		}

		// Test invalid nested property order
		err = Lint(invalidNestedYamlPath, nestedSchemaPath)
		if err == nil {
			t.Errorf("Lint() did not return an error for invalid nested property order")
		} else {
			if !strings.Contains(err.Error(), "name") || !strings.Contains(err.Error(), "email") {
				t.Errorf("Lint() returned unexpected error for invalid nested property order: %v", err)
			}
		}

		// Test invalid top-level property order
		err = Lint(invalidOrderYamlPath, nestedSchemaPath)
		if err == nil {
			t.Errorf("Lint() did not return an error for invalid top-level property order")
		} else {
			if !strings.Contains(err.Error(), "personal") || !strings.Contains(err.Error(), "address") {
				t.Errorf("Lint() returned unexpected error for invalid top-level property order: %v", err)
			}
		}
	})

	t.Run("JSON file order preservation", func(t *testing.T) {
		// Create a JSON schema with properties in a specific order
		jsonSchemaPath := filepath.Join(tempDir, "json_order_schema.json")
		jsonSchemaContent := []byte(`{
  "properties": {
    "first": {},
    "second": {},
    "third": {}
  }
}`)
		err := os.WriteFile(jsonSchemaPath, jsonSchemaContent, 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Create a JSON file with properties in the same order as schema
		validJsonPath := filepath.Join(tempDir, "valid_order.json")
		validJsonContent := []byte(`{
  "first": "value1",
  "second": "value2",
  "third": "value3"
}`)
		err = os.WriteFile(validJsonPath, validJsonContent, 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Create a JSON file with properties in a different order
		invalidJsonPath := filepath.Join(tempDir, "invalid_order.json")
		invalidJsonContent := []byte(`{
  "second": "value2",
  "first": "value1",  
  "third": "value3"
}`)
		err = os.WriteFile(invalidJsonPath, invalidJsonContent, 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Test valid JSON order
		err = Lint(validJsonPath, jsonSchemaPath)
		if err != nil {
			t.Errorf("Lint() returned an error for valid JSON order: %v", err)
		}

		// Test invalid JSON order
		err = Lint(invalidJsonPath, jsonSchemaPath)
		if err == nil {
			t.Errorf("Lint() did not return an error for invalid JSON order")
		} else {
			if !strings.Contains(err.Error(), "first") || !strings.Contains(err.Error(), "second") {
				t.Errorf("Lint() returned unexpected error for invalid JSON order: %v", err)
			}
		}

		// Create a nested JSON file with order violations
		nestedInvalidJsonPath := filepath.Join(tempDir, "nested_invalid.json")
		nestedInvalidJsonContent := []byte(`{
  "first": {
    "sub-second": "value",
    "sub-first": "value"
  },
  "second": "value2",
  "third": "value3"
}`)
		err = os.WriteFile(nestedInvalidJsonPath, nestedInvalidJsonContent, 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Create a nested schema that matches the nested JSON structure
		nestedSchemaPath := filepath.Join(tempDir, "nested_json_schema.json")
		nestedSchemaContent := []byte(`{
  "properties": {
    "first": {
      "properties": {
        "sub-first": {},
        "sub-second": {}
      }
    },
    "second": {},
    "third": {}
  }
}`)
		err = os.WriteFile(nestedSchemaPath, nestedSchemaContent, 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Test nested JSON with order violation
		err = Lint(nestedInvalidJsonPath, nestedSchemaPath)
		if err == nil {
			t.Errorf("Lint() did not return an error for nested JSON with order violation")
		} else {
			if !strings.Contains(err.Error(), "sub-first") || !strings.Contains(err.Error(), "sub-second") {
				t.Errorf("Lint() returned unexpected error for nested JSON with order violation: %v", err)
			}
		}
	})
}

func TestExtractSchemaOrderFromPath(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("JSON Schema with ordered properties", func(t *testing.T) {
		// Create JSON file with properties in specific order
		jsonPath := filepath.Join(tempDir, "schema.json")
		jsonContent := []byte(`{
  "properties": {
    "first": {},
    "second": {},
    "third": {}
  }
}`)
		err := os.WriteFile(jsonPath, jsonContent, 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		keys, err := extractSchemaOrderFromJsonSchemaPath(jsonPath)
		if err != nil {
			t.Errorf("extractSchemaOrderFromPath() returned an error for valid JSON: %v", err)
		}

		expected := []string{"first", "second", "third"}
		if !reflect.DeepEqual(keys, expected) {
			t.Errorf("extractSchemaOrderFromPath() returned incorrect order: got %v, expected %v", keys, expected)
		}
	})

	t.Run("Invalid file format", func(t *testing.T) {
		invalidPath := filepath.Join(tempDir, "invalid.txt")
		invalidContent := []byte(`Not a valid YAML or JSON file`)
		err := os.WriteFile(invalidPath, invalidContent, 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		keys, err := extractSchemaOrderFromJsonSchemaPath(invalidPath)
		if err == nil {
			t.Errorf("extractSchemaOrderFromPath() did not return an error for invalid file format")
		}

		if keys != nil {
			t.Errorf("extractSchemaOrderFromPath() returned non-nil keys for invalid file format: %v", keys)
		}
	})

	t.Run("JSON Schema with nested properties", func(t *testing.T) {
		// Create JSON file with nested properties
		nestedJsonPath := filepath.Join(tempDir, "nested_schema.json")
		nestedJsonContent := []byte(`{
  "properties": {
    "personal": {
      "properties": {
        "name": {},
        "email": {},
        "phone": {}
      }
    },
    "address": {
      "properties": {
        "street": {},
        "city": {},
        "zipCode": {}
      }
    },
    "preferences": {}
  }
}`)
		err := os.WriteFile(nestedJsonPath, nestedJsonContent, 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Call the function that will need to be modified to handle nested properties
		schema, err := extractSchemaOrderFromJsonSchemaPath(nestedJsonPath)
		if err != nil {
			t.Errorf("extractSchemaOrderFromJsonSchemaPath() returned an error for valid nested JSON: %v", err)
		}

		// For now, we're only checking top-level properties until we modify the function
		expectedTopLevel := []string{"personal", "address", "preferences"}
		if !reflect.DeepEqual(schema, expectedTopLevel) {
			t.Errorf("extractSchemaOrderFromJsonSchemaPath() returned incorrect top-level order: got %v, expected %v",
				schema, expectedTopLevel)
		}

		// Future test after we enhance the function to handle nested properties
		// The test will be updated once we modify the function to return a nested structure
	})

	t.Run("Nested schema extraction", func(t *testing.T) {
		// Create JSON file with nested properties
		nestedJsonPath := filepath.Join(tempDir, "nested_schema_full.json")
		nestedJsonContent := []byte(`{
  "properties": {
    "personal": {
      "properties": {
        "name": {},
        "email": {},
        "phone": {}
      }
    },
    "address": {
      "properties": {
        "street": {},
        "city": {},
        "zipCode": {}
      }
    },
    "preferences": {}
  }
}`)
		err := os.WriteFile(nestedJsonPath, nestedJsonContent, 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Test extractNestedSchemaOrder
		properties, err := extractNestedSchemaOrder(nestedJsonPath)
		if err != nil {
			t.Errorf("extractNestedSchemaOrder() returned an error for valid nested JSON: %v", err)
		}

		// Verify the top-level properties
		if len(properties) != 3 {
			t.Errorf("extractNestedSchemaOrder() returned incorrect number of top-level properties: got %d, expected %d",
				len(properties), 3)
		}

		// Verify property names and order
		expectedNames := []string{"personal", "address", "preferences"}
		for i, prop := range properties {
			if prop.Name != expectedNames[i] {
				t.Errorf("Property at index %d has incorrect name: got %s, expected %s", i, prop.Name, expectedNames[i])
			}
		}

		// Verify nested properties for "personal"
		personalNestedProps := properties[0].Properties
		if len(personalNestedProps) != 3 {
			t.Errorf("'personal' property has incorrect number of nested properties: got %d, expected %d",
				len(personalNestedProps), 3)
		} else {
			expectedPersonalSubprops := []string{"name", "email", "phone"}
			for i, prop := range personalNestedProps {
				if prop.Name != expectedPersonalSubprops[i] {
					t.Errorf("'personal' subproperty at index %d has incorrect name: got %s, expected %s",
						i, prop.Name, expectedPersonalSubprops[i])
				}
			}
		}

		// Verify nested properties for "address"
		addressNestedProps := properties[1].Properties
		if len(addressNestedProps) != 3 {
			t.Errorf("'address' property has incorrect number of nested properties: got %d, expected %d",
				len(addressNestedProps), 3)
		} else {
			expectedAddressSubprops := []string{"street", "city", "zipCode"}
			for i, prop := range addressNestedProps {
				if prop.Name != expectedAddressSubprops[i] {
					t.Errorf("'address' subproperty at index %d has incorrect name: got %s, expected %s",
						i, prop.Name, expectedAddressSubprops[i])
				}
			}
		}

		// Verify "preferences" has no nested properties
		if len(properties[2].Properties) > 0 {
			t.Errorf("'preferences' property should not have nested properties, but has %d",
				len(properties[2].Properties))
		}
	})

	t.Run("Deeply nested schema", func(t *testing.T) {
		// Create JSON file with multiple levels of nested properties
		deeplyNestedJsonPath := filepath.Join(tempDir, "deeply_nested_schema.json")
		deeplyNestedJsonContent := []byte(`{
  "properties": {
    "user": {
      "properties": {
        "profile": {
          "properties": {
            "personalInfo": {
              "properties": {
                "firstName": {},
                "lastName": {},
                "middleName": {}
              }
            },
            "contactInfo": {
              "properties": {
                "email": {},
                "phone": {
                  "properties": {
                    "home": {},
                    "mobile": {},
                    "work": {}
                  }
                }
              }
            }
          }
        },
        "settings": {
          "properties": {
            "notifications": {},
            "privacy": {}
          }
        }
      }
    },
    "metadata": {}
  }
}`)
		err := os.WriteFile(deeplyNestedJsonPath, deeplyNestedJsonContent, 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Test extractNestedSchemaOrder
		properties, err := extractNestedSchemaOrder(deeplyNestedJsonPath)
		if err != nil {
			t.Errorf("extractNestedSchemaOrder() returned an error for deeply nested JSON: %v", err)
		}

		// Verify top-level properties
		if len(properties) != 2 {
			t.Errorf("extractNestedSchemaOrder() returned incorrect number of top-level properties: got %d, expected %d",
				len(properties), 2)
		}

		// Navigate to "user.profile.personalInfo" and verify order
		userProp := properties[0] // "user"
		if userProp.Name != "user" || len(userProp.Properties) != 2 {
			t.Errorf("Expected 'user' property with 2 sub-properties, got %s with %d sub-properties",
				userProp.Name, len(userProp.Properties))
		} else {
			profileProp := userProp.Properties[0] // "profile"
			if profileProp.Name != "profile" || len(profileProp.Properties) != 2 {
				t.Errorf("Expected 'profile' property with 2 sub-properties, got %s with %d sub-properties",
					profileProp.Name, len(profileProp.Properties))
			} else {
				personalInfoProp := profileProp.Properties[0] // "personalInfo"
				if personalInfoProp.Name != "personalInfo" || len(personalInfoProp.Properties) != 3 {
					t.Errorf("Expected 'personalInfo' property with 3 sub-properties, got %s with %d sub-properties",
						personalInfoProp.Name, len(personalInfoProp.Properties))
				} else {
					// Verify order of personal info properties
					expectedNames := []string{"firstName", "lastName", "middleName"}
					for i, prop := range personalInfoProp.Properties {
						if prop.Name != expectedNames[i] {
							t.Errorf("personalInfo sub-property at index %d has incorrect name: got %s, expected %s",
								i, prop.Name, expectedNames[i])
						}
					}
				}

				// Check nested phone properties under contactInfo
				contactInfoProp := profileProp.Properties[1] // "contactInfo"
				if contactInfoProp.Name != "contactInfo" || len(contactInfoProp.Properties) != 2 {
					t.Errorf("Expected 'contactInfo' property with 2 sub-properties, got %s with %d sub-properties",
						contactInfoProp.Name, len(contactInfoProp.Properties))
				} else {
					phoneProp := contactInfoProp.Properties[1] // "phone"
					if phoneProp.Name != "phone" || len(phoneProp.Properties) != 3 {
						t.Errorf("Expected 'phone' property with 3 sub-properties, got %s with %d sub-properties",
							phoneProp.Name, len(phoneProp.Properties))
					} else {
						// Verify order of phone properties
						expectedPhoneProps := []string{"home", "mobile", "work"}
						for i, prop := range phoneProp.Properties {
							if prop.Name != expectedPhoneProps[i] {
								t.Errorf("phone sub-property at index %d has incorrect name: got %s, expected %s",
									i, prop.Name, expectedPhoneProps[i])
							}
						}
					}
				}
			}
		}
	})

	t.Run("Missing properties", func(t *testing.T) {
		noPropsPath := filepath.Join(tempDir, "no_props.json")
		noPropsContent := []byte(`{"type": "object"}`)
		err := os.WriteFile(noPropsPath, noPropsContent, 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		keys, err := extractSchemaOrderFromJsonSchemaPath(noPropsPath)
		if err == nil {
			t.Errorf("extractSchemaOrderFromPath() did not return an error for schema without properties")
		}

		if keys != nil {
			t.Errorf("extractSchemaOrderFromPath() returned non-nil keys for schema without properties: %v", keys)
		}
	})

	t.Run("Invalid nested schema", func(t *testing.T) {
		// Create a malformed schema with invalid nesting
		invalidNestedPath := filepath.Join(tempDir, "invalid_nested.json")
		invalidNestedContent := []byte(`{
  "properties": {
    "field1": {
      "properties": "this should be an object not a string"
    }
  }
}`)
		err := os.WriteFile(invalidNestedPath, invalidNestedContent, 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		properties, err := extractNestedSchemaOrder(invalidNestedPath)
		if err == nil {
			t.Errorf("extractNestedSchemaOrder() did not return an error for invalid nested structure")
		}

		if properties != nil {
			t.Errorf("extractNestedSchemaOrder() returned non-nil properties for invalid nested structure: %v", properties)
		}
	})
}
