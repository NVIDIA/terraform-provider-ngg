resource "mcahr_dashboard" "full_dashboard" {
  name           = "full_dashboard"
  dashboard_type = "TAGS_SEQUENCE"
  resource_query = "host"
  groups = jsonencode([
    {
      "name" : "g1",
      "tags" : [
        "cloud_provider",
        "release_tag"
      ]
    }
  ])
  values = jsonencode([
    {
      "color" : "#78909c",
      "values" : [
        "aws"
      ]
    },
    {
      "color" : "#ffa726",
      "values" : [
        "release-X"
      ]
    }
  ])
  other_tags  = ["<other_tag>"]
  identifiers = ["<identifier>"]
}


resource "mcahr_dashboard" "minimal_dashboard" {
  name           = "minimal_dashboard"
  dashboard_type = "TAGS_SEQUENCE"
}
