
resource "mcahr_alarm" "full_alarm" {
  name               = "full_alarm"
  fire_query         = "(cpu_usage > 1 | sum(5)) >= 2.75"
  mute_query         = "(cpu_usage < 0 | sum(5)) >= 2.75"
  clear_query        = "(cpu_usage < 0 | sum(5)) >= 2.75"
  description        = "Watch CPU usage."
  resource_query     = "host"
  check_interval_sec = 50
  condition_type     = "above"
  condition_value    = "10"
  metric_name        = "<metric_name>"
  raise_for          = "local"
  resource_type      = "HOST"

  fire_title_template    = "fired cpu_alarm title"
  resolve_title_template = "cleared cpu_alarm title"

  fire_short_template    = "fired cpu_alarm short"
  resolve_short_template = "cleared cpu_alarm short"

  fire_long_template    = "fired cpu_alarm long"
  resolve_long_template = "cleared cpu_alarm long"

  enabled = true
}


resource "mcahr_alarm" "minimal_alarm" {
  name       = "minimal_alarm"
  fire_query = "(cpu_usage > 1 | sum(5)) >= 2.75"
}

