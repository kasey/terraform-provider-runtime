package api

import (
	"fmt"

	"github.com/crossplane/provider-terraform-plugin/pkg/client"
	"github.com/hashicorp/terraform/providers"
	k8schema "k8s.io/apimachinery/pkg/runtime/schema"
)

func GetSchema(p *client.Provider) (map[string]providers.Schema, error) {
	resp := p.GRPCProvider.GetSchema()
	if resp.Diagnostics.HasErrors() {
		return nil, resp.Diagnostics.NonFatalErr()
	}

	return resp.ResourceTypes, nil
}

func SchemaForGVK(gvk k8schema.GroupVersionKind, p *client.Provider) (*providers.Schema, error) {
	schema, err := GetSchema(p)
	if err != nil {
		return nil, err
	}
	rs, ok := schema[gvk.Kind]
	if !ok {
		return nil, fmt.Errorf("Could not find schema for GVK=%s", gvk.String())
	}
	return &rs, nil
}
