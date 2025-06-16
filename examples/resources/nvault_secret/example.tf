resource "mcahr_nvault_secret" "remote_secret" {
  name              = "remote_secret"
  integration_name  = "existing_nvault_integration"
  vault_secret_key  = "nvault_secret_key"
  vault_secret_path = "nvault_secret_path"
}