
resource "ngg_integration" "datadog_integration" {
  name             = "datadog_integration"
  service_name     = "datadog"
  site_url         = "https://app.datadoghq.com"
  app_key          = "<app_key>"
  api_url          = "https://api.datadoghq.com"
  api_key          = "<api_key>"
  webhook_name     = "tf_webhook"
  serial_number    = "001"
  enabled          = true
  permissions_user = "<permissions_user>"
}