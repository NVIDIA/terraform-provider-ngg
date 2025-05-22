
resource "mcahr_action" "cpu_threshold_action" {
  name = "cpu_threshold_action"
  # Evaluates current CPU usage and compares it to a parameter value named $cpu_threshold
  command = "`if [ $[100-$(vmstat 1 2|tail -1|awk '{print $15}')] -gt $cpu_threshold ]; then exit 1; fi`"

  description    = "Check CPU usage"
  enabled        = true
  params         = ["cpu_threshold"]
  resource_query = "hosts"
  timeout        = 5000

  start_title_template    = "CPU threshold action started"
  complete_title_template = "CPU threshold action completed"
  error_title_template    = "CPU threshold action failed"

  start_short_template    = "CPU threshold action short started"
  complete_short_template = "CPU threshold action short completed"
  error_short_template    = "CPU threshold action short failed"
}

