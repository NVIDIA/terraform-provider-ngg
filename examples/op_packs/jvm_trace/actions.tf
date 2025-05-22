
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
