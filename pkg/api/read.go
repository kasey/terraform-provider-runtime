package api

import (
	"fmt"

	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/crossplane/provider-terraform-plugin/pkg/client"
	"github.com/crossplane/provider-terraform-plugin/pkg/registry"
	"github.com/hashicorp/terraform/providers"
	"github.com/pkg/errors"
)

var ErrNotFound = errors.New("Resource not found")

// Read returns an up-to-date version of the resource
// TODO: If `id` is unset for a new resource, how do we figure out
// what value needs to be used as the id?
func Read(p *client.Provider, r *registry.Registry, res resource.Managed) (resource.Managed, error) {
	gvk := res.GetObjectKind().GroupVersionKind()
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
	ctyEncoder, err := r.GetCtyEncoder(gvk)
	if err != nil {
		return nil, err
	}
	encoded, err := ctyEncoder(res, &s)
	if err != nil {
		return nil, err
	}
	req := providers.ReadResourceRequest{
		TypeName:   tfName,
		PriorState: encoded,
		Private:    nil,
	}
	resp := p.GRPCProvider.ReadResource(req)
	if resp.Diagnostics.HasErrors() {
		return res, resp.Diagnostics.NonFatalErr()
	}
	// should we persist resp.Private in a blob in the resource to use on the next call?
	// Risky since size is unbounded, but we might be matching core behavior more carefully
	ctyDecoder, err := r.GetCtyDecoder(gvk)
	if err != nil {
		return nil, err
	}
	if resp.NewState.IsNull() {
		return nil, ErrNotFound
	}
	return ctyDecoder(res, resp.NewState, &s)
}
