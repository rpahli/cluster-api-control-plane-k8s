package kubeadm

import (
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	kubeadmapiv1 "sigs.k8s.io/cluster-api-provider-nested/pkg/kubeadm/api/v1beta3"
	kubeadmscheme "sigs.k8s.io/cluster-api-provider-nested/pkg/kubeadm/remote/scheme"
	"sigs.k8s.io/cluster-api-provider-nested/pkg/kubeadm/remote/strict"
	kubeadmutil "sigs.k8s.io/cluster-api-provider-nested/pkg/kubeadm/remote/util"
)

// BytesToInitConfiguration converts a byte slice to an internal, defaulted and validated InitConfiguration object.
// The map may contain many different YAML documents. These YAML documents are parsed one-by-one
// and well-known ComponentConfig GroupVersionKinds are stored inside of the internal InitConfiguration struct.
// The resulting InitConfiguration is then dynamically defaulted and validated prior to return.
func BytesToInitConfiguration(b []byte) (*kubeadmapiv1.InitConfiguration, error) {
	gvkmap, err := kubeadmutil.SplitYAMLDocuments(b)
	if err != nil {
		return nil, err
	}

	return documentMapToInitConfiguration(gvkmap, false)
}

// documentMapToInitConfiguration converts a map of GVKs and YAML documents to defaulted and validated configuration object.
func documentMapToInitConfiguration(gvkmap kubeadmapiv1.DocumentMap, allowDeprecated bool) (*kubeadmapiv1.InitConfiguration, error) {
	var initcfg *kubeadmapiv1.InitConfiguration
	var clustercfg *kubeadmapiv1.ClusterConfiguration

	for gvk, fileContent := range gvkmap {
		// first, check if this GVK is supported and possibly not deprecated
		if err := validateSupportedVersion(gvk.GroupVersion(), allowDeprecated); err != nil {
			return nil, err
		}

		// verify the validity of the YAML
		if err := strict.VerifyUnmarshalStrict([]*runtime.Scheme{kubeadmscheme.Scheme}, gvk, fileContent); err != nil {
			klog.Warning(err.Error())
		}

		if kubeadmutil.GroupVersionKindsHasInitConfiguration(gvk) {
			// Set initcfg to an empty struct value the deserializer will populate
			initcfg = &kubeadmapiv1.InitConfiguration{}
			// Decode the bytes into the internal struct. Under the hood, the bytes will be unmarshalled into the
			// right external version, defaulted, and converted into the internal version.
			if err := runtime.DecodeInto(kubeadmscheme.Codecs.UniversalDecoder(), fileContent, initcfg); err != nil {
				return nil, err
			}
			continue
		}
		if kubeadmutil.GroupVersionKindsHasClusterConfiguration(gvk) {
			// Set clustercfg to an empty struct value the deserializer will populate
			clustercfg = &kubeadmapiv1.ClusterConfiguration{}
			// Decode the bytes into the internal struct. Under the hood, the bytes will be unmarshalled into the
			// right external version, defaulted, and converted into the internal version.
			if err := runtime.DecodeInto(kubeadmscheme.Codecs.UniversalDecoder(), fileContent, clustercfg); err != nil {
				return nil, err
			}
			continue
		}

		// If the group is neither a kubeadm core type or of a supported component config group, we dump a warning about it being ignored
		/*		if !componentconfigs.Scheme.IsGroupRegistered(gvk.Group) {
				klog.Warningf("[config] WARNING: Ignored YAML document with GroupVersionKind %v\n", gvk)
			}*/
	}

	// Enforce that InitConfiguration and/or ClusterConfiguration has to exist among the YAML documents
	if initcfg == nil && clustercfg == nil {
		return nil, errors.New("no InitConfiguration or ClusterConfiguration kind was found in the YAML file")
	}

	// If InitConfiguration wasn't given, default it by creating an external struct instance, default it and convert into the internal type
	if initcfg == nil {
		extinitcfg := &kubeadmapiv1.InitConfiguration{}
		kubeadmscheme.Scheme.Default(extinitcfg)
		// Set initcfg to an empty struct value the deserializer will populate
		initcfg = &kubeadmapiv1.InitConfiguration{}
		if err := kubeadmscheme.Scheme.Convert(extinitcfg, initcfg, nil); err != nil {
			return nil, err
		}
	}
	// If ClusterConfiguration was given, populate it in the InitConfiguration struct
	if clustercfg != nil {
		initcfg.ClusterConfiguration = *clustercfg
	} else {
		// Populate the internal InitConfiguration.ClusterConfiguration with defaults
		extclustercfg := &kubeadmapiv1.ClusterConfiguration{}
		kubeadmscheme.Scheme.Default(extclustercfg)
		if err := kubeadmscheme.Scheme.Convert(extclustercfg, &initcfg.ClusterConfiguration, nil); err != nil {
			return nil, err
		}
	}

	/*
		// Load any component configs
		if err := componentconfigs.FetchFromDocumentMap(&initcfg.ClusterConfiguration, gvkmap); err != nil {
			return nil, err
		}

		// Applies dynamic defaults to settings not provided with flags
		if err := SetInitDynamicDefaults(initcfg); err != nil {
			return nil, err
		}

		// Validates cfg (flags/configs + defaults + dynamic defaults)
		if err := validation.ValidateInitConfiguration(initcfg).ToAggregate(); err != nil {
			return nil, err
		}
	*/
	return initcfg, nil
}
