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

package controller

import (
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"

	crossplaneapis "github.com/crossplane/crossplane/apis"
	"github.com/crossplane/terraform-provider-runtime/pkg/client"
	"github.com/crossplane/terraform-provider-runtime/pkg/registry"
)

func StartTerraformManager(r *registry.Registry, opts ctrl.Options, ropts *client.RuntimeOptions, log logging.Logger) error {
	cfg, err := ctrl.GetConfig()
	if err != nil {
		return errors.Wrap(err, "Cannot get API server rest config")
	}
	mgr, err := ctrl.NewManager(cfg, opts)
	if err != nil {
		return errors.Wrap(err, "Cannot create controller manager")
	}
	err = crossplaneapis.AddToScheme(mgr.GetScheme())
	if err != nil {
		return errors.Wrap(err, "Cannot add core Crossplane APIs to scheme")
	}
	/*
		err = apis.AddToScheme(mgr.GetScheme())
		if err != nil {
			return errors.Wrap(err, "Cannot add Template APIs to scheme")
		}
	*/
	for _, sb := range r.GetSchemeBuilders() {
		if err := sb.AddToScheme(mgr.GetScheme()); err != nil {
			return err
		}
	}
	p, err := r.GetProviderEntry()
	if err != nil {
		return errors.Wrap(err, "Failed to get ProviderEntry from StartTerraformManager")
	}
	p.SchemeBuilder.AddToScheme(mgr.GetScheme())
	pool := client.NewProviderPool(p.Initializer, ropts)
	for _, configure := range r.GetReconcilerConfigurers() {
		if err := configure(mgr, log, r, pool); err != nil {
			return err
		}
	}
	err = mgr.Start(ctrl.SetupSignalHandler())
	if err != nil {
		return errors.Wrap(err, "Cannot start controller manager")
	}
	return nil
}
