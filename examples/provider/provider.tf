
terraform {
  required_providers {
    mcahr = {
      source  = "registry.opentofu.org/NVIDIA/mcahr"
      version = ">= 1.0.6"
    }
  }
}

provider "mcahr" {
  # provider configuration here
  #token = "xyz1.asdfj.asd3fas..."
  url     = "https://<url>"
  retries = 2
  debug   = true
}

