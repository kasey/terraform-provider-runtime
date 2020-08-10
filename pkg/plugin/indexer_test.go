package plugin

import (
	"fmt"
	"testing"

	xpresource "github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/crossplane/crossplane-runtime/pkg/resource/fake"
	"github.com/hashicorp/terraform/providers"
	"github.com/zclconf/go-cty/cty"
	k8schema "k8s.io/apimachinery/pkg/runtime/schema"
)

func gvkFixture() k8schema.GroupVersionKind {
	return k8schema.FromAPIVersionAndKind("test.crossplane.io/v1alpha1", "FakeResource")
}

func TestIndexerAddThenLookup(t *testing.T) {
	idxr := NewIndexer()
	err := idxr.IndexFuncTable(&FuncTable{})
	if err == nil {
		t.Errorf("Expected error when attempting to index a functable with no gvk")
	}
	gvk := gvkFixture()
	f := &FuncTable{
		GVK: gvk,
	}
	err = idxr.IndexFuncTable(f)
	if err != nil {
		t.Errorf("Unexpected error calling IndexFuncTable with gvk=%s", gvk.String())
	}

	ix, err := idxr.BuildIndex()
	if err != nil {
		t.Errorf("Unexpected error calling IndexFuncTable with gvk=%s", gvk.String())
	}
	_, err = ix.APIForGVK(gvk)
	if err != nil {
		t.Errorf("Unexpected error calling APIForGVK with gvk=%s", gvk.String())
	}
}

type mockCtyDecoder struct {
	fakeError string
}

func (mock *mockCtyDecoder) DecodeCty(xpresource.Managed, cty.Value, *providers.Schema) (xpresource.Managed, error) {
	return nil, fmt.Errorf("%s", mock.fakeError)
}

type mockYAMLMarshaller struct {
	fakeError string
}

func (mock *mockYAMLMarshaller) MarshalResourceYAML(xpresource.Managed) ([]byte, error) {
	return nil, fmt.Errorf("%s", mock.fakeError)
}

func TestIndexerLayerMerging(t *testing.T) {
	idxr := NewIndexer()

	gvk := gvkFixture()
	f1 := &FuncTable{
		GVK:                    gvk,
		CtyDecoder:             &mockCtyDecoder{"f1"},
		ResourceYAMLMarshaller: &mockYAMLMarshaller{"f1"},
	}
	f2 := &FuncTable{
		GVK:        gvk,
		CtyDecoder: &mockCtyDecoder{"f2"},
	}
	idxr.IndexFuncTable(f1)
	idxr.IndexFuncTable(f2)
	ix, err := idxr.BuildIndex()
	if err != nil {
		t.Errorf("Unexpected error calling Indexer.BuildIndex(), err=%s", err.Error())
	}
	api, err := ix.APIForGVK(gvk)
	if err != nil {
		t.Errorf("Unexpected error calling Index.APIforGVK(), err=%s", err.Error())
	}
	_, err = api.DecodeCty(&fake.Managed{}, cty.Value{}, &providers.Schema{})
	if err.Error() != "f2" {
		t.Errorf("Unexpected error, the 'f2' functable added second did not occlude 'f1' added first")
	}
	_, err = api.MarshalResourceYaml(&fake.Managed{})
	if err.Error() != "f1" {
		t.Errorf("Unexpected error, expected to see f2 occlude f1, instead saw err=%s", err.Error())
	}
}
