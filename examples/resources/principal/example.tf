
resource "mcahr_principal" "full_principal" {
  name                  = "full_principal"
  identity              = "<full_identity_name>"
  idp_name              = "azure"
  action_limit          = 100
  execute_limit         = 50
  administer_permission = false
  configure_permission  = false
}


resource "mcahr_principal" "minimal_principal" {
  name     = "minimal_principal"
  identity = "<minimal_identity_name>"
}
