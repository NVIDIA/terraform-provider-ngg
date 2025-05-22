
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

