// Copyright 2025 NVIDIA Corporation
// SPDX-License-Identifier: Apache-2.0

package tests

import (
	"encoding/json"
	"testing"

	"terraform/terraform-provider/provider"
)

// SupportedResourceTypes contains all resource types supported by the provider
// Update this list when adding new resource types to ensure consistent testing
var SupportedResourceTypes = []string{
	"action",
	"alarm",
	"time_trigger",
	"bot",
	"file",
	"integration",
	"notebook",
	"principal",
	"resource",
	"system_settings",
	"report_template",
	"dashboard",
}

// GetProviderConfig parses the provider configuration JSON into a map
func GetProviderConfig(t *testing.T) map[string]interface{} {
	var providerConfig map[string]interface{}
	err := json.Unmarshal([]byte(provider.ObjectConfigJsonStr), &providerConfig)
	if err != nil {
		t.Fatalf("Failed to parse provider config: %v", err)
	}
	return providerConfig
}

// GetResourceConfig returns the configuration for a specific resource type
func GetResourceConfig(t *testing.T, providerConfig map[string]interface{}, resType string) map[string]interface{} {
	resourceConfig, ok := providerConfig[resType]
	if !ok {
		t.Skipf("Resource type %s not found in provider config", resType)
		return nil
	}

	resourceMap, ok := resourceConfig.(map[string]interface{})
	if !ok {
		t.Fatalf("Resource config for %s is not a map", resType)
	}

	return resourceMap
}

// GetResourceAttributes returns the attributes map for a resource
func GetResourceAttributes(t *testing.T, resourceMap map[string]interface{}, resType string) map[string]interface{} {
	attributes, ok := resourceMap["attributes"].(map[string]interface{})
	if !ok {
		t.Fatalf("Attributes for %s is not a map", resType)
	}

	return attributes
}

// GetAttributeMap returns the attribute configuration map for a specific attribute
func GetAttributeMap(t *testing.T, attributes map[string]interface{}, attrName string) map[string]interface{} {
	attrMap, ok := attributes[attrName].(map[string]interface{})
	if !ok {
		t.Fatalf("Attribute %s config is not a map", attrName)
	}

	return attrMap
}
