package provider

import (
	"context"
	"slices"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// The framework validates every schema when Terraform asks for them, the way it would at the start
// of a plan: reserved attribute names, defaults on attributes that aren't computed, duplicate type
// names, and so on.
func TestSchemas(t *testing.T) {
	server, err := providerserver.NewProtocol6WithError(New("test")())()
	if err != nil {
		t.Fatal(err)
	}
	resp, err := server.GetProviderSchema(context.Background(), &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatal(err)
	}
	for _, diagnostic := range resp.Diagnostics {
		t.Errorf("%s: %s", diagnostic.Summary, diagnostic.Detail)
	}
	if got, want := len(resp.ResourceSchemas), len(resources()); got != want {
		t.Errorf("got %d resource schemas, want %d", got, want)
	}
	if got, want := len(resp.DataSourceSchemas), len(dataSources()); got != want {
		t.Errorf("got %d data source schemas, want %d", got, want)
	}
}

// Overrides are kept by hand, so this catches one that no longer matches what's generated.
func TestOverrides(t *testing.T) {
	for name, override := range resourceOverrides {
		api, ok := resourceAPIs[name]
		if !ok {
			t.Errorf("resourceOverrides has %s, which isn't generated any more: delete its override", name)
			continue
		}
		for _, attribute := range override.Updatable {
			if !slices.Contains(api.Replace, attribute) {
				t.Errorf("%s: Updatable lists %s, which doesn't force replacement any more: remove it", name, attribute)
			}
		}
	}
	for name := range dataSourceOverrides {
		if _, ok := dataSourceAPIs[name]; !ok {
			t.Errorf("dataSourceOverrides has %s, which isn't generated any more: delete its override", name)
		}
	}
}
