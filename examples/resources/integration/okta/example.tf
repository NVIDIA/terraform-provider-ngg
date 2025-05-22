
resource "ngg_integration" "okta_integration" {
  name             = "okta_integration"
  service_name     = "okta"
  serial_number    = "001"
  idp_name         = "okta"
  api_url          = "<url>"
  api_rate_limit   = 1000
  cache_ttl_ms     = 3000
  enabled          = true
  permissions_user = "<permissions_user>"
}
