package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"

	"github.com/gitmoap/terraform-provider-gitmoap/internal/provider"
)

// Set at build time with -ldflags "-X main.version=1.2.3".
var version = "dev"

func main() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "start the provider with support for debuggers like delve")
	flag.Parse()

	err := providerserver.Serve(context.Background(), provider.New(version), providerserver.ServeOpts{
		Address: "registry.terraform.io/gitmoap/gitmoap",
		Debug:   debug,
	})
	if err != nil {
		log.Fatal(err.Error())
	}
}
