package registry

import (
	"fmt"

	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	xpresource "github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/crossplane/terraform-provider-runtime/pkg/client"
	"github.com/hashicorp/terraform/providers"
	"github.com/zclconf/go-cty/cty"
	k8schema "k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	kubeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

type ResourceUnmarshalFunc func([]byte) (xpresource.Managed, error)
type CtyEncodeFunc func(xpresource.Managed, *providers.Schema) (cty.Value, error)
type CtyDecodeFunc func(xpresource.Managed, cty.Value, *providers.Schema) (xpresource.Managed, error)
type YAMLEncodeFunc func(xpresource.Managed) ([]byte, error)
type ReconcilerConfigurerFunc func(ctrl.Manager, logging.Logger, *Registry, *client.ProviderPool) error
type ResourceDiffIniter func(kubeResource xpresource.Managed, providerResource xpresource.Managed) (ResourceDiff, error)
type PoolInitializer func(kube *kubeclient.Client)

type ResourceDiff struct {
	KubeResource            xpresource.Managed
	ProviderResource        xpresource.Managed
	ForProviderDiffCallback func(xpresource.Managed, xpresource.Managed) []string
	AtProviderDiffCallback  func(xpresource.Managed, xpresource.Managed) []string
	MergeFunc               func(xpresource.Managed, xpresource.Managed) (xpresource.Managed, error)
}

func (rd *ResourceDiff) DifferentAtProvider() bool {
	if len(rd.AtProviderDiffCallback(rd.KubeResource, rd.ProviderResource)) > 0 {
		return true
	}
	return false
}

func (rd *ResourceDiff) DifferentForProvider() bool {
	if len(rd.ForProviderDiffCallback(rd.KubeResource, rd.ProviderResource)) > 0 {
		return true
	}
	return false
}

func (rd *ResourceDiff) Merged() (xpresource.Managed, error) {
	return rd.MergeFunc(rd.KubeResource, rd.ProviderResource)
}

type ResourceDiffer interface {
	DifferentAtProvider() bool
	DifferentForProvider() bool
	Merged() (xpresource.Managed, error)
}

type Registry struct {
	resourceRepresenterMap     map[k8schema.GroupVersionKind]ResourceUnmarshalFunc
	ctyEncodeFuncMap           map[k8schema.GroupVersionKind]CtyEncodeFunc
	ctyDecodeFuncMap           map[k8schema.GroupVersionKind]CtyDecodeFunc
	terraformNameToGVK         map[string]k8schema.GroupVersionKind
	gvkToTerraformName         map[k8schema.GroupVersionKind]string
	yamlEncodeFuncMap          map[k8schema.GroupVersionKind]YAMLEncodeFunc
	reconcilerConfigurerMap    map[k8schema.GroupVersionKind]ReconcilerConfigurerFunc
	schemaBuilderMap           map[k8schema.GroupVersionKind]*scheme.Builder
	externalClientCallbacksMap map[k8schema.GroupVersionKind]*managed.ExternalClientFns
	resourceDiffIniters        map[k8schema.GroupVersionKind]ResourceDiffIniter
	provider                   *ProviderEntry
}

func NewRegistry() *Registry {
	return &Registry{
		resourceRepresenterMap:     make(map[k8schema.GroupVersionKind]ResourceUnmarshalFunc),
		ctyEncodeFuncMap:           make(map[k8schema.GroupVersionKind]CtyEncodeFunc),
		ctyDecodeFuncMap:           make(map[k8schema.GroupVersionKind]CtyDecodeFunc),
		terraformNameToGVK:         make(map[string]k8schema.GroupVersionKind),
		gvkToTerraformName:         make(map[k8schema.GroupVersionKind]string),
		yamlEncodeFuncMap:          make(map[k8schema.GroupVersionKind]YAMLEncodeFunc),
		reconcilerConfigurerMap:    make(map[k8schema.GroupVersionKind]ReconcilerConfigurerFunc),
		schemaBuilderMap:           make(map[k8schema.GroupVersionKind]*scheme.Builder),
		externalClientCallbacksMap: make(map[k8schema.GroupVersionKind]*managed.ExternalClientFns),
		resourceDiffIniters:        make(map[k8schema.GroupVersionKind]ResourceDiffIniter),
	}
}

func (r *Registry) RegisterProvider(entry *ProviderEntry) {
	r.provider = entry
}

func (r *Registry) RegisterResourceDiffIniter(gvk k8schema.GroupVersionKind, di ResourceDiffIniter) {
	if gvk.String() == "" {
		panic("RegisterResourceDiffer called with uninitialized GroupVersionKind")
	}
	if di == nil {
		panic(fmt.Sprintf("Cannot initialize, RegisterResourceDiffer called with nil value for gvk=%s", gvk.String()))
	}
	r.resourceDiffIniters[gvk] = di
}

func (r *Registry) RegisterExternalClientCallbacks(gvk k8schema.GroupVersionKind, cbfns *managed.ExternalClientFns) {
	if gvk.String() == "" {
		panic("RegisterExternalClientCallbacks called with uninitialized GroupVersionKind")
	}
	if cbfns == nil {
		panic(fmt.Sprintf("Cannot initialize RegisterExternalClientCallbacks called with nil value for gvk=%s", gvk.String()))
	}
	r.externalClientCallbacksMap[gvk] = cbfns
}

func (r *Registry) RegisterSchemeBuilder(gvk k8schema.GroupVersionKind, sb *scheme.Builder) {
	if gvk.String() == "" {
		panic("RegisterSchemeBuilder called with uninitialized GroupVersionKind")
	}
	if sb == nil {
		panic(fmt.Sprintf("Cannot initialize RegisterSchemeBuilder called with nil value for gvk=%s", gvk.String()))
	}
	r.schemaBuilderMap[gvk] = sb
}

func (r *Registry) RegisterReconcilerConfigureFunc(gvk k8schema.GroupVersionKind, f ReconcilerConfigurerFunc) {
	if gvk.String() == "" {
		panic("RegisterReconcilerConfigureFunc called with uninitialized GroupVersionKind")
	}
	if f == nil {
		panic(fmt.Sprintf("Cannot initialize RegisterReconcilerConfigureFunc called with nil value for gvk=%s", gvk.String()))
	}
	r.reconcilerConfigurerMap[gvk] = f
}

func (r *Registry) RegisterYAMLEncodeFunc(gvk k8schema.GroupVersionKind, f YAMLEncodeFunc) {
	if gvk.String() == "" {
		panic("RegisterYAMLEncodeFunc called with uninitialized GroupVersionKind")
	}
	if f == nil {
		panic(fmt.Sprintf("Cannot initialize RegisterYAMLEncodeFunc called with nil value for gvk=%s", gvk.String()))
	}
	r.yamlEncodeFuncMap[gvk] = f
}

func (r *Registry) RegisterResourceUnmarshalFunc(gvk k8schema.GroupVersionKind, f ResourceUnmarshalFunc) {
	if gvk.String() == "" {
		panic("RegisterResourceUnmarshalFunc called with uninitialized GroupVersionKind")
	}
	if f == nil {
		panic(fmt.Sprintf("Cannot initialize RegisterResourceUnmarshalFunc called with nil value for gvk=%s", gvk.String()))
	}
	r.resourceRepresenterMap[gvk] = f
}

func (r *Registry) RegisterCtyEncodeFunc(gvk k8schema.GroupVersionKind, f CtyEncodeFunc) {
	if gvk.String() == "" {
		panic("RegisterCtyEncodeFunc called with uninitialized GroupVersionKind")
	}
	if f == nil {
		panic(fmt.Sprintf("Cannot initialize: RegisterCtyEncodeFunc called with nil value for gvk=%s", gvk.String()))
	}
	r.ctyEncodeFuncMap[gvk] = f
}

func (r *Registry) RegisterCtyDecodeFunc(gvk k8schema.GroupVersionKind, f CtyDecodeFunc) {
	if gvk.String() == "" {
		panic("RegisterCtyDecodeFunc called with uninitialized GroupVersionKind")
	}
	if f == nil {
		panic(fmt.Sprintf("Cannot initialize: RegisterCtyDecodeFunc called with nil value for gvk=%s", gvk.String()))
	}
	r.ctyDecodeFuncMap[gvk] = f
}

func (r *Registry) RegisterTerraformNameMapping(tfname string, gvk k8schema.GroupVersionKind) {
	if gvk.String() == "" {
		panic("RegisterTerraformNameMapping called with uninitialized GroupVersionKind")
	}
	if tfname == "" {
		panic("RegisterTerraformNameMapping called with uninitialized tfname")
	}
	r.terraformNameToGVK[tfname] = gvk
	r.gvkToTerraformName[gvk] = tfname
}

func (r *Registry) GetYAMLEncodeFunc(gvk k8schema.GroupVersionKind) (YAMLEncodeFunc, error) {
	f, ok := r.yamlEncodeFuncMap[gvk]
	if !ok {
		return nil, fmt.Errorf("Could not find a yaml encoder function for GVK=%s", gvk.String())
	}
	return f, nil
}

func (r *Registry) GetCtyEncoder(gvk k8schema.GroupVersionKind) (CtyEncodeFunc, error) {
	f, ok := r.ctyEncodeFuncMap[gvk]
	if !ok {
		return nil, fmt.Errorf("Could not find a cty encoder function for GVK=%s", gvk.String())
	}
	return f, nil
}

func (r *Registry) GetCtyDecoder(gvk k8schema.GroupVersionKind) (CtyDecodeFunc, error) {
	f, ok := r.ctyDecodeFuncMap[gvk]
	if !ok {
		return nil, fmt.Errorf("Could not find a cty decoder function for GVK=%s", gvk.String())
	}
	return f, nil
}

func (r *Registry) GetResourceUnmarshalFunc(gvk k8schema.GroupVersionKind) (ResourceUnmarshalFunc, error) {
	rep, ok := r.resourceRepresenterMap[gvk]
	if !ok {
		return nil, fmt.Errorf("Could not find a resource representer for GVK=%s", gvk.String())
	}
	return rep, nil
}

func (r *Registry) GetGVKForTerraformName(name string) (k8schema.GroupVersionKind, error) {
	gvk, ok := r.terraformNameToGVK[name]
	if !ok {
		return gvk, fmt.Errorf("Could not find GVK for Terraform resource name=%s", name)
	}
	return gvk, nil
}

func (r *Registry) GetTerraformNameForGVK(gvk k8schema.GroupVersionKind) (string, error) {
	name, ok := r.gvkToTerraformName[gvk]
	if !ok {
		return "", fmt.Errorf("Could not find GVK for Terraform resource gvk=%s", name)
	}
	return name, nil
}

func (r *Registry) GetSchemeBuilderForGVK(gvk k8schema.GroupVersionKind) (*scheme.Builder, error) {
	sb, ok := r.schemaBuilderMap[gvk]
	if !ok {
		return nil, fmt.Errorf("Could not find scheme.Builder for gvk=%s", gvk.String())
	}
	return sb, nil
}

func (r *Registry) GetExternalClientCallbacksForGVK(gvk k8schema.GroupVersionKind) (*managed.ExternalClientFns, error) {
	cbfns, ok := r.externalClientCallbacksMap[gvk]
	if !ok {
		return nil, fmt.Errorf("Could not find managed.ExternalClientFns for gvk=%s", gvk.String())
	}
	return cbfns, nil
}

func (r *Registry) GetResourceDiffIniter(gvk k8schema.GroupVersionKind) (ResourceDiffIniter, error) {
	di, ok := r.resourceDiffIniters[gvk]
	if !ok {
		return nil, fmt.Errorf("Could not find ResourceDiffer for gvk=%s", gvk.String())
	}
	return di, nil
}

func (r *Registry) GetSchemeBuilders() []*scheme.Builder {
	builders := make([]*scheme.Builder, 0, len(r.schemaBuilderMap))
	for _, sb := range r.schemaBuilderMap {
		builders = append(builders, sb)
	}
	return builders
}

func (r *Registry) GetReconcilerConfigurers() []ReconcilerConfigurerFunc {
	funcs := make([]ReconcilerConfigurerFunc, 0, len(r.reconcilerConfigurerMap))
	for _, f := range r.reconcilerConfigurerMap {
		funcs = append(funcs, f)
	}
	return funcs
}

func (r *Registry) GetProviderEntry() (*ProviderEntry, error) {
	if r.provider == nil {
		return nil, fmt.Errorf("Could not find ProviderEntry, failed to bind with generated code")
	}
	return r.provider, nil
}

func (r *Registry) Register(entry *Entry) {
	r.RegisterCtyEncodeFunc(entry.GVK, entry.EncodeCtyCallback)
	r.RegisterCtyDecodeFunc(entry.GVK, entry.DecodeCtyCallback)
	r.RegisterResourceUnmarshalFunc(entry.GVK, entry.UnmarshalResourceCallback)
	r.RegisterTerraformNameMapping(entry.TerraformResourceName, entry.GVK)
	r.RegisterYAMLEncodeFunc(entry.GVK, entry.YamlEncodeCallback)
	r.RegisterReconcilerConfigureFunc(entry.GVK, entry.ReconcilerConfigurer)
	r.RegisterSchemeBuilder(entry.GVK, entry.SchemeBuilder)
	r.RegisterResourceDiffIniter(entry.GVK, entry.ResourceDiffIniter)
}
