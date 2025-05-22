
resource "mcahr_system_settings" "system_settings" {
  name = "system_settings"
  # Access Control
  administrator_grants_create_user_token     = true
  administrator_grants_read_user_token       = true
  administrator_grants_regenerate_user_token = false
  administrator_grants_create_user           = true
  # Runbooks
  approval_feature_enabled                         = true
  runbook_ad_hoc_approval_request_enabled          = true
  runbook_approval_request_expiry_time             = 6
  run_approval_expiry_time                         = 5
  approval_editable_allowed_resource_query_enabled = true
  approval_allow_individual_notification           = true
  approval_optional_request_ticket_url             = false
  time_trigger_permissions_user                    = "NVIDIA"
  parallel_runs_fired_by_time_triggers             = 5
  # Audit
  external_audit_storage_enabled          = false
  external_audit_storage_type             = "ELASTIC"
  external_audit_storage_batch_period_sec = 10
  # General
  environment_name            = "Env_Name via TF"
  environment_name_background = "#673ab7"
  param_value_max_length      = 10000
  maintenance_mode_enabled    = false
  allowed_tags                = [".*"]
  skipped_tags                = []
}
