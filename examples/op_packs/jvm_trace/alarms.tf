
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

