package registry

import (
	"github.com/crossplane/terraform-provider-runtime/pkg/client"
	k8schema "k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

// Entry is a collection of callbacks and metadata needed to dynamically
// reconcile a resource through the terraform ExternalClient.
type Entry struct {
	// GVK is used to index other elements of the Entry by GVK
	GVK k8schema.GroupVersionKind
	// TerraformResourceName is needed to map the crossplane type
	// to the Terraform type name. This is needed to find the schema
	// for the type and to identify the type in API calls.
	TerraformResourceName string
	// EncodeCtyCallback produces the cty.Value (cty-encoded resource for
	// terraform) for a resource.Managed, given the corresponding schema
	// object. Note that we do not try to compile schemas in to the generated
	// code, they are always obtained from the terraform process itself.
	EncodeCtyCallback CtyEncodeFunc
	// DecodeCtyCallback is the complement to EncodeCtyCallback. In addition
	// to the schema and cty.Value, it also requires a resource.Managed, using
	// the deepcopied value from this resource as the base structure (and
	// providing values for .Spec fields and k8s metadata)
	DecodeCtyCallback CtyDecodeFunc
	// ResourceDiffIniter is a callback function that creates a ResourceDiff
	// for a pair of managed.Resource objects. This type uses generated
	// callbacks under the hood to:
	// - determine if the provider's State needs to be synced locally
	// - determine if the k8s resource Spec needs to be synced to the provider
	// - create a merged representation of the two (eg update local with remote)
	ResourceDiffIniter ResourceDiffIniter
	// SchemeBuilder is used to register the controller for this type with the
	// controller runtime. StartTerraformManager (in pkg/controller) iterates
	// through all the registration entries and performs the bindings.
	SchemeBuilder *scheme.Builder
	// ReconcilerConfigurer is the function responsible for calling
	// managed.NewReconciler to bind the reconciler to the managed resource
	// type. It is also invoked in StartTerraformManager.
	ReconcilerConfigurer ReconcilerConfigurerFunc
	// UnmarshalResourceCallback is only used for prototyping atm -- it's a
	// function that can parse the []byte representation of a managed resource
	// to a resource.Managed
	UnmarshalResourceCallback ResourceUnmarshalFunc
	// YamlEncodeCallback is the complement to UnmarshalResourceCallback, taking
	// a resource.Managed and producing the []byte representation.
	YamlEncodeCallback YAMLEncodeFunc
}

type ProviderEntry struct {
	GVK           k8schema.GroupVersionKind
	SchemeBuilder *scheme.Builder
	Initializer   client.Initializer
}
