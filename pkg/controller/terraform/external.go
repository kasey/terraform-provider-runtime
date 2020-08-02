/*
Copyright 2020 The Crossplane Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package terraform

import (
	"context"
	"fmt"

	kubeclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/crossplane/terraform-provider-runtime/pkg/api"
	"github.com/crossplane/terraform-provider-runtime/pkg/client"
	"github.com/crossplane/terraform-provider-runtime/pkg/registry"
)

const (
	errNotMyType                  = "managed resource is not a MyType custom resource"
	errProviderNotRetrieved       = "provider could not be retrieved"
	errProviderSecretNil          = "cannot find Secret reference on Provider"
	errProviderSecretNotRetrieved = "secret referred in provider could not be retrieved"

	errNewClient = "cannot create new Service"
)

type External struct {
	KubeClient kubeclient.Client
	Registry   *registry.Registry
	Callbacks  managed.ExternalClientFns
	logger     logging.Logger
	provider   *client.Provider
}

func (c *External) Observe(ctx context.Context, kres resource.Managed) (managed.ExternalObservation, error) {
	gvk := kres.GetObjectKind().GroupVersionKind()
	c.logger.Debug(fmt.Sprintf("terraform.External.Observe: %s", gvk.String()))
	if c.Callbacks.ObserveFn != nil {
		return c.Callbacks.Observe(ctx, kres)
	}

	ares, err := api.Read(c.provider, c.Registry, kres)
	if err != nil {
		if err == api.ErrNotFound {
			return managed.ExternalObservation{}, nil
		}
		return managed.ExternalObservation{}, err
	}

	diffIniter, err := c.Registry.GetResourceDiffIniter(gvk)
	if err != nil {
		return managed.ExternalObservation{}, err
	}
	diff, err := diffIniter(kres, ares)
	if err != nil {
		return managed.ExternalObservation{}, err
	}
	/*
		if diff.DifferentAtProvider() {
			merged, err := diff.Merged()
			if err != nil {
				return managed.ExternalObservation{}, err
			}
			if err := c.KubeClient.Update(ctx, merged); err != nil {
				return managed.ExternalObservation{}, err
			}
		}
	*/

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: diff.DifferentForProvider(),
		// ConnectionDetails: getConnectionDetails(cr, instance),
	}, nil
}

func (c *External) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	gvk := mg.GetObjectKind().GroupVersionKind()
	c.logger.Debug(fmt.Sprintf("terraform.External.Create: %s", gvk.String()))
	if c.Callbacks.CreateFn != nil {
		return c.Callbacks.Create(ctx, mg)
	}

	created, err := api.Create(c.provider, c.Registry, mg)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	/*
		if err := c.KubeClient.Update(ctx, created); err != nil {
			return managed.ExternalCreation{}, err
		}
	*/

	diffIniter, err := c.Registry.GetResourceDiffIniter(gvk)
	if err != nil {
		return managed.ExternalCreation{}, err
	}
	diff, err := diffIniter(mg, created)
	if err != nil {
		return managed.ExternalCreation{}, err
	}
	if diff.DifferentAtProvider() {
		merged, err := diff.Merged()
		if err != nil {
			return managed.ExternalCreation{}, err
		}
		if err := c.KubeClient.Update(ctx, merged); err != nil {
			return managed.ExternalCreation{}, err
		}
	}

	return managed.ExternalCreation{}, nil
}

func (c *External) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	gvk := mg.GetObjectKind().GroupVersionKind()
	c.logger.Debug(fmt.Sprintf("terraform.External.Update: %s", gvk.String()))
	if c.Callbacks.UpdateFn != nil {
		return c.Callbacks.Update(ctx, mg)
	}

	return managed.ExternalUpdate{}, nil
}

func (c *External) Delete(ctx context.Context, mg resource.Managed) error {
	gvk := mg.GetObjectKind().GroupVersionKind()
	c.logger.Debug(fmt.Sprintf("terraform.External.Delete: %s", gvk.String()))
	if c.Callbacks.DeleteFn != nil {
		return c.Callbacks.Delete(ctx, mg)
	}

	return nil
}
