resource "mcahr_integration" "nvault_integration" {
  name          = "nvault_integration"
  service_name  = "nvault"
  address       = "address"
  namespace     = "nvault_namespace"
  role_name     = "nvault_role"
  jwt_auth_path = "auth/jwt"
  serial_number = "001"
  enabled       = true
}
