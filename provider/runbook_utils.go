// Copyright 2025 NVIDIA Corporation
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// buildRunbookDataObject builds a JSON containing the core runbook data. expects at least "cells" to be present
func buildRunbookDataObject(d *schema.ResourceData, cells interface{}) (interface{}, error) {
	runbookData := map[string]interface{}{}

	cellsData, err := buildCellsData(cells)
	if err != nil {
		return nil, err
	}
	runbookData["cells"] = cellsData

	// TODO this should be passed in, it might not always come from ResourceData
	params, exists := d.GetOk("params")
	appendActionLog(fmt.Sprintf("calling buildParametersData (exists:%v) from: %v\n", exists, params))
	if exists {
		params = CastToObject(params)
	}
	paramsData, err := buildParametersData(params, exists)
	if err != nil {
		return nil, err
	}
	runbookData["params"] = paramsData

	externalParametersData, err := buildExternalParametersData(d)
	if err != nil {
		return nil, err
	}
	runbookData["external_params"] = externalParametersData

	enabled, exists := d.GetOk("enabled")
	if !exists {
		enabled = true
	}
	runbookData["enabled"] = enabled

	// return the json encoded runbookData
	encodedRunbookData, err := json.Marshal(runbookData)
	if err != nil {
		return nil, fmt.Errorf("error encoding runbook data: %v", err)
	}
	return string(encodedRunbookData), nil
}

func buildCellsData(cells interface{}) (interface{}, error) {

	var decodedCells []interface{}
	if cellsList, ok := cells.([]interface{}); ok {
		decodedCells = cellsList
	} else {
		return nil, fmt.Errorf(`runbook cells must be a list`)
	}
	cellsData := []interface{}{}

	appendActionLog(fmt.Sprintf("building runbook cells from: %v\n", cells))

	for _, cell := range decodedCells {
		markdownContent := GetNestedValueOrDefault(cell, ToKeyPath("md"), nil)
		oplangContent := GetNestedValueOrDefault(cell, ToKeyPath("op"), nil)
		if markdownContent == nil && oplangContent == nil {
			return nil, fmt.Errorf(`runbook cell must specify either an oplang command or markdown content using the "md" or "op" fields`)
		}
		if markdownContent != nil && oplangContent != nil {
			return nil, fmt.Errorf("runbook cell cannot have both markdown and oplang content")
		}

		enabled, enOk := GetNestedValueOrDefault(cell, ToKeyPath("enabled"), true).(bool)
		if !enOk {
			return nil, fmt.Errorf(`runbook cell 'enabled' must be a boolean (or not set).`)
		}

		name, enOk := GetNestedValueOrDefault(cell, ToKeyPath("name"), "unnamed").(string)
		if !enOk {
			return nil, fmt.Errorf(`runbook cell 'name' must be a string (or not set).`)
		}

		secretAware := GetNestedValueOrDefault(cell, ToKeyPath("secret_aware"), nil)

		cellContent, err := GetCellContent(markdownContent, oplangContent, enabled, secretAware, name)
		if err != nil {
			return nil, err
		}

		cellsData = append(cellsData, cellContent)
	}

	return cellsData, nil
}

func GetCellContent(markdownContent interface{}, oplangContent interface{}, enabled bool, secretAware interface{}, name string) (map[string]interface{}, error) {
	var cellContent map[string]interface{}
	if markdownContent != nil {
		if _, ok := markdownContent.(string); !ok {
			return nil, fmt.Errorf(`runbook cell markdown must be a string`)
		}

		cellContent = map[string]interface{}{
			"content": markdownContent,
			"enabled": enabled,
			"type":    "MARKDOWN",
			"name":    name,
		}
	} else {
		if _, ok := oplangContent.(string); !ok {
			return nil, fmt.Errorf(`runbook cell oplang must be a string`)
		}

		cellContent = map[string]interface{}{
			"content": oplangContent,
			"enabled": enabled,
			"type":    "OP_LANG",
			"name":    name,
		}
	}

	// Only add secret_aware if the backend version supports it
	backendVersion := GetBackendVersionInfoStruct()
	if IsSecretAwareSupported(backendVersion) {
		if secretAware != nil {
			if _, ok := secretAware.(bool); !ok {
				return nil, fmt.Errorf(`runbook cell 'secret_aware' must be a boolean (or not set).`)
			}

			cellContent["secret_aware"] = secretAware.(bool)
		}
	}

	return cellContent, nil
}

func buildParametersData(params interface{}, exists bool) ([]interface{}, error) {
	appendActionLog(fmt.Sprintf("building runbook params (exists:%v) from: %v\n", exists, params))

	paramsOut := []interface{}{}
	paramsArray, ok := params.([]interface{})
	if !exists {
		return []interface{}{}, nil
	}
	if !ok {
		return nil, fmt.Errorf("error notebook params is not an object array.")
	}

	for _, parameter := range paramsArray {
		name, ok := GetNestedValueOrDefault(parameter, ToKeyPath("name"), nil).(string)
		if !ok {
			return nil, fmt.Errorf("parameter name is required string")
		}
		required := CastToBool(GetNestedValueOrDefault(parameter, ToKeyPath("required"), true))
		value := CastToString(GetNestedValueOrDefault(parameter, ToKeyPath("value"), ""))
		export := CastToBool(GetNestedValueOrDefault(parameter, ToKeyPath("export"), false))
		description := CastToString(GetNestedValueOrDefault(parameter, ToKeyPath("description"), ""))

		parameterData := map[string]interface{}{
			"export":      export, // false by default
			"name":        name,
			"required":    required,    // true by default
			"value":       value,       // empty string by default
			"description": description, // empty string by default
		}

		paramsOut = append(paramsOut, parameterData)
	}

	return paramsOut, nil
}

func buildExternalParametersData(d *schema.ResourceData) ([]interface{}, error) {
	var decodedExternalParameters []interface{}
	externalParametersData := []interface{}{}
	externalParameters, exists := d.GetOk("external_params")

	appendActionLog(fmt.Sprintf("building runbook external params from: %v\n", externalParameters))

	if !exists {
		return []interface{}{}, nil
	}

	err := json.Unmarshal([]byte(externalParameters.(string)), &decodedExternalParameters)
	if err != nil {
		return nil, fmt.Errorf("error decoding external params: %v", err)
	}

	for _, externalParameter := range decodedExternalParameters {
		name, exists := externalParameter.(map[string]interface{})["name"]
		if !exists {
			return nil, fmt.Errorf("external parameter name is required")
		}
		source, exists := externalParameter.(map[string]interface{})["source"]
		if !exists {
			return nil, fmt.Errorf("external parameter source is required")
		}
		value, exists := externalParameter.(map[string]interface{})["value"]
		if !exists {
			value = ""
		}
		jsonPath, exists := externalParameter.(map[string]interface{})["json_path"]
		if !exists {
			jsonPath = ""
		}
		export, exists := externalParameter.(map[string]interface{})["export"]
		if !exists {
			export = false
		}

		description, exists := externalParameter.(map[string]interface{})["description"]
		if !exists {
			description = ""
		}

		externalParameterData := map[string]interface{}{
			"name":        name,
			"source":      source,
			"value":       value,
			"json_path":   jsonPath,
			"export":      export,
			"description": description,
		}

		externalParametersData = append(externalParametersData, externalParameterData)
	}

	return externalParametersData, nil
}
