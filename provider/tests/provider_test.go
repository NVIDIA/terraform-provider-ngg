// Copyright 2025 NVIDIA Corporation
// SPDX-License-Identifier: Apache-2.0

package tests

import (
	"fmt"
	"terraform/terraform-provider/provider"
	"testing"
)

func TestGetBackendVersionInfo(t *testing.T) {
	// Reset global state
	provider.ResetGlobalBackendVersionForTesting()
	apiCallCount := 0

	// Setup mock
	provider.SetRunOpCommandToJsonFuncForTesting(func(command string) (map[string]interface{}, error) {
		apiCallCount++
		return map[string]interface{}{
			"get_backend_version": `{"tag": "release-1.2.3-test", "build_date": "Wed_May_18_00:07:11_UTC_2022"}`,
		}, nil
	})
	defer provider.ResetRunOpCommandToJsonFuncForTesting()

	// First call should make API call
	build1, version1, major1, minor1, patch1, err1 := provider.GetBackendVersionInfo()

	if apiCallCount != 1 {
		t.Errorf("Expected 1 API call, got %d", apiCallCount)
	}
	if err1 != nil {
		t.Errorf("Expected no error, got %v", *err1)
	}
	if version1 != "release-1.2.3-test" {
		t.Errorf("Expected version 'release-1.2.3-test', got '%s'", version1)
	}
	if major1 != 1 || minor1 != 2 || patch1 != 3 {
		t.Errorf("Expected version parts 1.2.3, got %d.%d.%d", major1, minor1, patch1)
	}

	// Second call should use cached data (no additional API call)
	build2, version2, major2, minor2, patch2, err2 := provider.GetBackendVersionInfo()

	if apiCallCount != 1 {
		t.Errorf("Expected still 1 API call after using cache, got %d", apiCallCount)
	}
	if build1 != build2 || version1 != version2 || major1 != major2 || minor1 != minor2 || patch1 != patch2 {
		t.Error("Cached call should return same values")
	}
	if err2 != err1 {
		t.Error("Cached call should return same error state")
	}
}

func TestGetBackendVersionInfoWithError(t *testing.T) {
	// Reset global state
	provider.ResetGlobalBackendVersionForTesting()
	apiCallCount := 0

	// Setup mock that returns error
	provider.SetRunOpCommandToJsonFuncForTesting(func(command string) (map[string]interface{}, error) {
		apiCallCount++
		return nil, fmt.Errorf("API connection failed")
	})
	defer provider.ResetRunOpCommandToJsonFuncForTesting()

	// First call should make API call and return error
	build1, version1, major1, minor1, patch1, err1 := provider.GetBackendVersionInfo()

	if apiCallCount != 1 {
		t.Errorf("Expected 1 API call, got %d", apiCallCount)
	}
	if err1 == nil {
		t.Error("Expected error but got nil")
	}
	if build1 != "unknown" || version1 != "unknown" {
		t.Errorf("Expected unknown values on error, got build='%s', version='%s'", build1, version1)
	}
	if major1 != 0 || minor1 != 0 || patch1 != 0 {
		t.Errorf("Expected zero version parts on error, got %d.%d.%d", major1, minor1, patch1)
	}

	// Second call should retry API call (errors are not cached)
	_, _, _, _, _, err2 := provider.GetBackendVersionInfo()

	if apiCallCount != 2 {
		t.Errorf("Expected 2 API calls after error (retry), got %d", apiCallCount)
	}
	if err2 == nil {
		t.Error("Expected error on retry but got nil")
	}
}
