// Copyright 2025 NVIDIA Corporation
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"flag"
	"os"

	"terraform/terraform-provider/provider"

	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
	//"github.com/hashicorp/terraform-provider-scaffolding/internal/provider"
)

// Run "go generate" to format example terraform files and generate the docs for the registry

// If you do not have OpenTofu installed, you can remove the formatting command, but its suggested to
// ensure the documentation is formatted properly.
//go:generate tofu fmt -recursive ../target/examples/

// Run the docs generation tool, check its repository for more information on how it works and how docs
// can be customized.
//go:generate go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs --provider-name=terraform-provider-$PROVIDER_SHORT_NAME --rendered-provider-name=$RENDERED_PROVIDER_NAME --tf-version=1.5.7

var (
	// these will be set by the goreleaser configuration
	// to appropriate values for the compiled binary
	version string = "dev"

	// goreleaser can also pass the specific commit if you want
	// commit  string = ""

	ProviderPath string = os.Getenv("PROVIDER_PATH")
)

func main() {
	var debugMode bool

	flag.BoolVar(&debugMode, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	opts := &plugin.ServeOpts{
		Debug:        debugMode,
		ProviderFunc: provider.New(version),
		ProviderAddr: ProviderPath,
	}

	plugin.Serve(opts)
}
