
module "action" {
  source = "../action"
}

module "alarm" {
  source                    = "../alarm"
  cpu_threshold_action_name = module.action.cpu_threshold_action_name
}

module "time_trigger" {
  source = "../time_trigger"
}

module "bot" {
  source                    = "../bot"
  full_alarm_name           = module.alarm.full_alarm_name
  minimal_alarm_name        = module.alarm.minimal_alarm_name
  full_action_name          = module.action.full_action_name
  minimal_action_name       = module.action.minimal_action_name
  full_time_trigger_name    = module.time_trigger.full_time_trigger_name
  minimal_time_trigger_name = module.time_trigger.minimal_time_trigger_name
  full_runbook_name         = module.runbook.full_runbook_name
  minimal_runbook_name      = module.runbook.minimal_runbook_name
}

module "circuit_breaker" {
  source              = "../circuit_breaker"
  full_action_name    = module.action.full_action_name
  minimal_action_name = module.action.minimal_action_name
}

module "file" {
  source = "../file"
}

module "metric" {
  source = "../metric"
}

module "runbook" {
  source = "../runbook"
}

module "principal" {
  source = "../principal"
}


module "resource" {
  source = "../resource"
}

module "system_settings" {
  source = "../system_settings"
}


module "report_template" {
  source = "../report_template"
}

module "dashboard" {
  source = "../dashboard"
}

# This needs auth keys for each integration.
# module "integration" {
#   source    = "../integration"

