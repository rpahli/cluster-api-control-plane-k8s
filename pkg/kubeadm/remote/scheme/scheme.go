package scheme

import (
	"k8s.io/api/flowcontrol/v1beta3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	kubeadm "sigs.k8s.io/cluster-api-provider-nested/pkg/kubeadm/api/v1beta3"
)

// Scheme is the runtime.Scheme to which all kubeadm api types are registered.
var Scheme = runtime.NewScheme()

// Codecs provides access to encoding and decoding for the scheme.
var Codecs = serializer.NewCodecFactory(Scheme)

func init() {
	metav1.AddToGroupVersion(Scheme, schema.GroupVersion{Version: "v1"})
	AddToScheme(Scheme)
}

// AddToScheme builds the kubeadm scheme using all known versions of the kubeadm api.
func AddToScheme(scheme *runtime.Scheme) {
	utilruntime.Must(kubeadm.AddToScheme(scheme))
	// utilruntime.Must(v1beta3.AddToScheme(scheme))
	utilruntime.Must(scheme.SetVersionPriority(v1beta3.SchemeGroupVersion))
}
