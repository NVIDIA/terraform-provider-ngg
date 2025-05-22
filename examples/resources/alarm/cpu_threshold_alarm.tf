
resource "mcahr_alarm" "cpu_threshold_alarm" {
  fire_query = "cpu_threshold_action(cpu_threshold=75) == 1"
  name       = "cpu_threshold_alarm"

  clear_query        = "cpu_threshold_action(cpu_threshold=75) == 0"
  description        = "High CPU usage alarm"
  enabled            = true
  resource_query     = "hosts"
  check_interval_sec = 10

  fire_short_template    = "High CPU Alarm fired"
  resolve_short_template = "High CPU Alarm resolved"
}

