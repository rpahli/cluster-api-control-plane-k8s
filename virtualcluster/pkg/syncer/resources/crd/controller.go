/*
Copyright 2021 The Kubernetes Authors.

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

package crd

import (
	"context"
	"fmt"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/informers"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	rinformer "sigs.k8s.io/controller-runtime/pkg/cache"
	dclient "sigs.k8s.io/controller-runtime/pkg/client"

	vcclient "sigs.k8s.io/cluster-api-provider-nested/virtualcluster/pkg/client/clientset/versioned"
	vcinformers "sigs.k8s.io/cluster-api-provider-nested/virtualcluster/pkg/client/informers/externalversions/tenancy/v1alpha1"
	"sigs.k8s.io/cluster-api-provider-nested/virtualcluster/pkg/syncer/apis/config"
	"sigs.k8s.io/cluster-api-provider-nested/virtualcluster/pkg/syncer/constants"
	"sigs.k8s.io/cluster-api-provider-nested/virtualcluster/pkg/syncer/manager"
	pa "sigs.k8s.io/cluster-api-provider-nested/virtualcluster/pkg/syncer/patrol"
	uw "sigs.k8s.io/cluster-api-provider-nested/virtualcluster/pkg/syncer/uwcontroller"
	"sigs.k8s.io/cluster-api-provider-nested/virtualcluster/pkg/util/listener"
	mc "sigs.k8s.io/cluster-api-provider-nested/virtualcluster/pkg/util/mccontroller"
	"sigs.k8s.io/cluster-api-provider-nested/virtualcluster/pkg/util/plugin"
)

var SchemeGroupVersion = schema.GroupVersion{
	Group:   "apiextensions.k8s.io",
	Version: "v1",
}

var (
	SchemeBuilder      runtime.SchemeBuilder
	localSchemeBuilder = &SchemeBuilder
)

func init() {
	localSchemeBuilder.Register(addKnownTypes)
	_ = localSchemeBuilder.AddToScheme(scheme.Scheme)

	plugin.SyncerResourceRegister.Register(&plugin.Registration{
		ID: "crd",
		InitFn: func(ctx *plugin.InitContext) (interface{}, error) {
			return NewCrdController(ctx.Config.(*config.SyncerConfiguration), ctx.Client, ctx.Informer, ctx.VCClient, ctx.VCInformer, manager.ResourceSyncerOptions{})
		},
		Disable: true,
	})
}

// Adds the list of known types to the given scheme
func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(
		SchemeGroupVersion,
		&apiextensionsv1.CustomResourceDefinition{},
		&apiextensionsv1.CustomResourceDefinitionList{},
	)
	scheme.AddKnownTypes(
		SchemeGroupVersion,
		&metav1.Status{},
	)
	metav1.AddToGroupVersion(
		scheme,
		SchemeGroupVersion,
	)
	return nil
}

type controller struct {
	manager.BaseResourceSyncer
	config    *config.SyncerConfiguration
	crdSynced cache.InformerSynced
	vcSynced  cache.InformerSynced
	// Super cluster restful config
	restConfig  *restclient.Config
	superClient dclient.Client
	crdcache    rinformer.Cache
	informer    rinformer.Informer
}

func NewCrdController(config *config.SyncerConfiguration,
	client clientset.Interface,
	informer informers.SharedInformerFactory,
	vcClient vcclient.Interface,
	vcInformer vcinformers.VirtualClusterInformer,
	options manager.ResourceSyncerOptions) (manager.ResourceSyncer, error) {
	var err error
	var sc dclient.Client

	c := &controller{
		BaseResourceSyncer: manager.BaseResourceSyncer{
			Config: config,
		},
		config:     config,
		restConfig: config.RestConfig,
		crdcache:   nil,
	}

	if config.RestConfig == nil {
		return nil, fmt.Errorf("cannot get super control plane restful config")
	}
	sc, err = dclient.New(config.RestConfig, dclient.Options{})
	if err != nil {
		return nil, err
	}
	c.superClient = sc

	if config.RestConfig == nil {
		return nil, fmt.Errorf("cannot get super control plane restful config")
	}

	c.crdcache, err = rinformer.New(config.RestConfig, rinformer.Options{})
	if err != nil {
		return nil, err
	}
	c.informer, err = c.crdcache.GetInformer(context.Background(), &apiextensionsv1.CustomResourceDefinition{})
	if err != nil {
		return nil, err
	}

	c.MultiClusterController, err = mc.NewMCController(&apiextensionsv1.CustomResourceDefinition{}, &apiextensionsv1.CustomResourceDefinitionList{}, c,
		mc.WithMaxConcurrentReconciles(constants.DwsControllerWorkerLow), mc.WithOptions(options.MCOptions))
	if err != nil {
		return nil, fmt.Errorf("failed to create crd mc controller: %v", err)
	}

	if options.IsFake {
		c.crdSynced = func() bool { return true }
		c.vcSynced = func() bool { return true }
	} else {
		c.crdSynced = c.informer.HasSynced
		c.vcSynced = vcInformer.Informer().HasSynced
	}

	c.UpwardController, err = uw.NewUWController(&apiextensionsv1.CustomResourceDefinition{}, c, uw.WithOptions(options.UWOptions))
	if err != nil {
		return nil, err
	}
	c.Patroller, err = pa.NewPatroller(&apiextensionsv1.CustomResourceDefinition{}, c, pa.WithOptions(options.PatrolOptions))
	if err != nil {
		return nil, fmt.Errorf("failed to create crd patroller: %v", err)
	}

	c.informer.AddEventHandler(
		cache.FilteringResourceEventHandler{
			FilterFunc: func(obj interface{}) bool {
				switch t := obj.(type) {
				case *apiextensionsv1.CustomResourceDefinition:
					return publicCRD(t)
				case cache.DeletedFinalStateUnknown:
					if e, ok := t.Obj.(*apiextensionsv1.CustomResourceDefinition); ok {
						return publicCRD(e)
					}
					utilruntime.HandleError(fmt.Errorf("unable to convert object %v to *apiextensionsv1.CustomResourceDefinition", obj))
					return false
				default:
					utilruntime.HandleError(fmt.Errorf("unable to handle object in super control plane CRD controller: %v", obj))
					return false
				}
			},
			Handler: cache.ResourceEventHandlerFuncs{
				AddFunc: c.enqueueCRD,
				UpdateFunc: func(oldObj, newObj interface{}) {
					newCRD := newObj.(*apiextensionsv1.CustomResourceDefinition)
					oldCRD := oldObj.(*apiextensionsv1.CustomResourceDefinition)
					if newCRD.ResourceVersion != oldCRD.ResourceVersion {
						c.enqueueCRD(newObj)
					}
				},
				DeleteFunc: c.enqueueCRD,
			},
		})
	return c, nil
}

func (c *controller) GetMCController() *mc.MultiClusterController {
	return c.MultiClusterController
}

func (c *controller) GetListener() listener.ClusterChangeListener {
	return listener.NewMCControllerListener(c.MultiClusterController, mc.WatchOptions{})
}

func publicCRD(e *apiextensionsv1.CustomResourceDefinition) bool {
	// We only backpopulate specific crds to tenant control planes
	return e.Labels[constants.PublicObjectKey] == "true"
}

func (c *controller) enqueueCRD(obj interface{}) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("couldn't get key for object %v: %v", obj, err))
		return
	}
	clusterNames := c.MultiClusterController.GetClusterNames()
	if len(clusterNames) == 0 {
		return
	}
	for _, clusterName := range clusterNames {
		c.UpwardController.AddToQueue(clusterName + "/" + key)
	}
}
