
resource "mcahr_file" "full_path_file" {
  name             = "full_path_file"
  input_file       = "${path.module}/../../../data/opcp_example.sh"
  destination_path = "/tmp/opcp_example.sh"
  resource_query   = "host"
  description      = "op_copy example script."
  md5              = filemd5("${path.module}/../../../data/opcp_example.sh")
  mode             = "776"
  owner            = "owner"
  enabled          = true
}


resource "mcahr_file" "full_inline_file" {
  name             = "full_inline_file"
  inline_data      = "<file_content>"
  destination_path = "/tmp/full_inline_file.txt"
  resource_query   = "host"
  description      = "op_copy example script."
  enabled          = false
  md5              = md5("<file_content>")
  mode             = "776"
  owner            = "owner"
}


resource "mcahr_file" "minimal_path_file" {
  name             = "minimal_path_file"
  input_file       = "${path.module}/../../../data/opcp_example.sh"
  destination_path = "/tmp/minimal_path_file.txt"
  resource_query   = "host"
}


resource "mcahr_file" "minimal_inline_file" {
  name             = "minimal_inline_file"
  inline_data      = "<file_content>"
  destination_path = "/tmp/minimal_inline_file.txt"
  resource_query   = "host"
}
