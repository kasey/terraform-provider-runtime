package api

import (
	"fmt"

	"github.com/crossplane/provider-terraform-plugin/pkg/client"
	"github.com/crossplane/provider-terraform-plugin/pkg/registry"
	"github.com/hashicorp/terraform/providers"
	"github.com/pkg/errors"
	k8schema "k8s.io/apimachinery/pkg/runtime/schema"
)

func GetSchema(p *client.Provider) (map[string]providers.Schema, error) {
	resp := p.GRPCProvider.GetSchema()
	if resp.Diagnostics.HasErrors() {
		return nil, resp.Diagnostics.NonFatalErr()
	}

	return resp.ResourceTypes, nil
}

/*
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
*/

func SchemaForGVK(gvk k8schema.GroupVersionKind, p *client.Provider, r *registry.Registry) (*providers.Schema, error) {
	schema, err := GetSchema(p)
	if err != nil {
		msg := "Failed to retrieve schema from provider in api.Read"
		return nil, errors.Wrap(err, msg)
	}
	tfName, err := r.GetTerraformNameForGVK(gvk)
	if err != nil {
		msg := fmt.Sprintf("Could not look up terraform resource name for gvk=%s", gvk.String())
		return nil, errors.Wrap(err, msg)
	}
	s, ok := schema[tfName]
	if !ok {
		return nil, fmt.Errorf("Could not look up schema using terraform resource name=%s (for gvk=%s", tfName, gvk.String())
	}

	return &s, nil
}
