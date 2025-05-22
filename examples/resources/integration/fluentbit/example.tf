
resource "ngg_integration" "fluentbit_elastic_integration" {
  name          = "fluentbit_elastic_integration"
  service_name  = "fluentbit_elastic"
  api_url       = "http://0.0.0.0:9200"
  serial_number = "001"
  enabled       = true
}