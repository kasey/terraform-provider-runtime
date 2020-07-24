package client

import (
	"context"

	"github.com/crossplane/crossplane-runtime/pkg/resource"
	kubeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type ProviderPool struct {
	size               int
	indexFifo          chan int
	providers          []*Provider
	reverseMap         map[*Provider]int
	initializeProvider Initializer
}

func (pp *ProviderPool) Borrow(ctx context.Context, res resource.Managed, kube kubeclient.Client) (*Provider, error) {
	index := <-pp.indexFifo
	if pp.providers[index] == nil {
		provider, err := pp.initializeProvider(ctx, res, kube)
		if err != nil {
			pp.indexFifo <- index
			return provider, err
		}
		pp.reverseMap[provider] = index
		pp.providers[index] = provider
	}
	return pp.providers[index], nil
}

func (pp *ProviderPool) Return(p *Provider) {
	index := pp.reverseMap[p]
	pp.indexFifo <- index
}

func NewProviderPool(initializer Initializer, size int) *ProviderPool {
	pool := &ProviderPool{
		size:               size,
		indexFifo:          make(chan int, size),
		providers:          make([]*Provider, size, size),
		reverseMap:         make(map[*Provider]int),
		initializeProvider: initializer,
	}
	for i := 0; i < size; i++ {
		pool.indexFifo <- i
	}

	return pool
}
