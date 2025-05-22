---
page_title: 'Importing Existing Nvidia GPU Guardian Objects'
subcategory: ''
description: |-
---

# Importing Existing Nvidia GPU Guardian Objects

Import existing Nvidia GPU Guardian objects into your local Terraform state using the standard terraform import command.

## How to Import Existing Objects

1. Define a Terraform resource block in your local configuration with the appropriate Nvidia GPU Guardian `object_type` and desired `resource_name`.

    ```tf
    resource "Nvidia GPU Guardian_<object_type>" "<resource_name>" {
      # ...
    }
    ```

    For example, to import an existing Alarm named `heap_alarm`:

    ```tf
    resource "Nvidia GPU Guardian_alarm" "heap_alarm" {
      # ...
    }
    ```

    This resource block can be empty but it is required by Terraform to correctly map your local configuration to the remote resource.

2. Execute the `terraform import` command with the appropriate `resource_type`, `resource_name`, and `Nvidia GPU Guardian_object_name`.

    ```
    terraform import <resource_type>.<resource_name> <Nvidia GPU Guardian_object_name>
    ```

    In this case, we're importing a `Nvidia GPU Guardian_alarm` named `heap_alarm` in the local configuration.  The existing Nvidia GPU Guardian Alarm object is also named `heap_alarm`.

    ```
    $ terraform import Nvidia GPU Guardian_alarm.heap_alarm heap_alarm

    Nvidia GPU Guardian_alarm.heap_alarm: Importing from ID "heap_alarm"...
    Nvidia GPU Guardian_alarm.heap_alarm: Import prepared!
      Prepared Nvidia GPU Guardian_alarm for import
    Nvidia GPU Guardian_alarm.heap_alarm: Refreshing state... [id=heap_alarm]

    Import successful!

    The resources that were imported are shown above. These resources are now in
    your Terraform state and will henceforth be managed by Terraform.
    ```

    The `heap_alarm` Alarm is now mapped to the local `Nvidia GPU Guardian_alarm.heap_alarm` configuration block and you're free to adjust it as needed.

## Always Pre-define the Configuration

~> You _MUST_ define a Terraform resource configuration block for the imported resource, otherwise the import will fail with the following error:

```
Error: resource address "Nvidia GPU Guardian_alarm.missing_resource_name" does not exist in the configuration.

Before importing this resource, please create its configuration in the root module. For example:

resource "Nvidia GPU Guardian_alarm" "missing_resource_name" {
  # (resource arguments)
}
```

Once the resource is imported you can freely modify the configuration to match the remote resource or alter it as necessary.
