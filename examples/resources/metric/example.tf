
resource "mcahr_metric" "full_metric" {
  name          = "full_metric"
  value         = "cpu_usage + 4"
  description   = "Erroneous CPU usage."
  units         = "<units>"
  resource_type = "HOST"
}


resource "mcahr_metric" "minimal_metric" {
  name  = "cpu_plus"
  value = "cpu_usage + 2"
}

