// Copyright 2025 NVIDIA Corporation
// SPDX-License-Identifier: Apache-2.0

package tests

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"terraform/terraform-provider/provider"
)

// TestAttributeDefaults verifies that the SetAttributeDefaultValue function
// returns proper defaults based on attribute type and explicit defaults
func TestAttributeDefaults(t *testing.T) {
	// Test cases structure
	type testCase struct {
		name     string
		attrMap  map[string]interface{}
		expected interface{}
	}

	// Define test cases
	tests := []testCase{
		{
			name: "explicit default value",
			attrMap: map[string]interface{}{
				"type":    "string",
				"default": "test-value",
			},
			expected: "test-value",
		},
		{
			name: "no default, string type",
			attrMap: map[string]interface{}{
				"type": "string",
			},
			expected: "",
		},
		{
			name: "no default, bool type",
			attrMap: map[string]interface{}{
				"type": "bool",
			},
			expected: false,
		},
		{
			name: "no default, int type",
			attrMap: map[string]interface{}{
				"type": "int",
			},
			expected: int(0),
		},
		{
			name: "no default, float type",
			attrMap: map[string]interface{}{
				"type": "float",
			},
			expected: float64(0),
		},
		// Defaults are not set for lists/sets anymore
		// {
		// 	name: "no default, string[] type",
		// 	attrMap: map[string]interface{}{
		// 		"type": "string[]",
		// 	},
		// 	expected: []string{},
		// },
		// {
		// 	name: "no default, string_set type",
		// 	attrMap: map[string]interface{}{
		// 		"type": "string_set",
		// 	},
		// 	expected: []string{},
		// },
		{
			name: "no default, intbool type",
			attrMap: map[string]interface{}{
				"type": "intbool",
			},
			expected: false,
		},
		{
			name: "no default, command type",
			attrMap: map[string]interface{}{
				"type": "command",
			},
			expected: "",
		},
		{
			name: "no default, time_s type",
			attrMap: map[string]interface{}{
				"type": "time_s",
			},
			expected: "",
		},
		{
			name: "no default, label type",
			attrMap: map[string]interface{}{
				"type": "label",
			},
			expected: "",
		},
		{
			name: "no default, resource type",
			attrMap: map[string]interface{}{
				"type": "resource",
			},
			expected: "",
		},
		{
			name: "no default, b64json type",
			attrMap: map[string]interface{}{
				"type": "b64json",
			},
			expected: "",
		},
		{
			name: "no default, unsigned type",
			attrMap: map[string]interface{}{
				"type": "unsigned",
			},
			expected: uint(0), // Note: AttrValueDefault returns uint, but comparison might need adjustment if SetAttributeDefaultValue differs
		},
		{
			name:     "no type specified, default to string",
			attrMap:  map[string]interface{}{},
			expected: "",
		},
		{
			name: "required field, no default set",
			attrMap: map[string]interface{}{
				"type":     "string",
				"required": true,
			},
			expected: nil, // Expect nil because default shouldn't be set
		},
		{
			name: "computed field, no default set",
			attrMap: map[string]interface{}{
				"type":     "string",
				"computed": true,
			},
			expected: nil, // Expect nil because default shouldn't be set
		},
		{
			name: "string[] type, no default set",
			attrMap: map[string]interface{}{
				"type": "string[]",
			},
			expected: nil, // Expect nil because default shouldn't be set for lists
		},
		{
			name: "string_set type, no default set",
			attrMap: map[string]interface{}{
				"type": "string_set",
			},
			expected: nil, // Expect nil because default shouldn't be set for sets
		},
	}

	// Run tests
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Call SetAttributeDefaultValue and get the calculated default value (second return)
			dummySchema := &schema.Schema{}
			_, result := provider.SetAttributeDefaultValue(tc.attrMap, dummySchema)

			// Check if the result matches the expected value
			// Special check for nil expected value
			if tc.expected == nil {
				if result != nil {
					t.Errorf("Expected nil, got %v (%T)", result, result)
				}
			} else if result != tc.expected {
				// Handle type mismatches for unsigned, as AttrValueDefault returns uint
				if expectedUint, ok := tc.expected.(uint); ok {
					if resultInt, ok := result.(int); ok && uint(resultInt) == expectedUint {
						// Allow int(0) to match uint(0) for the unsigned test case if necessary
					} else {
						t.Errorf("Expected %v (%T), got %v (%T)", tc.expected, tc.expected, result, result)
					}
				} else {
					t.Errorf("Expected %v (%T), got %v (%T)", tc.expected, tc.expected, result, result)
				}
			}
		})
	}
}

// TestResourceTypeAttributeDefaults verifies default value handling across
// all resource types defined in provider_conf.go, focusing on the fallback
// to type-specific defaults when no explicit default is provided.
func TestResourceTypeAttributeDefaults(t *testing.T) {
	// Get the provider config using helper function
	providerConfig := GetProviderConfig(t)

	// Test resources using the global variable
	for _, resType := range SupportedResourceTypes {
		t.Run(resType, func(t *testing.T) {
			resourceMap := GetResourceConfig(t, providerConfig, resType)
			if resourceMap == nil {
				return // Skip if resource type not found
			}

			attributes := GetResourceAttributes(t, resourceMap, resType)

			// Test each attribute in the resource
			for attrName, attrConfig := range attributes {
				attrMap, ok := attrConfig.(map[string]interface{})
				if !ok {
					t.Fatalf("Attribute %s config is not a map", attrName)
				}

				// Skip attributes with explicit defaults set in the config
				if _, hasDefault := attrMap["default"]; hasDefault {
					continue
				}
				// Skip computed or required fields as they won't get defaults
				if computed, _ := attrMap["computed"].(bool); computed {
					continue
				}
				if required, _ := attrMap["required"].(bool); required {
					continue
				}

				attrType, ok := attrMap["type"].(string)
				if !ok {
					t.Fatalf("Attribute %s type is not a string", attrName)
				}

				// Skip list/set types as they don't get defaults
				if attrType == "string[]" || attrType == "string_set" {
					continue
				}

				// Call SetAttributeDefaultValue and get the calculated default value
				dummySchema := &schema.Schema{}
				_, result := provider.SetAttributeDefaultValue(attrMap, dummySchema)

				// Get the expected type-specific default
				expected := provider.AttrValueDefault(attrType)

				// Compare the result from SetAttributeDefaultValue with the type-specific default
				if result != expected {
					// Handle type mismatches for unsigned, as AttrValueDefault returns uint
					if expectedUint, ok := expected.(uint); ok {
						if resultInt, ok := result.(int); ok && uint(resultInt) == expectedUint {
							// Allow int(0) to match uint(0) for the unsigned test case if necessary
						} else {
							t.Errorf("Resource %s, attribute %s (%s): Expected type-default %v (%T), got %v (%T)",
								resType, attrName, attrType, expected, expected, result, result)
						}
					} else {
						t.Errorf("Resource %s, attribute %s (%s): Expected type-default %v (%T), got %v (%T)",
							resType, attrName, attrType, expected, expected, result, result)
					}
				}
			}
		})
	}
}
