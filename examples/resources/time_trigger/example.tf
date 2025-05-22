
resource "mcahr_time_trigger" "full_time_trigger" {
  name       = "full_time_trigger"
  fire_query = "every 5m"
  start_date = "2024-02-29T08:00:00"
  end_date   = "2100-02-28T08:00:00"
  enabled    = true
}


resource "mcahr_time_trigger" "minimal_time_trigger" {
  name       = "minimal_time_trigger"
  fire_query = "every 5m"
}
