
resource "ngg_integration" "azure_active_directory_integration" {
  name             = "azure_active_directory_integration"
  service_name     = "azure_active_directory"
  serial_number    = "001"
  idp_name         = "azure"
  tenant_id        = "<tenant_id>"
  client_id        = "<client_id>"
  client_secret    = "<client_secret>"
  api_rate_limit   = 1000
  cache_ttl_ms     = 3000
  enabled          = true
  permissions_user = "<permissions_user>"
}
