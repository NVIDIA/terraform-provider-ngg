// Copyright 2025 NVIDIA Corporation
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	//"regexp"

	"os"
	"testing"

	//"github.com/hashicorp/terraform-plugin-sdk/acctest"
	//"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"math/rand"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func getProviderConfigString() string {
	url, _ := os.LookupEnv(EnvVarsNamePrefix + "_URL")

	return `
	provider "` + ProviderShortName + `" {
		url = "` + url + `"
		retries = 2
		debug = true
	}
`
}

// providerFactories are used to instantiate a provider during acceptance testing.
// The factory function will be invoked for every Terraform CLI command executed
// to create a provider server to which the CLI can reattach.
var providerFactories = map[string]func() (*schema.Provider, error){
	ProviderShortName: func() (*schema.Provider, error) {
		return New("dev")(), nil
	},
}

func TestProvider(t *testing.T) {
	if err := New("dev")().InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

//func TestMain(m *testing.M) {
//  acctest.UseBinaryDriver(ProviderShortName, New("dev"))
//  resource.TestMain(m)
//}

func testAccPreCheck(t *testing.T) {
	// You can add code here to run prior to any test case execution, for example assertions
	// about the appropriate environment variables being set are common to see in a pre-check
	// function.
}

func RandStringFromCharSet(strlen int, charSet string) string {
	result := make([]byte, strlen)
	for i := 0; i < strlen; i++ {
		result[i] = charSet[rand.Intn(len(charSet))]
	}
	return string(result)
}

func RandomAlphaPrefix(slen int) string {
	alpha := "abcdefghijklmnopqrstuvwxyz"
	return RandStringFromCharSet(slen, alpha)
}

////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////
// Action

func TestAccResourceAction(t *testing.T) {
	//t.Skip("resource not yet implemented, remove this once you add your own code")
	pre := RandomAlphaPrefix(5)

	resource.UnitTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: getProviderConfigString() + buildMockAccResourceAction(pre, false),
				Check: resource.ComposeTestCheckFunc(
					//resource.TestMatchResourceAttr( ProviderShortName + "_action.ls_action", "name", regexp.MustCompile("^ba")),
					resource.TestCheckResourceAttr(ProviderShortName+"_action."+pre+"_ls_action", "name", pre+"_ls_action"),
					resource.TestCheckResourceAttr(ProviderShortName+"_action."+pre+"_ls_action", "command", "`ls ${dir}; export FOO='bar'`"),
					resource.TestCheckResourceAttr(ProviderShortName+"_action."+pre+"_ls_action", "description", "List some files"),
					resource.TestCheckResourceAttr(ProviderShortName+"_action."+pre+"_ls_action", "enabled", "true"),
					// FIXME: there array type attribute can be checked correctly.
					//resource.TestCheckResourceAttr(ProviderShortName + "_action."+pre+"_ls_action", "params", "[\"dir\"]"),
					resource.TestCheckResourceAttr(ProviderShortName+"_action."+pre+"_ls_action", "start_title_template", "my_action started"),
					resource.TestCheckResourceAttr(ProviderShortName+"_action."+pre+"_ls_action", "complete_title_template", "my_action completed"),
					resource.TestCheckResourceAttr(ProviderShortName+"_action."+pre+"_ls_action", "error_title_template", "my_action failed"),
					// resource.TestCheckResourceAttr(ProviderShortName + "_action."+pre+"_ls_action", "file_deps", "[\""+pre+"_action_file\"]"),
				),
			},
			{
				Config: getProviderConfigString() + buildMockAccResourceAction(pre, true),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(ProviderShortName+"_action."+pre+"_ls_action", "name", pre+"_ls_action"),
					resource.TestCheckResourceAttr(ProviderShortName+"_action."+pre+"_ls_action", "command", "`ls ${dir}; export FOO='bar'`"),
					resource.TestCheckResourceAttr(ProviderShortName+"_action."+pre+"_ls_action", "enabled", "true"),
					//resource.TestCheckResourceAttr(ProviderShortName + "_action."+pre+"_ls_action", "params", "[\"dir\"]"),
					//resource.TestCheckResourceAttr(ProviderShortName + "_action."+pre+"_ls_action", "resource_tags_to_export", "[\"kubernetes.io/os\"]"),
					resource.TestCheckResourceAttr(ProviderShortName+"_action."+pre+"_ls_action", "start_title_template", "my_action started"),
					resource.TestCheckResourceAttr(ProviderShortName+"_action."+pre+"_ls_action", "complete_title_template", "my_action completed"),
					resource.TestCheckResourceAttr(ProviderShortName+"_action."+pre+"_ls_action", "error_title_template", "my_action failed"),
					resource.TestCheckResourceAttr(ProviderShortName+"_action."+pre+"_ls_action", "resource_query", "host"),
					resource.TestCheckResourceAttr(ProviderShortName+"_action."+pre+"_ls_action", "timeout", "20"),
					resource.TestCheckResourceAttr(ProviderShortName+"_action."+pre+"_ls_action", "shell", "/bin/bash"),
					resource.TestCheckResourceAttr(ProviderShortName+"_action."+pre+"_ls_action", "res_env_var", "FOO"),
					resource.TestCheckResourceAttr(ProviderShortName+"_action."+pre+"_ls_action", "start_short_template", "started"),
					resource.TestCheckResourceAttr(ProviderShortName+"_action."+pre+"_ls_action", "start_long_template", "started..."),
					resource.TestCheckResourceAttr(ProviderShortName+"_action."+pre+"_ls_action", "complete_short_template", "completed"),
					resource.TestCheckResourceAttr(ProviderShortName+"_action."+pre+"_ls_action", "complete_long_template", "completed..."),
					resource.TestCheckResourceAttr(ProviderShortName+"_action."+pre+"_ls_action", "error_short_template", "failed"),
					resource.TestCheckResourceAttr(ProviderShortName+"_action."+pre+"_ls_action", "error_long_template", "failed..."),
					resource.TestCheckResourceAttr(ProviderShortName+"_action."+pre+"_ls_action", "allowed_entities.#", "2"),
					resource.TestCheckResourceAttr(ProviderShortName+"_action."+pre+"_ls_action", "allowed_entities.0", "user1"),
					resource.TestCheckResourceAttr(ProviderShortName+"_action."+pre+"_ls_action", "allowed_entities.1", "user2"),
					// resource.TestCheckResourceAttr(ProviderShortName + "_action."+pre+"_ls_action", "file_deps", "[\""+pre+"_action_file\"]"),
				),
			},
			{
				// Test Importer..
				ResourceName:      ProviderShortName + "_action." + pre + "_ls_action",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})

	//resource.UnitTest(t, resource.TestCase{
	//	PreCheck:          func() { testAccPreCheck(t) },
	//	ProviderFactories: providerFactories,
	//})
}

func buildMockAccResourceAction(prefix string, full bool) string {
	extra := `
			resource_query = "host"
			timeout = 20
			shell = "/bin/bash"
			res_env_var = "FOO"
			resource_tags_to_export = ["kubernetes.io/os"]
			start_short_template    = "started"
			start_long_template    = "started..."
			complete_short_template = "completed"
			complete_long_template = "completed..."
			error_short_template    = "failed"
			error_long_template    = "failed..."
			allowed_entities = ["user1", "user2"]
`
	depFile := `
		resource "` + ProviderShortName + `_file" "` + prefix + `_action_file" {
			name = "` + prefix + `_action_file"
			input_file = "${path.module}/../data/opcp_example.sh"
			destination_path = "/tmp/opcp_action.sh"
			resource_query = "host"
			description = "op_copy action script."
			enabled = false
		}
`
	if !full {
		extra = ""
	}
	return depFile + `
		resource "` + ProviderShortName + `_action" "` + prefix + `_ls_action" {
			name = "` + prefix + `_ls_action"
			command = "` + "`ls $${dir}; export FOO='bar'`" + `"
			description = "List some files"
			params = ["dir"]
			enabled = true
			start_title_template    = "my_action started"
			complete_title_template = "my_action completed"
			error_title_template    = "my_action failed"
			file_deps = [` + ProviderShortName + `_file.` + prefix + `_action_file.name]
			` + extra + `
		}
`
}

////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////
// Alarm

func TestAccResourceAlarm(t *testing.T) {
	pre := RandomAlphaPrefix(5)

	resource.UnitTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: getProviderConfigString() + buildMockAccResourceAlarm(pre, false),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(ProviderShortName+"_alarm."+pre+"_cpu_alarm", "name", pre+"_cpu_alarm"),
					resource.TestCheckResourceAttr(ProviderShortName+"_alarm."+pre+"_cpu_alarm", "fire_query", "( cpu_usage > 0 | sum ( 5 ) ) >= 2"),
					resource.TestCheckResourceAttr(ProviderShortName+"_alarm."+pre+"_cpu_alarm", "clear_query", "( cpu_usage < 0 | sum ( 5 ) ) >= 2"),
					resource.TestCheckResourceAttr(ProviderShortName+"_alarm."+pre+"_cpu_alarm", "description", "Watch CPU usage."),
					resource.TestCheckResourceAttr(ProviderShortName+"_alarm."+pre+"_cpu_alarm", "resource_query", "host"),
					resource.TestCheckResourceAttr(ProviderShortName+"_alarm."+pre+"_cpu_alarm", "enabled", "true"),
					resource.TestCheckResourceAttr(ProviderShortName+"_alarm."+pre+"_cpu_alarm", "fire_title_template", "alarm fired"),
					resource.TestCheckResourceAttr(ProviderShortName+"_alarm."+pre+"_cpu_alarm", "resolve_title_template", "alarm resolved"),
				),
			},
			{
				Config: getProviderConfigString() + buildMockAccResourceAlarm(pre, true),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(ProviderShortName+"_alarm."+pre+"_cpu_alarm", "name", pre+"_cpu_alarm"),
					resource.TestCheckResourceAttr(ProviderShortName+"_alarm."+pre+"_cpu_alarm", "fire_query", "( cpu_usage > 0 | sum ( 5 ) ) >= 2"),
					resource.TestCheckResourceAttr(ProviderShortName+"_alarm."+pre+"_cpu_alarm", "clear_query", "( cpu_usage < 0 | sum ( 5 ) ) >= 2"),
					resource.TestCheckResourceAttr(ProviderShortName+"_alarm."+pre+"_cpu_alarm", "description", "Watch CPU usage."),
					resource.TestCheckResourceAttr(ProviderShortName+"_alarm."+pre+"_cpu_alarm", "resource_query", "host"),
					resource.TestCheckResourceAttr(ProviderShortName+"_alarm."+pre+"_cpu_alarm", "enabled", "true"),
					resource.TestCheckResourceAttr(ProviderShortName+"_alarm."+pre+"_cpu_alarm", "fire_short_template", "fired"),
					resource.TestCheckResourceAttr(ProviderShortName+"_alarm."+pre+"_cpu_alarm", "fire_long_template", "fired..."),
					resource.TestCheckResourceAttr(ProviderShortName+"_alarm."+pre+"_cpu_alarm", "resolve_short_template", "resolved"),
					resource.TestCheckResourceAttr(ProviderShortName+"_alarm."+pre+"_cpu_alarm", "resolve_long_template", "resolved..."),
					resource.TestCheckResourceAttr(ProviderShortName+"_alarm."+pre+"_cpu_alarm", "raise_for", "local"),
					resource.TestCheckResourceAttr(ProviderShortName+"_alarm."+pre+"_cpu_alarm", "check_interval_sec", "50"),
					resource.TestCheckResourceAttr(ProviderShortName+"_alarm."+pre+"_cpu_alarm", "resource_type", "HOST"),
					resource.TestCheckResourceAttr(ProviderShortName+"_alarm."+pre+"_cpu_alarm", "family", "custom"),
					resource.TestCheckResourceAttr(ProviderShortName+"_alarm."+pre+"_cpu_alarm", "metric_name", "cpu_usage"),
					resource.TestCheckResourceAttr(ProviderShortName+"_alarm."+pre+"_cpu_alarm", "condition_type", "above"),
					resource.TestCheckResourceAttr(ProviderShortName+"_alarm."+pre+"_cpu_alarm", "condition_value", "1"),
				),
			},
			{
				// Test Importer..
				ResourceName:      ProviderShortName + "_alarm." + pre + "_cpu_alarm",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func buildMockAccResourceAlarm(prefix string, full bool) string {
	extra := `
			fire_short_template     = "fired"
			fire_long_template      = "fired..."
			resolve_short_template  = "resolved"
			resolve_long_template   = "resolved..."
			raise_for               = "local"
			check_interval_sec      = 50
			resource_type           = "HOST"
			family                  = "custom"
			metric_name             = "cpu_usage"
			condition_type          = "above"
			condition_value         = "1"
`
	if !full {
		extra = ""
	}
	return `
		resource "` + ProviderShortName + `_alarm" "` + prefix + `_cpu_alarm" {
			name = "` + prefix + `_cpu_alarm"
			fire_query = "( cpu_usage > 0 | sum ( 5 ) ) >= 2"
			clear_query = "( cpu_usage < 0 | sum ( 5 ) ) >= 2"
			description = "Watch CPU usage."
			resource_query = "host"
			enabled = true
			fire_title_template     = "alarm fired"
			resolve_title_template  = "alarm resolved"
			` + extra + `
		}
`
}

////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////
// Time Trigger

func TestAccResourceTimeTrigger(t *testing.T) {
	pre := RandomAlphaPrefix(5)

	resource.UnitTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: getProviderConfigString() + buildMockAccResourceTimeTrigger(pre, false),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(ProviderShortName+"_time_trigger."+pre+"_time_trigger", "name", pre+"_time_trigger"),
					resource.TestCheckResourceAttr(ProviderShortName+"_time_trigger."+pre+"_time_trigger", "fire_query", "every 5m"),
				),
			},
			{
				Config: getProviderConfigString() + buildMockAccResourceTimeTrigger(pre, true),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(ProviderShortName+"_time_trigger."+pre+"_time_trigger", "name", pre+"_time_trigger"),
					resource.TestCheckResourceAttr(ProviderShortName+"_time_trigger."+pre+"_time_trigger", "fire_query", "every 5m"),
					resource.TestCheckResourceAttr(ProviderShortName+"_time_trigger."+pre+"_time_trigger", "start_date", "2024-02-29T08:00:00"),
					resource.TestCheckResourceAttr(ProviderShortName+"_time_trigger."+pre+"_time_trigger", "end_date", "2100-02-28T08:00:00"),
					resource.TestCheckResourceAttr(ProviderShortName+"_time_trigger."+pre+"_time_trigger", "enabled", "true"),
				),
			},
			{
				// Test Importer..
				ResourceName:      ProviderShortName + "_time_trigger." + pre + "_time_trigger",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func buildMockAccResourceTimeTrigger(prefix string, full bool) string {
	extra := `
			start_date = "2024-02-29T08:00:00"
			end_date   = "2100-02-28T08:00:00"
			enabled    = true
`
	if !full {
		extra = ""
	}
	return `
		resource "` + ProviderShortName + `_time_trigger" "` + prefix + `_time_trigger" {
			name = "` + prefix + `_time_trigger"
			fire_query = "every 5m"
			` + extra + `
		}
`
}

////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////
// Bot

func TestAccResourceBot(t *testing.T) {
	pre := RandomAlphaPrefix(5)

	resource.UnitTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: getProviderConfigString() + buildMockAccResourceAction(pre, false) + buildMockAccResourceAlarm(pre, false) + buildMockAccResourceBot(pre),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(ProviderShortName+"_bot."+pre+"_cpu_bot", "name", pre+"_cpu_bot"),
					resource.TestCheckResourceAttr(ProviderShortName+"_bot."+pre+"_cpu_bot", "command", "if "+pre+"_cpu_alarm then "+pre+"_ls_action(dir=\"/tmp\") fi"),
					resource.TestCheckResourceAttr(ProviderShortName+"_bot."+pre+"_cpu_bot", "description", "Act on \"CPU\" usage."),
					resource.TestCheckResourceAttr(ProviderShortName+"_bot."+pre+"_cpu_bot", "enabled", "true"),
					resource.TestCheckResourceAttr(ProviderShortName+"_bot."+pre+"_cpu_bot", "family", "custom"),
				),
			},
			{
				// Test Importer..
				ResourceName:      ProviderShortName + "_bot." + pre + "_cpu_bot",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func buildMockAccResourceBot(prefix string) string {
	return `
		resource "` + ProviderShortName + `_bot" "` + prefix + `_cpu_bot" {
			name        = "` + prefix + `_cpu_bot"
			command     = "if ${` + ProviderShortName + `_alarm.` + prefix + `_cpu_alarm.name} then ${` + ProviderShortName + `_action.` + prefix + `_ls_action.name}(dir=\"/tmp\")fi "
			description = "Act on \"CPU\" usage."
			enabled     = true
			family      = "custom"
		}
`
}

////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////
// Metric

func TestAccResourceMetric(t *testing.T) {
	pre := RandomAlphaPrefix(5)

	resource.UnitTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: getProviderConfigString() + buildMockAccResourceMetric(pre),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(ProviderShortName+"_metric."+pre+"_cpu_plus_one", "name", pre+"_cpu_plus_one"),
					resource.TestCheckResourceAttr(ProviderShortName+"_metric."+pre+"_cpu_plus_one", "value", "cpu_usage + 2"),
					resource.TestCheckResourceAttr(ProviderShortName+"_metric."+pre+"_cpu_plus_one", "description", "Erroneous CPU usage."),
					resource.TestCheckResourceAttr(ProviderShortName+"_metric."+pre+"_cpu_plus_one", "resource_type", "POD"),
					resource.TestCheckResourceAttr(ProviderShortName+"_metric."+pre+"_cpu_plus_one", "units", "cores"),
				),
			},
			{
				// Test Importer..
				ResourceName:      ProviderShortName + "_metric." + pre + "_cpu_plus_one",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func buildMockAccResourceMetric(prefix string) string {
	return `
		resource "` + ProviderShortName + `_metric" "` + prefix + `_cpu_plus_one" {
			name = "` + prefix + `_cpu_plus_one"
			value = "cpu_usage + 2"
			description = "Erroneous CPU usage."
			resource_type = "POD"
			units = "cores"
		}
`
}

////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////
// Resource

func TestAccResourceResource(t *testing.T) {
	pre := RandomAlphaPrefix(5)

	resource.UnitTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: getProviderConfigString() + buildMockAccResourceResource(pre),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(ProviderShortName+"_resource."+pre+"_books", "name", pre+"_books"),
					resource.TestCheckResourceAttr(ProviderShortName+"_resource."+pre+"_books", "description", "Pods with books app."),
					resource.TestCheckResourceAttr(ProviderShortName+"_resource."+pre+"_books", "value", "host | pod | app = 'bookstore'"),
				),
			},
			{
				// Test Importer..
				ResourceName:      ProviderShortName + "_resource." + pre + "_books",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func buildMockAccResourceResource(prefix string) string {
	return `
		resource "` + ProviderShortName + `_resource" "` + prefix + `_books" {
			name = "` + prefix + `_books"
			description = "Pods with books app."
			value = "host | pod | app = 'bookstore'"
		}
`
}

////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////
// Circuit Breaker

func TestAccResourceCircuitBreaker(t *testing.T) {
	pre := RandomAlphaPrefix(5)
	name := pre + "_circuit_breaker"
	fullName := ProviderShortName + "_circuit_breaker." + name

	resource.UnitTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: getProviderConfigString() + buildMockAccResourceAction(pre, false) + buildMockAccResourceCircuitBreaker(pre),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fullName, "name", name),
					resource.TestCheckResourceAttr(fullName, "command", "hosts | id=[1,2] | "+pre+"_ls_action"),
					resource.TestCheckResourceAttr(fullName, "breaker_type", "hard"),
					resource.TestCheckResourceAttr(fullName, "hard_limit", "5"),
					resource.TestCheckResourceAttr(fullName, "duration", "10s"),
					resource.TestCheckResourceAttr(fullName, "fail_over", "safe"),
				),
			},
			{
				// Test Importer..
				ResourceName:      fullName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func buildMockAccResourceCircuitBreaker(prefix string) string {
	name := prefix + "_circuit_breaker"
	return `
		resource "` + ProviderShortName + `_circuit_breaker" "` + name + `" {
			name = "` + name + `"
			command = "hosts | id=[1,2] | ${` + ProviderShortName + `_action.` + prefix + `_ls_action.name} "
			breaker_type = "hard"
			hard_limit = 5
			duration = "10s"
			fail_over = "safe"
			enabled = true
		}
`
}

////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////
// File

func TestAccResourceFile(t *testing.T) {
	pre := RandomAlphaPrefix(5)

	resource.UnitTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: getProviderConfigString() + buildMockAccResourceFile(pre),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(ProviderShortName+"_file."+pre+"_ex_file", "name", pre+"_ex_file"),
					resource.TestCheckResourceAttr(ProviderShortName+"_file."+pre+"_ex_file", "destination_path", "/tmp/opcp_example.sh"),
					resource.TestCheckResourceAttr(ProviderShortName+"_file."+pre+"_ex_file", "description", "op_copy example script."),
					resource.TestCheckResourceAttr(ProviderShortName+"_file."+pre+"_ex_file", "resource_query", "host"),
					resource.TestCheckResourceAttr(ProviderShortName+"_file."+pre+"_ex_file", "enabled", "false"),
					// computed values...
					resource.TestCheckResourceAttr(ProviderShortName+"_file."+pre+"_ex_file", "file_length", "58"),
					resource.TestCheckResourceAttr(ProviderShortName+"_file."+pre+"_ex_file", "checksum", "dbfb2a7d8176bd6e3dde256824421de3"),
					// just check that it's set
					resource.TestCheckResourceAttrSet(ProviderShortName+"_file."+pre+"_ex_file", "file_length"),
				),
			},
			{
				// Test Importer..
				ResourceName:      ProviderShortName + "_file." + pre + "_ex_file",
				ImportState:       true,
				ImportStateVerify: true,
				//// The filename (input_file) is not stored in the Op DB, and so can't be recreated for "import".
				ImportStateVerifyIgnore: []string{"input_file", "inline_data"},
				//ExpectError: regexp.MustCompile("input_file"), // Despite tickets to the contrary, this doesn't seem to work with ImportStateVerify
			},
		},
	})
}

func buildMockAccResourceFile(prefix string) string {
	return `
		resource "` + ProviderShortName + `_file" "` + prefix + `_ex_file" {
			name = "` + prefix + `_ex_file"
			input_file = "${path.module}/../data/opcp_example.sh"
			destination_path = "/tmp/opcp_example.sh"
			resource_query = "host"
			description = "op_copy example script."
			enabled = false
		}
`
}

////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////
// inline File

func TestAccResourceFileContent(t *testing.T) {
	pre := RandomAlphaPrefix(5)

	resource.UnitTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: getProviderConfigString() + buildMockAccResourceFileContent(pre),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(ProviderShortName+"_file."+pre+"_ex_file_inline", "name", pre+"_ex_file_inline"),
					resource.TestCheckResourceAttr(ProviderShortName+"_file."+pre+"_ex_file_inline", "destination_path", "/tmp/opcp_example_inline.sh"),
					resource.TestCheckResourceAttr(ProviderShortName+"_file."+pre+"_ex_file_inline", "description", "op_copy example script."),
					resource.TestCheckResourceAttr(ProviderShortName+"_file."+pre+"_ex_file_inline", "resource_query", "host"),
					resource.TestCheckResourceAttr(ProviderShortName+"_file."+pre+"_ex_file_inline", "enabled", "false"),
					// computed values...
					resource.TestCheckResourceAttr(ProviderShortName+"_file."+pre+"_ex_file_inline", "file_length", "58"),
					resource.TestCheckResourceAttr(ProviderShortName+"_file."+pre+"_ex_file_inline", "checksum", "dbfb2a7d8176bd6e3dde256824421de3"),
					// just check that it's set
					resource.TestCheckResourceAttrSet(ProviderShortName+"_file."+pre+"_ex_file_inline", "file_length"),
				),
			},
			{
				// Test Importer..
				ResourceName:      ProviderShortName + "_file." + pre + "_ex_file_inline",
				ImportState:       true,
				ImportStateVerify: true,
				//// The filename (input_file) is not stored in the Op DB, and so can't be recreated for "import".
				ImportStateVerifyIgnore: []string{"input_file", "inline_data"},
				//ExpectError: regexp.MustCompile("input_file"), // Despite tickets to the contrary, this doesn't seem to work with ImportStateVerify
			},
		},
	})
}

func buildMockAccResourceFileContent(prefix string) string {
	return `
		resource "` + ProviderShortName + `_file" "` + prefix + `_ex_file_inline" {
			name = "` + prefix + `_ex_file_inline"
			#inline_data = "file(${path.module}/../data/opcp_example.sh)"
			inline_data = "#!/bin/bash\n\necho \"sample text 1\" > /tmp/sample_text.txt\n\n"
			destination_path = "/tmp/opcp_example_inline.sh"
			resource_query = "host"
			description = "op_copy example script."
			enabled = false
		}
`
}

////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////
// Principal

func TestAccResourcePrincipal(t *testing.T) {
	pre := RandomAlphaPrefix(5)

	resource.UnitTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: getProviderConfigString() + buildMockAccResourcePrincipal(pre, false),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(ProviderShortName+"_principal."+pre+"_principal", "name", pre+"_principal"),
					resource.TestCheckResourceAttr(ProviderShortName+"_principal."+pre+"_principal", "identity", "group_identity"),
				),
			},
			{
				Config: getProviderConfigString() + buildMockAccResourcePrincipal(pre, true),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(ProviderShortName+"_principal."+pre+"_principal", "name", pre+"_principal"),
					resource.TestCheckResourceAttr(ProviderShortName+"_principal."+pre+"_principal", "identity", "group_identity"),
					// TODO: add a get_principal_class function in platform backend
					// and return the appropriate idp_name using the idp_id from db
					// otherwise it cannot be returned from symbol table manager
					// resource.TestCheckResourceAttr(ProviderShortName + "_principal."+pre+"_principal", "idp_name", "azure"),
					resource.TestCheckResourceAttr(ProviderShortName+"_principal."+pre+"_principal", "action_limit", "100"),
					resource.TestCheckResourceAttr(ProviderShortName+"_principal."+pre+"_principal", "execute_limit", "50"),
					resource.TestCheckResourceAttr(ProviderShortName+"_principal."+pre+"_principal", "administer_permission", "false"),
					resource.TestCheckResourceAttr(ProviderShortName+"_principal."+pre+"_principal", "configure_permission", "false"),
				),
			},
			{
				// Test Importer..
				ResourceName:      ProviderShortName + "_principal." + pre + "_principal",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func buildMockAccResourcePrincipal(prefix string, full bool) string {
	extra := `
			idp_name              = "azure"
			action_limit          = 100
			execute_limit         = 50
			administer_permission = false
			configure_permission  = false
`
	if !full {
		extra = ""
	}
	return `
		resource "` + ProviderShortName + `_principal" "` + prefix + `_principal" {
			name = "` + prefix + `_principal"
			identity = "group_identity"
			` + extra + `
		}
`
}

////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////
// System Settings

func TestAccResourceSystemSettings(t *testing.T) {
	pre := RandomAlphaPrefix(5)

	resource.UnitTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: getProviderConfigString() + buildMockAccResourceSystemSettings(pre),
				Check: resource.ComposeTestCheckFunc(
					// resource.TestCheckResourceAttr(ProviderShortName + "_system_settings.system_settings", "name", "system_settings"),
					// Access Control
					resource.TestCheckResourceAttr(ProviderShortName+"_system_settings.system_settings", "administrator_grants_create_user_token", "true"),
					resource.TestCheckResourceAttr(ProviderShortName+"_system_settings.system_settings", "administrator_grants_read_user_token", "true"),
					resource.TestCheckResourceAttr(ProviderShortName+"_system_settings.system_settings", "administrator_grants_regenerate_user_token", "false"),
					resource.TestCheckResourceAttr(ProviderShortName+"_system_settings.system_settings", "administrator_grants_create_user", "true"),
					// Runbooks
					resource.TestCheckResourceAttr(ProviderShortName+"_system_settings.system_settings", "approval_feature_enabled", "true"),
					resource.TestCheckResourceAttr(ProviderShortName+"_system_settings.system_settings", "runbook_ad_hoc_approval_request_enabled", "true"),
					resource.TestCheckResourceAttr(ProviderShortName+"_system_settings.system_settings", "runbook_approval_request_expiry_time", "6"),
					resource.TestCheckResourceAttr(ProviderShortName+"_system_settings.system_settings", "run_approval_expiry_time", "5"),
					resource.TestCheckResourceAttr(ProviderShortName+"_system_settings.system_settings", "approval_editable_allowed_resource_query_enabled", "true"),
					resource.TestCheckResourceAttr(ProviderShortName+"_system_settings.system_settings", "approval_allow_individual_notification", "true"),
					resource.TestCheckResourceAttr(ProviderShortName+"_system_settings.system_settings", "approval_optional_request_ticket_url", "false"),
					resource.TestCheckResourceAttr(ProviderShortName+"_system_settings.system_settings", "time_trigger_permissions_user", DefaultUserName),
					resource.TestCheckResourceAttr(ProviderShortName+"_system_settings.system_settings", "parallel_runs_fired_by_time_triggers", "5"),
					// Audit
					resource.TestCheckResourceAttr(ProviderShortName+"_system_settings.system_settings", "external_audit_storage_enabled", "false"),
					resource.TestCheckResourceAttr(ProviderShortName+"_system_settings.system_settings", "external_audit_storage_type", "ELASTIC"),
					resource.TestCheckResourceAttr(ProviderShortName+"_system_settings.system_settings", "external_audit_storage_batch_period_sec", "10"),
					// General
					resource.TestCheckResourceAttr(ProviderShortName+"_system_settings.system_settings", "environment_name", "Env_Name via TF"),
					resource.TestCheckResourceAttr(ProviderShortName+"_system_settings.system_settings", "environment_name_background", "#673ab7"),
					resource.TestCheckResourceAttr(ProviderShortName+"_system_settings.system_settings", "param_value_max_length", "10000"),
					resource.TestCheckResourceAttr(ProviderShortName+"_system_settings.system_settings", "maintenance_mode_enabled", "false"),
					resource.TestCheckResourceAttr(ProviderShortName+"_system_settings.system_settings", "allowed_tags.#", "1"),
					resource.TestCheckResourceAttr(ProviderShortName+"_system_settings.system_settings", "allowed_tags.0", ".*"),
					resource.TestCheckResourceAttr(ProviderShortName+"_system_settings.system_settings", "skipped_tags.#", "1"),
					resource.TestCheckResourceAttr(ProviderShortName+"_system_settings.system_settings", "skipped_tags.0", "skipped"),
				),
			},
			{
				// Test Importer..
				ResourceName:            ProviderShortName + "_system_settings.system_settings",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"name"},
			},
		},
	})
}

func buildMockAccResourceSystemSettings(prefix string) string {
	return `
		resource "` + ProviderShortName + `_system_settings" "system_settings" {
			name = "system_settings"
			# Access Control
			administrator_grants_create_user_token     = true
			administrator_grants_read_user_token       = true
			administrator_grants_regenerate_user_token = false
			administrator_grants_create_user           = true
			# Runbooks
			approval_feature_enabled                         = true
			runbook_ad_hoc_approval_request_enabled          = true
			runbook_approval_request_expiry_time             = 6
			run_approval_expiry_time                         = 5
			approval_editable_allowed_resource_query_enabled = true
			approval_allow_individual_notification           = true
			approval_optional_request_ticket_url             = false
			time_trigger_permissions_user                    = "` + DefaultUserName + `"
			parallel_runs_fired_by_time_triggers             = 5
			# Audit
			external_audit_storage_enabled          = false
			external_audit_storage_type             = "ELASTIC"
			external_audit_storage_batch_period_sec = 10
			# General
			environment_name            = "Env_Name via TF"
			environment_name_background = "#673ab7"
			param_value_max_length      = 10000
			maintenance_mode_enabled    = false
			allowed_tags                = [".*"]
			skipped_tags                = ["skipped"]
		}
`
}

////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////
// Report Template

func TestAccResourceReportTemplate(t *testing.T) {
	pre := RandomAlphaPrefix(5)

	resource.UnitTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: getProviderConfigString() + buildMockAccResourceReportTemplate(pre, false),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(ProviderShortName+"_report_template."+pre+"_report_template", "name", pre+"_report_template"),
					resource.TestCheckResourceAttr(ProviderShortName+"_report_template."+pre+"_report_template", "blocks", buildMockReportTemplateBlocks()),
				),
			},
			{
				Config: getProviderConfigString() + buildMockAccResourceReportTemplate(pre, true),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(ProviderShortName+"_report_template."+pre+"_report_template", "name", pre+"_report_template"),
					resource.TestCheckResourceAttr(ProviderShortName+"_report_template."+pre+"_report_template", "blocks", buildMockReportTemplateBlocks()),
					resource.TestCheckResourceAttr(ProviderShortName+"_report_template."+pre+"_report_template", "links", "\"[{\\\"label\\\":\\\"linked_report_template\\\",\\\"report_template_name\\\":\\\"linked_report_template\\\"}]\""),
				),
			},
			{
				// Test Importer..
				ResourceName:      ProviderShortName + "_report_template." + pre + "_report_template",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func wrapJsonEncode(data string) string {
	return "jsonencode(" + data + ")"
}

func buildMockReportTemplateBlocks() string {
	return `[{"breakdown_by_tag":"tag_1","breakdown_tags_values":[{"color":"#AAAAAA","label":"label_0","values":["passed","skipped"]}],"group_by_tag":"tag_0","group_by_tag_order":{"type":"DEFAULT","values":[]},"include_other_breakdown_tag_values":true,"include_resources_without_group_tag":false,"other_tags_to_export":["other_tag_1","other_tag_2"],"resource_query":"host","resources_breakdown":[{"breakdown_values":[{"count":1,"value":"value"}],"group_by_value":"tag_0"}],"title":"Block Name","view_mode":"PERCENTAGE"}]`
}

func buildMockReportTemplateLinks() string {
	return `[{"label":"linked_report_template","report_template_name":"linked_report_template"}]`
}

func buildMockAccResourceReportTemplate(prefix string, full bool) string {

	report_name := prefix + "_report_template"

	extra := `
			links = ` + wrapJsonEncode(buildMockReportTemplateLinks()) + `
			  depends_on = [
					` + ProviderShortName + `_report_template.` + report_name + `
				]
			}	
			
			resource "` + ProviderShortName + `_report_template" "linked_report_template" {
				name = "linked_report_template"
				blocks = ` + wrapJsonEncode("[]") + `
			}
			`

	if !full {
		extra = `
			}
		`
	}
	return `
		resource "` + ProviderShortName + `_report_template" "` + prefix + `_report_template" {
			name = "` + report_name + `"
			blocks = ` + wrapJsonEncode(buildMockReportTemplateBlocks()) + extra
}

////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////
// Dashboard

func TestAccResourceDashboard(t *testing.T) {
	pre := RandomAlphaPrefix(5)

	resource.UnitTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: getProviderConfigString() + buildMockAccResourceDashboard(pre, false),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(ProviderShortName+"_dashboard."+pre+"_dashboard", "name", pre+"_dashboard"),
					resource.TestCheckResourceAttr(ProviderShortName+"_dashboard."+pre+"_dashboard", "dashboard_type", "TAGS_SEQUENCE"),
				),
			},
			{
				Config: getProviderConfigString() + buildMockAccResourceDashboard(pre, true),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(ProviderShortName+"_dashboard."+pre+"_dashboard", "name", pre+"_dashboard"),
					resource.TestCheckResourceAttr(ProviderShortName+"_dashboard."+pre+"_dashboard", "dashboard_type", "TAGS_SEQUENCE"),
					resource.TestCheckResourceAttr(ProviderShortName+"_dashboard."+pre+"_dashboard", "resource_query", "host"),
					resource.TestCheckResourceAttr(ProviderShortName+"_dashboard."+pre+"_dashboard", "groups", buildMockDashboardGroups()),
					resource.TestCheckResourceAttr(ProviderShortName+"_dashboard."+pre+"_dashboard", "values", buildMockDashboardValues()),
					resource.TestCheckResourceAttr(ProviderShortName+"_dashboard."+pre+"_dashboard", "other_tags.#", "2"),
					resource.TestCheckResourceAttr(ProviderShortName+"_dashboard."+pre+"_dashboard", "other_tags.0", "other_tag1"),
					resource.TestCheckResourceAttr(ProviderShortName+"_dashboard."+pre+"_dashboard", "other_tags.1", "other_tag2"),
					resource.TestCheckResourceAttr(ProviderShortName+"_dashboard."+pre+"_dashboard", "identifiers.#", "2"),
					resource.TestCheckResourceAttr(ProviderShortName+"_dashboard."+pre+"_dashboard", "identifiers.0", "identifier1"),
					resource.TestCheckResourceAttr(ProviderShortName+"_dashboard."+pre+"_dashboard", "identifiers.1", "identifier2"),
				),
			},
			{
				// Test Importer..
				ResourceName:      ProviderShortName + "_dashboard." + pre + "_dashboard",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func buildMockDashboardGroups() string {
	return `[{"name":"g1","tags":["cloud_provider","release_tag"]}]`
}

func buildMockDashboardValues() string {
	return `[{"color":"#78909c","values":["aws"]},{"color":"#ffa726","values":["release-X"]}]`
}

func buildMockAccResourceDashboard(prefix string, full bool) string {
	extra := `
  			resource_query = "host"
			groups = ` + wrapJsonEncode(buildMockDashboardGroups()) + `
			values = ` + wrapJsonEncode(buildMockDashboardValues()) + `
			other_tags  = ["other_tag1", "other_tag2"]
			identifiers = ["identifier1", "identifier2"]
`
	if !full {
		extra = ""
	}
	return `
		resource "` + ProviderShortName + `_dashboard" "` + prefix + `_dashboard" {
			name = "` + prefix + `_dashboard"
			dashboard_type = "TAGS_SEQUENCE"` + extra + `
		}
`
}

// //////////////////////////////////////////////////////////////////////////////
// //////////////////////////////////////////////////////////////////////////////
// Runbook
func TestAccResourceRunbook(t *testing.T) {
	pre := RandomAlphaPrefix(5)

	resource.UnitTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: getProviderConfigString() + buildMockAccResourceRunbook(pre, "data"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(ProviderShortName+"_runbook."+pre+"_runbook", "name", pre+"_runbook"),
					resource.TestCheckResourceAttr(ProviderShortName+"_runbook."+pre+"_runbook", "data", buildExpectedRunbookData()),
				),
			},
			{
				Config: getProviderConfigString() + buildMockAccResourceRunbook(pre, "minimal"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(ProviderShortName+"_runbook."+pre+"_runbook", "name", pre+"_runbook"),
					resource.TestCheckResourceAttr(ProviderShortName+"_runbook."+pre+"_runbook", "cells", "[]"),
				),
			},
			{
				Config: getProviderConfigString() + buildMockAccResourceRunbook(pre, "full"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(ProviderShortName+"_runbook."+pre+"_runbook", "name", pre+"_runbook"),
					resource.TestCheckResourceAttr(ProviderShortName+"_runbook."+pre+"_runbook", "cells", buildExpectedRunbookCells()),
					resource.TestCheckResourceAttr(ProviderShortName+"_runbook."+pre+"_runbook", "params", buildExpectedRunbookParams()),
					// TODO: until we fix the external_params diff on second update (because there is no external alarm linked) we can't enable this
					// resource.TestCheckResourceAttr(ProviderShortName + "_runbook."+pre+"_runbook", "external_params", expectedRunbookExternalParams()),
					resource.TestCheckResourceAttr(ProviderShortName+"_runbook."+pre+"_runbook", "description", "A sample runbook."),
					resource.TestCheckResourceAttr(ProviderShortName+"_runbook."+pre+"_runbook", "timeout_ms", "5000"),
					resource.TestCheckResourceAttr(ProviderShortName+"_runbook."+pre+"_runbook", "allowed_entities.#", "2"),
					resource.TestCheckResourceAttr(ProviderShortName+"_runbook."+pre+"_runbook", "allowed_entities.0", "user_1"),
					resource.TestCheckResourceAttr(ProviderShortName+"_runbook."+pre+"_runbook", "allowed_entities.1", "user_2"),
					resource.TestCheckResourceAttr(ProviderShortName+"_runbook."+pre+"_runbook", "approvers.#", "2"),
					resource.TestCheckResourceAttr(ProviderShortName+"_runbook."+pre+"_runbook", "approvers.0", "user_2"),
					resource.TestCheckResourceAttr(ProviderShortName+"_runbook."+pre+"_runbook", "approvers.1", "user_3"),
					resource.TestCheckResourceAttr(ProviderShortName+"_runbook."+pre+"_runbook", "editors.#", "2"),
					resource.TestCheckResourceAttr(ProviderShortName+"_runbook."+pre+"_runbook", "editors.0", "user_2"),
					resource.TestCheckResourceAttr(ProviderShortName+"_runbook."+pre+"_runbook", "editors.1", "user_4"),
					resource.TestCheckResourceAttr(ProviderShortName+"_runbook."+pre+"_runbook", "is_run_output_persisted", "true"),
					resource.TestCheckResourceAttr(ProviderShortName+"_runbook."+pre+"_runbook", "allowed_resources_query", "host"),
					resource.TestCheckResourceAttr(ProviderShortName+"_runbook."+pre+"_runbook", "communication_workspace", "workspace_name"),
					resource.TestCheckResourceAttr(ProviderShortName+"_runbook."+pre+"_runbook", "communication_channel", "channel_name"),
					resource.TestCheckResourceAttr(ProviderShortName+"_runbook."+pre+"_runbook", "labels.#", "2"),
					resource.TestCheckResourceAttr(ProviderShortName+"_runbook."+pre+"_runbook", "labels.0", "label1"),
					resource.TestCheckResourceAttr(ProviderShortName+"_runbook."+pre+"_runbook", "labels.1", "label2"),
					resource.TestCheckResourceAttr(ProviderShortName+"_runbook."+pre+"_runbook", "communication_cud_notifications", "true"),
					resource.TestCheckResourceAttr(ProviderShortName+"_runbook."+pre+"_runbook", "communication_approval_notifications", "false"),
					resource.TestCheckResourceAttr(ProviderShortName+"_runbook."+pre+"_runbook", "communication_execution_notifications", "true"),
					resource.TestCheckResourceAttr(ProviderShortName+"_runbook."+pre+"_runbook", "filter_resource_to_action", "true"),
					resource.TestCheckResourceAttr(ProviderShortName+"_runbook."+pre+"_runbook", "enabled", "true"),
				),
			},
			{
				// Test Importer..
				ResourceName:            ProviderShortName + "_runbook." + pre + "_runbook",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"data", "cells", "enabled", "external_params", "params"},
			},
		},
	})
}

func buildMockRunbookCells() string {
	return "[{\"md\":\"CREATE\"},{\"op\":\"action success = `echo SUCCESS`\"},{\"op\":\"enable success\"},{\"op\":\"success\",\"enabled\":false},{\"md\":\"CLEANUP\"},{\"op\":\"delete success\"}]"
}

func buildExpectedRunbookCells() string {
	return "[\n  {\n    \"enabled\": true,\n    \"md\": \"CREATE\",\n    \"name\": \"unnamed\"\n  },\n  {\n    \"enabled\": true,\n    \"name\": \"unnamed\",\n    \"op\": \"action success = `echo SUCCESS`\"\n  },\n  {\n    \"enabled\": true,\n    \"name\": \"unnamed\",\n    \"op\": \"enable success\"\n  },\n  {\n    \"enabled\": false,\n    \"name\": \"unnamed\",\n    \"op\": \"success\"\n  },\n  {\n    \"enabled\": true,\n    \"md\": \"CLEANUP\",\n    \"name\": \"unnamed\"\n  },\n  {\n    \"enabled\": true,\n    \"name\": \"unnamed\",\n    \"op\": \"delete success\"\n  }\n]"
}

func buildMockRunbookParams() string {
	return `[{"name":"param_1","value":"default_value"},{"name":"param_2","value":"default_value","required":false,"export":true},{"name":"param_3","value":"default_value","export":true},{"name":"param_4","required":false}]`
}

func buildExpectedRunbookParams() string {
	return "[{\"value\":\"default_value\",\"required\":true,\"name\":\"param_1\",\"export\":false},{\"value\":\"default_value\",\"required\":false,\"name\":\"param_2\",\"export\":true},{\"value\":\"default_value\",\"required\":true,\"name\":\"param_3\",\"export\":true},{\"value\":\"\",\"required\":false,\"name\":\"param_4\",\"export\":false}]"
}

func buildMockRunbookExternalParams() string {
	return `[{"name":"external_param_1","source":"alertmanager","json_path":"$.<path>","export":true,"value":"default_value"},{"name":"external_param_2","source":"alertmanager","json_path":"$.<path>","value":"default_value"},{"name":"external_param_3","source":"alertmanager","json_path":"$.<path>","export":true},{"name":"external_param_4","source":"alertmanager","json_path":"$.<path>"}]`
}

func buildExpectedRunbookExternalParams() string {
	return "[{\"value\":\"default_value\",\"source\":\"alertmanager\",\"name\":\"external_param_1\",\"json_path\":\"$.<path>\"},{\"value\":\"default_value\",\"source\":\"alertmanager\",\"name\":\"external_param_2\",\"json_path\":\"$.<path>\"},{\"value\":\"\",\"source\":\"alertmanager\",\"name\":\"external_param_3\",\"json_path\":\"$.<path>\"},{\"value\":\"\",\"source\":\"alertmanager\",\"name\":\"external_param_4\",\"json_path\":\"$.<path>\"}]"
}

func buildMockRunbookData() string {
	return `
		{
			"cells": [
				{
				"content": "CREATE",
				"enabled": true,
				"name": "unnamed",
				"type": "MARKDOWN"
				},
				{
				"content": "action success = ` + "`echo SUCCESS`" + `",
				"enabled": true,
				"name": "unnamed",
				"type": "OP_LANG"
				},
				{
				"content": "enable success",
				"enabled": true,
				"name": "unnamed",
				"type": "OP_LANG"
				},
				{
				"content": "success",
				"enabled": false,
				"name": "unnamed",
				"type": "OP_LANG"
				},
				{
				"content": "CLEANUP",
				"enabled": true,
				"name": "unnamed",
				"type": "MARKDOWN"
				},
				{
				"content": "delete success",
				"enabled": true,
				"name": "unnamed",
				"type": "OP_LANG"
				}
			],
			"params": [
				{
				"value": "default_value",
				"required": true,
				"name": "param_1",
				"export": false
				},
				{
				"value": "default_value",
				"required": false,
				"name": "param_2",
				"export": true
				},
				{
				"value": "default_value",
				"required": true,
				"name": "param_3",
				"export": true
				},
				{
				"value": "",
				"required": false,
				"name": "param_4",
				"export": false
				}
			]
		}
	`
}

func buildExpectedRunbookData() string {
	return "{\"cells\":[{\"content\":\"CREATE\",\"enabled\":true,\"name\":\"unnamed\",\"type\":\"MARKDOWN\"},{\"content\":\"action success = `echo SUCCESS`\",\"enabled\":true,\"name\":\"unnamed\",\"type\":\"OP_LANG\"},{\"content\":\"enable success\",\"enabled\":true,\"name\":\"unnamed\",\"type\":\"OP_LANG\"},{\"content\":\"success\",\"enabled\":false,\"name\":\"unnamed\",\"type\":\"OP_LANG\"},{\"content\":\"CLEANUP\",\"enabled\":true,\"name\":\"unnamed\",\"type\":\"MARKDOWN\"},{\"content\":\"delete success\",\"enabled\":true,\"name\":\"unnamed\",\"type\":\"OP_LANG\"}],\"external_params\":[],\"interactive_state\":{\"param_1\":\"default_value\",\"param_2\":\"default_value\",\"param_3\":\"default_value\",\"param_4\":\"\"},\"params\":[{\"export\":false,\"name\":\"param_1\",\"required\":true,\"value\":\"default_value\"},{\"export\":true,\"name\":\"param_2\",\"required\":false,\"value\":\"default_value\"},{\"export\":true,\"name\":\"param_3\",\"required\":true,\"value\":\"default_value\"},{\"export\":false,\"name\":\"param_4\",\"required\":false,\"value\":\"\"}],\"secret_aliases\":[]}"
}

func buildMockAccResourceRunbook(prefix string, extraType string) string {

	extra := ""
	switch extraType {
	case "minimal":
		extra = "cells = jsonencode([])"
	case "full":
		extra = `
			cells = ` + wrapJsonEncode(buildMockRunbookCells()) + `
			params = ` + wrapJsonEncode(buildMockRunbookParams()) + `
			description                           = "A sample runbook."
			timeout_ms                            = 5000
			allowed_entities                      = ["user_1", "user_2"]
			approvers                             = ["user_2", "user_3"]
			editors                               = ["user_2", "user_4"]
			is_run_output_persisted               = true
			allowed_resources_query               = "host"
			communication_workspace               = "workspace_name"
			communication_channel                 = "channel_name"
			labels                                = ["label1", "label2"]
			communication_cud_notifications       = true
			communication_approval_notifications  = false
			communication_execution_notifications = true
			filter_resource_to_action             = true
			enabled                               = true
		`
	case "data":
		extra = "data = " + wrapJsonEncode(buildMockRunbookData())
	}

	return `
		resource "` + ProviderShortName + `_runbook" "` + prefix + `_runbook" {
			name = "` + prefix + `_runbook"
			` + extra + `
		}
`
}

// func testAccCompareNotebookCells(resourceName string, expected string) resource.TestCheckFunc {
// 	return func(s *terraform.State) error {
// 		// retrieve the resource by name from state
// 		rs, ok := s.RootModule().Resources[resourceName]
// 		if !ok {
// 			return fmt.Errorf("Not found: %s", resourceName)
// 		}
// 		appendActionLog(fmt.Sprintf("rs resource is: %+v, cells: %+v\n", rs, rs.Primary.Attributes["cells"]))
// 		//inJs, inErr := Base64ToJsonArray(rs.Primary.Attributes["cells"])
// 		//exJs, exErr := Base64ToJsonArray(expected)
// 		inJs, inErr := StringToJsonArray(rs.Primary.Attributes["cells"])
// 		exJs, exErr := StringToJsonArray(expected)
// 		if inErr != nil || exErr != nil {
// 			return fmt.Errorf("Notebook cells failed to decode/unmarshall: %s", resourceName)
// 		}
// 		if !reflect.DeepEqual(inJs, exJs) {
// 			return fmt.Errorf("Notebook cells differs from expected: %s", resourceName)
// 		}
// 		return nil
// 	}
// }

///// TODO XXX re-enable this when the CI backend is updated
// func TestAccResourceNotebook(t *testing.T) {
// 	pre := RandomAlphaPrefix(5)
//
// 	resource.UnitTest(t, resource.TestCase{
// 		PreCheck:          func() { testAccPreCheck(t) },
// 		ProviderFactories: providerFactories,
// 		Steps: []resource.TestStep{
// 			{
// 				Config: getProviderConfigString() + getAccResourceNotebook(pre),
// 				Check: resource.ComposeTestCheckFunc(
// 					resource.TestCheckResourceAttr(ProviderShortName + "_notebook."+pre+"_notebook", "name", pre+"_notebook"),
// 					resource.TestCheckResourceAttr(ProviderShortName + "_notebook."+pre+"_notebook", "description", "A sample notebook."),
// 				        resource.TestCheckResourceAttr(ProviderShortName + "_notebook."+pre+"_notebook", "enabled", "true"),
//                                      resource.TestCheckResourceAttr(ProviderShortName + "_notebook."+pre+"_ls_notebook", "allowed_entities.#", "2"),
//					resource.TestCheckResourceAttr(ProviderShortName + "_notebook."+pre+"_ls_notebook", "allowed_entities.0", "user1"),
//					resource.TestCheckResourceAttr(ProviderShortName + "_notebook."+pre+"_ls_notebook", "allowed_entities.1", "user2"),
// 					testAccCompareNotebookCells(ProviderShortName + "_notebook."+pre+"_notebook", getNotebookData()),
// 				),
// 			},
// 			{
// 				// Test Importer..
// 				ResourceName:      ProviderShortName + "_notebook." + pre + "_notebook",
// 				ImportState:       true,
// 				ImportStateVerify: true,
// 			},
// 		},
// 	})
// }

// func getNotebookData() string {
// 	return `[{"type":"MARKDOWN","name":"K","enabled":true,"content":"## This is a title"},{"type":"TEXT","name":"K2","enabled":false,"content":"Insert Text Here"},{"type":"MARKDOWN","name":"K2","enabled":true,"content":"Lorem ipsum in exemplum ad naseum."},{"type":"OP_LANG","name":"resource query","enabled":false,"content":"host"}]`
// }

// func getAccResourceNotebook(prefix string) string {
// 	return `
// 		resource "` + ProviderShortName + `_notebook" "` + prefix + `_notebook" {
// 			name = "` + prefix + `_notebook"
// 			description = "A sample notebook."
// 			cells = "` + strings.Replace(getNotebookData(), "\"", "\\\"", -1) + `"
// 			enabled = true
// 			allowed_entities = ["user1", "user2"]
// 		}
// `
// }
