---
page_title: "mcahr_bot Resource - terraform-provider-mcahr"
subcategory: ""
description: |-
  Alarms use Bots to execute automated Actions.
---

# mcahr_bot (Resource)

A Bot connects a single Alarm to one or more Actions. When the Alarm is raised the Bot fires all associated and enabled Actions to close the auto-remediation loop.

## Required Properties

Each Bot has various configurable properties that determine its behavior. The minimal required properties to create a Bot are:

- name - The name of the Bot
- command - An `if-then-fi` statement containing the Alarm name and Action name associated with the Bot. Alternatively, the `command` property can be a custom Linux command.

## Usage

The following example creates a Bot named `cpu_bot` that executes the `restart_action` Action when the `high_cpu_alarm` Alarm's fire_query is true:

```tf
resource "mcahr_bot" "cpu_bot" {
  name = "cpu_bot"
  command = "if ${ mcahr_alarm.high_cpu_alarm.name} then ${ mcahr_action.restart_action.name} fi"
  description = "Restart on high CPU usage."
  enabled = true
}
```

The `command` property specifies the Alarm and Action that are connected by this Bot. It uses Terraform's built-in string interpolation to evaluate the name of both the Alarm and Action.

### Advanced Usage

Configuring a combination of an Alarm, Action, and Bot closes the fundamental auto-remediation loop provided by NVIDIA Mission Control autonomous hardware recovery.  Below we're using portions of NVIDIA Mission Control autonomous hardware recovery's JVM Op Pack to create a full incident automation loop when JVM memory usage gets too high.

First, the `jvm_trace_check_heap` Action determines if JVM heap usage exceeds a variable-defined threshold:

```terraform
# Action to check the JVM heap usage on the selected resources and process.
resource "mcahr_action" "jvm_trace_check_heap" {
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

The `jvm_trace_heap_alarm` Alarm executes the `jvm_trace_check_heap` Action as part of its fire_query and clear_query:

```terraform
# Alarm that triggers when the selected JVM heap usage exceeds the chosen size.
resource "mcahr_alarm" "jvm_trace_heap_alarm" {
  name        = "${var.namespace}_jvm_heap_alarm"
  description = "Alarm on JVM heap usage growing larger than a threshold."
  # The query that triggers the alarm: is the JVM memory usage greater than a threshold.
  fire_query = "${mcahr_action.jvm_trace_check_heap.name}('${var.jvm_process_regex}') == 1"
  # The query that ends the alarm: is the JVM memory usage lower than the threshold.
  clear_query = "${mcahr_action.jvm_trace_check_heap.name}('${var.jvm_process_regex}') == 0"
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

We define another Action called `jvm_trace_jvm_debug` that executes a bash script that dumps JVM debug data to AWS S3 before restarting the JVM:

```tf
# Action to dump the JVM stack-trace on the selected resources and process.
resource "mcahr_action" "jvm_trace_jvm_debug" {
  name = "${var.namespace}_jvm_dump_stack"
  description = "Dump JVM process (by regex) heap, thread and GC info to s3, then kill the pod."
  # Parameters passed in: the regular expression to select process name, and destination AWS S3 bucket.
  params = [ "JVM_PROCESS_REGEX" , "S3_BUCKET"]
  # Extract process info, and kill the pod.
  command = "`cd ${var.script_path} && chmod +x jvm_dumps.sh && ./jvm_dumps.sh $${JVM_PROCESS_REGEX} $${S3_BUCKET} >>/tmp/dumps.log`"
  # Select the shell to run 'command' with.
  #shell = "/bin/sh"

  # UI / CLI annotation informational messages:
  start_short_template    = "Dumping JVM info."
  error_short_template    = "Error dumping JVM info."
  complete_short_template = "Finished dumping JVM info."
  start_long_template     = "Dumping JVM process ${var.jvm_process_regex} info."
  error_long_template     = "Error dumping JVM process ${var.jvm_process_regex} info."
  complete_long_template  = "Finished dumping JVM process ${var.jvm_process_regex} info."

  enabled = true
}
```

Lastly, we connect the `jvm_trace_heap_alarm` Alarm and the `jvm_trace_check_heap` Action with the `jvm_trace_dump_bot` Bot:

```terraform
# Bot that fires the stack-dump action when the jvm heap exceeds the chosen memory threshold.
resource "mcahr_bot" "jvm_trace_dump_bot" {
  name        = "${var.namespace}_jvm_dump_bot"
  description = "Disk utilization handler bot"
  # If the JVM heap usage exceeds the threshold, dump the process stack, and push to AWS S3.
  # NOTE: Use a reference to the action and alarm, to ensure they are created and available before the bot.
  command = "if ${mcahr_alarm.jvm_trace_heap_alarm.name} then ${mcahr_action.jvm_trace_jvm_debug.name}(JVM_PROCESS_REGEX='${var.jvm_process_regex}', S3_BUCKET='${var.s3_bucket}') fi"

  # general type of bot this can be "standard" or "custom"
  family = "custom"

  enabled = true
}
```

Now, anytime JVM memory exceeds our defined threshold the JVM is automatically restarted and the debug data is exported for further analysis.

-> See the NVIDIA Mission Control autonomous hardware recovery Bots Documentation for more info.

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `command` (String) A specific action to run.
- `name` (String) The name/symbol for the object within NVIDIA Mission Control autonomous hardware recovery and the op language (must be unique, only alphanumeric/underscore).

### Optional

- `alarm_resource_query` (String) Defaults to ``.
- `communication_channel` (String) A string value denoting the slack channel where notifications related to the object should be sent to. Defaults to ``.
- `communication_workspace` (String) A string value denoting the slack workspace where notifications related to the object should be sent to. Defaults to ``.
- `description` (String) A user-friendly explanation of an object. Defaults to ``.
- `enabled` (Boolean) If the object is currently enabled or disabled. Defaults to `false`.
- `event_type` (String) Used to tag external triggers vs 'SYSTEM' alarms (default). Defaults to ``.
- `family` (String) General class for an Action or Bot (e.g., custom, standard, metric, or system check). Defaults to `custom`.
- `integration_name` (String) The name/symbol of a NVIDIA Mission Control autonomous hardware recovery integration involved in triggering the bot. Defaults to ``.
- `monitor_id` (String) For 'datadog' monitor triggered bots, the DD monitor identifier. Defaults to ``.

### Read-Only

- `id` (String) The ID of this resource.
- `type` (String) The type of object (i.e., Alarm, Action, Bot, Metric, Resource, or File).