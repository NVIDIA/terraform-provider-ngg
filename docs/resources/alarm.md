---
page_title: "ngg_alarm Resource - terraform-provider-ngg"
subcategory: ""
description: |-
  Alarms are fully-customizable Metric or status checks that automatically trigger remediation Actions.
---

# ngg_alarm (Resource)

Alarms frequently check one or more Metric thresholds or custom Resource queries. The Alarm is raised based on custom thresholds or shell commands you define, which informs any connected Bot to trigger remedial Actions.

-> Nvidia GPU Guardian includes dozens of built-in Metrics on which to base your Alarms. You can also combine multiple Metrics into Metric Sets for monitoring many Metrics at once. You can even create your own Derived Metrics if none of the built-in options meet your needs.

## Required Properties

Each Alarm can define many properties to determine its behavior. The required properties when creating an Alarm are:

- name - The name of the Alarm.
- fire_query - The Op statement that triggers the Alarm.
- clear_query - The Op statement that clears the Alarm.
- resource_query - The Op query that selects which Resources the Alarm triggers from.

## Usage

The following example creates an Alarm named `my_cpu_alarm` that fires when at least 80% of a host Resource's CPU usage metric measurements are equal to or exceed `40%` over the previous minute.

```tf
resource "ngg_alarm" "cpu_alarm" {
  name = "my_cpu_alarm"
  fire_query = "(cpu_usage > 40 | sum(60)) >= 48.0"
  clear_query = "(cpu_usage < 40 | sum(60)) >= 48.0"
  resource_query = "hosts"
}
```

-> Metric data points are collected once per second for all Nvidia GPU Guardian Resources (i.e. hosts, pods, and containers). Thus, a Metric query of `(cpu_usage > 40 | sum(60)) >= 48.0` determines if at least 48 of the last 60 `cpu_usage` data points exceeded `40%`.  You can learn more from the Metrics documentation.

### Advanced Usage

You can also combine other Terraform resource blocks and variables to create complex Alarms.  In this example we're defining an Action called `jvm_trace_check_heap` that determines if JVM heap usage exceeds a variable-defined threshold:

```terraform
# Action to check the JVM heap usage on the selected resources and process.
resource "ngg_action" "jvm_trace_check_heap" {
  name        = "${var.namespace}_jvm_check_heap"
  description = "Check heap utilization by process regex."
  # Parameters passed in: the regular expression to select process name.
  params = ["JVM_PROCESS_REGEX"]
  # Extract the heap used for the matching process and return 1 if above threshold.
  command = "`hm=$(jstat -gc $(jps | grep \"$${JVM_PROCESS_REGEX}\" | awk '{print $1}') | tail -n 1 | awk '{split($0,a,\" \"); sum=a[3]+a[4]+a[6]+a[8]; print sum/1024}'); hm=$${hm%.*}; if [ $hm -gt ${var.mem_threshold} ]; then echo \"heap memory $hm MB > threshold ${var.mem_threshold} MB\"; exit 1; fi`"

  # UI / CLI annotation informational messages:
  start_short_template    = "Checking JVM heap usage."
  error_short_template    = "Error checking JVM heap usage."
  complete_short_template = "Finished checking JVM heap usage."
  start_long_template     = "Checking JVM process ${var.jvm_process_regex} heap usage."
  error_long_template     = "Error checking JVM process ${var.jvm_process_regex} heap usage."
  complete_long_template  = "Finished checking JVM process ${var.jvm_process_regex} heap usage."

  enabled = true
}
```

The `jvm_trace_heap_alarm` Alarm executes the `jvm_trace_check_heap` Action as part of its fire_query and clear_query, to determine when the Alarm is raised or resolved, respectively:

```terraform
# Alarm that triggers when the selected JVM heap usage exceeds the chosen size.
resource "ngg_alarm" "jvm_trace_heap_alarm" {
  name        = "${var.namespace}_jvm_heap_alarm"
  description = "Alarm on JVM heap usage growing larger than a threshold."
  # The query that triggers the alarm: is the JVM memory usage greater than a threshold.
  fire_query = "${ngg_action.jvm_trace_check_heap.name}('${var.jvm_process_regex}') == 1"
  # The query that ends the alarm: is the JVM memory usage lower than the threshold.
  clear_query = "${ngg_action.jvm_trace_check_heap.name}('${var.jvm_process_regex}') == 0"
  # How often is the alarm evaluated. This is a more slowly changing metric, so every 60 seconds is fine.
  check_interval_sec = var.check_interval
  # User-provided resource selection
  resource_query = var.resource_query

  # UI / CLI annotation informational messages:
  fire_short_template    = "JVM heap usage exceeded memory threshold."
  resolve_short_template = "JVM heap usage below memory threshold."
  # include relevant parameters, in case the user has multiple instances on different volumes/resources
  fire_long_template    = "JVM heap usage (process ${var.jvm_process_regex}) exceeded memory threshold ${var.mem_threshold} on ${var.resource_query}"
  resolve_long_template = "JVM heap usage (process ${var.jvm_process_regex}) below memory threshold ${var.mem_threshold} on ${var.resource_query}"

  # low-frequency, and a linux command, so compiling won't help
  compile_eligible = false

  # alarm is raised local to a resource (vs global)
  raise_for = "local"
  # raised on a linux command (not a standard metric)
  metric_name = "jvm_trace_check_heap"
  # threshold value
  condition_value = var.mem_threshold
  # fires when above the threshold
  condition_type = "above"
  # general type of alarm ("metric", "custom", or "system check")
  family = "custom"

  enabled = true
}
```

-> See the Nvidia GPU Guardian Alarms Documentation for more info.

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `fire_query` (String) The trigger condition for an Alarm (general expression) or the TimeTrigger (e.g. 'every 5m').
- `name` (String) The name/symbol for the object within Shoreline and the op language (must be unique, only alphanumeric/underscore).

### Optional

- `check_interval_sec` (String) Defaults to `1`.
- `clear_query` (String) The Alarm's resolution condition.
- `compile_eligible` (Boolean) If the Alarm can be effectively optimized. Defaults to `true`.
- `condition_type` (String) Kind of check in an Alarm (e.g. above or below) vs a threshold for a Metric.
- `condition_value` (String) Switching value (threshold) for a Metric in an Alarm.
- `description` (String) A user-friendly explanation of an object.
- `enabled` (Boolean) If the object is currently enabled or disabled. Defaults to `false`.
- `family` (String) General class for an Action or Bot (e.g., custom, standard, metric, or system check). Defaults to `custom`.
- `fire_long_template` (String) The long description of the Alarm's triggering condition.
- `fire_short_template` (String) The short description of the Alarm's triggering condition.
- `fire_title_template` (String) UI title of the Alarm's triggering condition.
- `metric_name` (String) The Alarm's triggering Metric.
- `mute_query` (String) The Alarm's mute condition.
- `raise_for` (String) Where an Alarm is raised (e.g., local to a resource, or global to the system). Defaults to `local`.
- `resolve_long_template` (String) The long description of the Alarm's resolution.
- `resolve_short_template` (String) The short description of the Alarm's resolution.
- `resolve_title_template` (String) UI title of the Alarm's' resolution.
- `resource_query` (String) A set of Resources (e.g. host, pod, container), optionally filtered on tags or dynamic conditions.
- `resource_type` (String)

### Read-Only

- `id` (String) The ID of this resource.
- `type` (String) The type of object (i.e., Alarm, Action, Bot, Metric, Resource, or File).