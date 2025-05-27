// Copyright 2025 NVIDIA Corporation
// SPDX-License-Identifier: Apache-2.0

package tests

import (
	"testing"

	"terraform/terraform-provider/provider"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// Helper function to create a test resource
func createTestResource(t *testing.T, resourceType string) *schema.Resource {
	// Use provider config helper
	providerConfigMap := GetProviderConfig(t)

	_, ok := providerConfigMap[resourceType]
	if !ok {
		t.Fatalf("Resource type %s not found in provider config", resourceType)
	}

	// Create the resource
	resource := provider.ResourceObject(provider.ObjectConfigJsonStr, resourceType)
	return resource
}

// TestSetAttributeDefaultValueBehavior tests the behavior of SetAttributeDefaultValue
// focusing on explicit defaults and conditions where no default should be set.
func TestSetAttributeDefaultValueBehavior(t *testing.T) {
	// Mock attribute map configurations for testing
	testAttributeMaps := map[string]map[string]interface{}{
		"string_attr_with_default": {
			"type":    "string",
			"default": "custom-default",
		},
		"string_attr_no_default": {
			"type": "string",
		},
		"bool_attr_with_default": {
			"type":    "bool",
			"default": true,
		},
		"bool_attr_no_default": {
			"type": "bool",
		},
		"int_attr_with_default": {
			"type":    "int",
			"default": 42,
		},
		"int_attr_no_default": {
			"type": "int",
		},
		"required_attr": {
			"type":     "string",
			"required": true,
		},
		"computed_attr": {
			"type":     "string",
			"computed": true,
		},
		"list_attr": {
			"type": "string[]",
		},
		"set_attr": {
			"type": "string_set",
		},
	}

	for attrName, attrMap := range testAttributeMaps {
		t.Run(attrName, func(t *testing.T) {
			// Call SetAttributeDefaultValue
			dummySchema := &schema.Schema{}
			_, defaultVal := provider.SetAttributeDefaultValue(attrMap, dummySchema)

			// Check conditions where default should NOT be set
			if required, _ := attrMap["required"].(bool); required {
				if defaultVal != nil {
					t.Errorf("%s (required): Expected nil default, got %v", attrName, defaultVal)
				}
				return // End test for this case
			}
			if computed, _ := attrMap["computed"].(bool); computed {
				if defaultVal != nil {
					t.Errorf("%s (computed): Expected nil default, got %v", attrName, defaultVal)
				}
				return // End test for this case
			}
			attrType := attrMap["type"].(string)
			if attrType == "string[]" || attrType == "string_set" {
				if defaultVal != nil {
					t.Errorf("%s (list/set): Expected nil default, got %v", attrName, defaultVal)
				}
				return // End test for this case
			}

			// Verify explicit defaults are preserved
			if explicitDefault, hasDefault := attrMap["default"]; hasDefault {
				if defaultVal != explicitDefault {
					t.Errorf("%s: Expected explicit default %v, got %v",
						attrName, explicitDefault, defaultVal)
				}
			} else {
				// Verify type-based defaults for attributes without explicit defaults
				expectedDefault := provider.AttrValueDefault(attrType)
				if defaultVal != expectedDefault {
					// Handle type mismatches for unsigned, as AttrValueDefault returns uint
					if expectedUint, ok := expectedDefault.(uint); ok {
						if resultInt, ok := defaultVal.(int); ok && uint(resultInt) == expectedUint {
							// Allow int(0) to match uint(0) if necessary
						} else {
							t.Errorf("%s: Expected type default %v (%T), got %v (%T)",
								attrName, expectedDefault, expectedDefault, defaultVal, defaultVal)
						}
					} else {
						t.Errorf("%s: Expected type default %v (%T), got %v (%T)",
							attrName, expectedDefault, expectedDefault, defaultVal, defaultVal)
					}
				}
			}
		})
	}
}

// TestResourceSchemaDefaults tests that the schema for resources
// has correct defaults set (or not set) based on provider configuration and rules.
func TestResourceSchemaDefaults(t *testing.T) {
	// Using the global SupportedResourceTypes
	for _, resType := range SupportedResourceTypes {
		t.Run(resType, func(t *testing.T) {
			resource := createTestResource(t, resType)

			// Get provider config using helper function
			providerConfigMap := GetProviderConfig(t)
			resourceConfig := GetResourceConfig(t, providerConfigMap, resType)
			attributes := GetResourceAttributes(t, resourceConfig, resType)

			// Verify schema fields have correct defaults
			for fieldName, schemaField := range resource.Schema {
				attrConfig, hasAttr := attributes[fieldName]
				if !hasAttr {
					continue // Field might be added by provider, not in config
				}

				attrMap, ok := attrConfig.(map[string]interface{})
				if !ok {
					continue
				}

				// Determine if a default *should* be set based on rules
				shouldHaveDefault := true
				if required, _ := attrMap["required"].(bool); required {
					shouldHaveDefault = false
				}
				if computed, _ := attrMap["computed"].(bool); computed {
					shouldHaveDefault = false
				}
				attrType, _ := attrMap["type"].(string)
				if attrType == "string[]" || attrType == "string_set" {
					shouldHaveDefault = false
				}

				if shouldHaveDefault {
					// Check if field has explicit default in config
					explicitDefault, hasDefault := attrMap["default"]
					var expectedDefault interface{}
					if hasDefault {
						expectedDefault = explicitDefault
					} else {
						expectedDefault = provider.AttrValueDefault(attrType)
					}

					if schemaField.Default == nil {
						t.Errorf("Resource %s field %s: schema default is nil, but expected %v",
							resType, fieldName, expectedDefault)
					} else if schemaField.Default != expectedDefault {
						// Handle type mismatches for unsigned
						if expectedUint, ok := expectedDefault.(uint); ok {
							if resultInt, ok := schemaField.Default.(int); ok && uint(resultInt) == expectedUint {
								// Allow int(0) to match uint(0) if necessary
							} else {
								t.Errorf("Resource %s field %s: schema default %v (%T) doesn't match expected default %v (%T)",
									resType, fieldName, schemaField.Default, schemaField.Default, expectedDefault, expectedDefault)
							}
						} else {
							t.Errorf("Resource %s field %s: schema default %v (%T) doesn't match expected default %v (%T)",
								resType, fieldName, schemaField.Default, schemaField.Default, expectedDefault, expectedDefault)
						}
					}
				} else {
					// Field should NOT have a default (required, computed, list, set)
					if schemaField.Default != nil {
						t.Errorf("Resource %s field %s: schema default is %v, but expected nil (field is required/computed/list/set)",
							resType, fieldName, schemaField.Default)
					}
				}
			}
		})
	}
}
