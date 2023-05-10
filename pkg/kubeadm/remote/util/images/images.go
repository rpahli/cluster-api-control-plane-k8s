package images

import (
	"fmt"
	kubeadmutil "sigs.k8s.io/cluster-api-provider-nested/pkg/kubeadm/remote/util"
	bootstrapv1 "sigs.k8s.io/cluster-api/bootstrap/kubeadm/api/v1beta1"
)

const (
	DefaultImageRepository = "registry.k8s.io"
)

// GetGenericImage generates and returns a platform agnostic image (backed by manifest list)
func GetGenericImage(prefix, image, tag string) string {
	return fmt.Sprintf("%s/%s:%s", prefix, image, tag)
}

// GetKubernetesImage generates and returns the image for the components managed in the Kubernetes main repository,
// including the control-plane components and kube-proxy.
func GetKubernetesImage(image string, cfg *bootstrapv1.ClusterConfiguration) string {
	repoPrefix := cfg.ImageRepository
	if cfg.ImageRepository == "" {
		repoPrefix = DefaultImageRepository
	}
	kubernetesImageTag := kubeadmutil.KubernetesVersionToImageTag(cfg.KubernetesVersion)
	return GetGenericImage(repoPrefix, image, kubernetesImageTag)
}
