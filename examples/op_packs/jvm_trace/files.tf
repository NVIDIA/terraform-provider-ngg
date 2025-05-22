
locals {
  jvm_dump_sh = "${path.module}/data/jvm_dumps.sh"
}

# Push the script that actually performs the JVM stack dump to the selected nodes.
resource "mcahr_file" "jvm_trace_dump_script" {
  name             = "${var.namespace}_dump_script"
  description      = "Script to dump JVM stack traces."
  input_file       = local.jvm_dump_sh                 # source file (relative to this module)
  md5              = filemd5(local.jvm_dump_sh)        # file hash for auto-update on TF apply
  destination_path = "${var.script_path}/jvm_dumps.sh" # where it is copied to on the selected resources
  resource_query   = var.resource_query                # which resources to copy to
  enabled          = true
}

