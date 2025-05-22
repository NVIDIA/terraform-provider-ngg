################################################################################
# Module: jvm_stacktrace
# 
# Monitor JVM processes that match a reges, and if they exceed a memory limit,
# automatically collect a stack-trace from the selected process.
#
# Example usage:
#
#   module "jvm_stacktrace" {
#     # Location of the module:
#     source             = "./"
#   
#     # Namespace to allow multiple instances of the module, with different params:
#     namespace          = "jvm_trace"
#   
#     # Resource query to select the affected resources:
#     resource_query     = "jvm_pods"
#   
#     # Regular expresssion to select the monitored JVM processes:
#     pvc_regex          = "tomcat"
#   
#     # Maximum memory usage, in Mb, before the JVM process is traced:
#     disk_threshold     = 8
#   
#     # Destination of the memory-check, and trace scripts on the selected resources:
#     script_path = "/agent/scripts"
#   }

################################################################################


provider "mcahr" {
  # provider configuration here
  #url = "${var.mcahr_url}"
  retries = 2
  debug   = true
}


