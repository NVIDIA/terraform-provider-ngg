
resource "mcahr_circuit_breaker" "full_circuit_breaker" {
  name                    = "full_circuit_breaker_name"
  command                 = "host | ${var.full_action_name}"
  breaker_type            = "soft"
  hard_limit              = 7
  soft_limit              = 4
  duration                = "30s"
  fail_over               = "safe"
  enabled                 = true
  communication_workspace = "workspace_name"
  communication_channel   = "channel_name"
}


resource "mcahr_circuit_breaker" "minimal_circuit_breaker" {
  name       = "minimal_circuit_breaker"
  command    = "host | ${var.minimal_action_name}"
  hard_limit = 5
  duration   = "30s"
  enabled    = true
}