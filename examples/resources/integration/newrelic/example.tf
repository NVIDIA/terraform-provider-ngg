
resource "ngg_integration" "newrelic_integration_eu" {
  name                       = "newrelic_integration_eu"
  service_name               = "newrelic"
  account_id                 = "<account_id>"
  api_url                    = "https://api.eu.newrelic.com/graphql"
  api_key                    = "<api_key>"
  insights_collector_url     = "<insights_collector_url>"
  insights_collector_api_key = "<insights_collector_api_key>"
  serial_number              = "001"
  enabled                    = true
  permissions_user           = "<permissions_user>"
}

resource "ngg_integration" "newrelic_integration_us" {
  name                       = "newrelic_integration_us"
  service_name               = "newrelic"
  account_id                 = "<account_id>"
  api_url                    = "https://api.newrelic.com/graphql"
  api_key                    = "<api_key>"
  insights_collector_url     = "<insights_collector_url>"
  insights_collector_api_key = "<insights_collector_api_key>"
  serial_number              = "001"
  enabled                    = true
  permissions_user           = "<permissions_user>"
}