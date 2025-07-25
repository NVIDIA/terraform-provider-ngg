---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "mcahr_principal Resource - terraform-provider-mcahr"
subcategory: ""
description: |-
  NVIDIA Mission Control autonomous hardware recovery principal. An authorization group (e.g. Okta groups). Note: Admin privilege (in NVIDIA Mission Control autonomous hardware recovery) to create principal objects.
---

# mcahr_principal (Resource)

NVIDIA Mission Control autonomous hardware recovery principal. An authorization group (e.g. Okta groups). Note: Admin privilege (in NVIDIA Mission Control autonomous hardware recovery) to create principal objects.



<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `identity` (String) The email address or provider's (e.g. Okta) group-name for a permissions group.
- `name` (String) The name/symbol for the object within NVIDIA Mission Control autonomous hardware recovery and the op language (must be unique, only alphanumeric/underscore).

### Optional

- `action_limit` (Number) The number of simultaneous actions allowed for a permissions group. Defaults to `0`.
- `administer_permission` (Boolean) If a permissions group is allowed to perform "administer" actions. Defaults to `false`.
- `configure_permission` (Boolean) If a permissions group is allowed to perform "configure" actions. Defaults to `false`.
- `execute_limit` (Number) The number of simultaneous linux (shell) commands allowed for a permissions group. Defaults to `0`.
- `idp_name` (String) The Identity Provider's name. Defaults to ``.
- `view_limit` (Number, Deprecated) **Deprecated** Field 'view_limit' is obsolete. The number of simultaneous metrics allowed for a permissions group. Defaults to `0`.

### Read-Only

- `id` (String) The ID of this resource.
- `type` (String) The type of object (i.e., Alarm, Action, Bot, Resource, or File).
