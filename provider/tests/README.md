# Provider Tests

This directory contains unit tests for the Terraform provider, organized by functionality.

## Test Organization

- **common.go**: Contains common utilities and global variables including resource type definitions
- **attribute_defaults_test.go**: Tests for the default value handling for different attribute types.
- **resource_schema_defaults_test.go**: Tests for schema integration with default values.

## Running Tests

To run all tests from the project root:

```bash
cd provider/tests
go test -v
```

To run a specific test file:

```bash
cd provider/tests
go test -v attribute_defaults_test.go
```

To run a specific test function:

```bash
cd provider/tests
go test -v -run TestAttributeDefaults
```

## Adding New Tests

When adding new tests:

1. Choose an existing file if testing related functionality, or create a new file with a descriptive name.
2. Follow the existing patterns for test organization.
3. Use table-driven tests where appropriate.
4. For resource testing that requires API access, use proper mocking.

## Adding New Resource Types

When adding new resource types to the provider, update the `SupportedResourceTypes` slice in `common.go`
