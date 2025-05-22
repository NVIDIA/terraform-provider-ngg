
resource "ngg_integration" "elastic_integration" {
  name          = "elastic_integration"
  service_name  = "elastic"
  api_url       = "https://12c9b5cae1cd4c2cb25676017cc04ade.us-central1.gcp.cloud.es.io/elastic_index"
  api_key       = "d3dQeWJvb0JfSl9yTVZ5cTBUQzE6SjlfNUZBUUxSbXlab2pfYWlhQXpWQQ=="
  serial_number = "001"
  enabled       = true
}