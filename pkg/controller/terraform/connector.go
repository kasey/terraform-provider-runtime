package terraform

import (
	"context"
	"fmt"

	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/crossplane/provider-terraform-plugin/pkg/client"
	"github.com/crossplane/provider-terraform-plugin/pkg/registry"
	kubeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	errProviderPoolBorrowFailed        = "Failed to Borrow a provider from the ProviderPool"
	errFromContextDuringExternalClient = "Error from context while waiting for ExternalClient to complete operations"
)

// TODO: make New func and take Logger private (maybe?)
type Connector struct {
	KubeClient kubeclient.Client
	Registry   *registry.Registry
	Logger     logging.Logger
	Pool       *client.ProviderPool
}

func (c *Connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	gvk := mg.GetObjectKind().GroupVersionKind()
	fmt.Printf("Connect: %s", gvk.String())

	provider, err := c.Pool.Borrow(ctx, mg, c.KubeClient)
	if err != nil {
		return &External{}, err
	}

	// The context passed in from the Reconciler is marked Done at the end of the Reconcile loop.
	// We bank on that fact to schedule the provider lock for cleanup once its work is done.
	go func() {
		select {
		case <-ctx.Done():
			if err := ctx.Err(); err != nil {
				if err.Error() != "context canceled" {
					c.Logger.WithValues("err", err).Debug(errFromContextDuringExternalClient)
				}
			}
		}
		c.Pool.Return(provider)
	}()

	return &External{KubeClient: c.KubeClient, Registry: c.Registry, logger: c.Logger, provider: provider}, nil
}
