package apiserver

import (
	"context"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	controlplanev1 "sigs.k8s.io/cluster-api-provider-nested/api/controlplane/v1beta1"
	infrav1 "sigs.k8s.io/cluster-api-provider-nested/api/infrastructure/v1beta1"
	"sigs.k8s.io/cluster-api-provider-nested/controlplane/nested/certificate"
	kubeadmconstants "sigs.k8s.io/cluster-api-provider-nested/pkg/kubeadm/remote"
	"sigs.k8s.io/cluster-api-provider-nested/pkg/kubeadm/remote/controlplane"
	"sigs.k8s.io/cluster-api-provider-nested/pkg/scope"
	bootstrapv1 "sigs.k8s.io/cluster-api/bootstrap/kubeadm/api/v1beta1"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/certs"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/secret"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// ApiServerService defines struct with machine scope to reconcile HCloudMachines.
type ApiServerService struct {
	scope *scope.ControlPlaneScope
}

// NewApiServerService outs a new service with machine scope.
func NewApiServerService(scope *scope.ControlPlaneScope) *ApiServerService {
	return &ApiServerService{
		scope: scope,
	}
}

// Reconcile implements reconcilement of HCloudMachines.
func (s *ApiServerService) Reconcile(ctx context.Context) (res reconcile.Result, err error) {

	err = s.createOrUpdateEtcdServerCertificates(ctx)
	if err != nil {
		return res, err
	}
	err = s.createOrUpdateApiServerCertificates(ctx)
	if err != nil {
		return res, err
	}
	err = s.createOrUpdateApiServerEtcdCertificates(ctx)
	if err != nil {
		return res, err
	}

	controlPlaneSecretName := fmt.Sprintf("%s-k8s", s.scope.Cluster.Name)
	s.scope.K8sControlPlane.Spec.ClusterConfiguration.CertificatesDir = "/etc/kubernetes/pki"
	s.scope.K8sControlPlane.Spec.ClusterConfiguration.APIServer.ExtraArgs["etcd-servers"] = "https://pl-etcd:2379"
	podSpecs := controlplane.GetStaticPodSpecs(&s.scope.K8sControlPlane.Spec.ClusterConfiguration, &bootstrapv1.APIEndpoint{
		AdvertiseAddress: "0.0.0.0",
		BindPort:         6443,
	}, controlPlaneSecretName)

	kubeAPIServerPodSpec := podSpecs[kubeadmconstants.KubeAPIServer]
	kubeAPIServerPodSpec.Name = s.scope.K8sControlPlane.Name
	kubeAPIServerPodSpec.Namespace = s.scope.Namespace

	kubeAPIServerPodSpec.SetOwnerReferences([]metav1.OwnerReference{*metav1.NewControllerRef(s.scope.K8sControlPlane, controlplanev1.GroupVersion.WithKind("K8sMachine"))})
	res, err = s.createPod(ctx, &kubeAPIServerPodSpec)
	if err == nil {
		return res, nil
	}
	return res, nil
}

// Delete implements delete method of server.
func (s *ApiServerService) Delete(ctx context.Context) (res reconcile.Result, err error) {
	return res, nil
}

func getEtcdServers(name, namespace string, replicas int32) (etcdServers []string) {
	var i int32
	for ; i < replicas; i++ {
		etcdServers = append(etcdServers, fmt.Sprintf("%s-%d.%s.%s.svc", name, i, name, namespace))
	}
	etcdServers = append(etcdServers, name, "localhost")
	return etcdServers
}

func (s *ApiServerService) createOrUpdateEtcdServerCertificates(ctx context.Context) error {
	etcdSecretName := fmt.Sprintf("%s-etcd-server", s.scope.Cluster.Name)
	namespaced := types.NamespacedName{Name: etcdSecretName, Namespace: s.scope.Namespace}
	etcdSecret := &corev1.Secret{}
	err := s.scope.ManagerClient.Get(ctx, namespaced, etcdSecret)
	isNew := false
	if err != nil {
		// return error if error not equal not found
		if apierrors.IsNotFound(err) {
			isNew = true
		} else {
			return err
		}
	}
	if !isNew {
		return nil
	}
	certificates := secret.NewCertificatesForInitialControlPlane(nil)
	if err := certificates.Lookup(ctx, s.scope.ManagerClient, util.ObjectKey(s.scope.Cluster)); err != nil {
		return err
	}
	etcdCert := certificates.GetByPurpose(secret.EtcdCA)
	if etcdCert == nil {
		return fmt.Errorf("could not fetch EtcdCA")
	}
	etcdCrt, err := certs.DecodeCertPEM(etcdCert.KeyPair.Cert)
	if err != nil {
		return err
	}

	etcdKey, err := certs.DecodePrivateKeyPEM(etcdCert.KeyPair.Key)
	if err != nil {
		return err
	}

	//TODO replicas
	etcdKeyPair, err := certificate.NewEtcdServerCertAndKey(&certificate.KeyPair{Cert: etcdCrt, Key: etcdKey}, getEtcdServers("pl-etcd", s.scope.Namespace, 3))
	if err != nil {
		return err
	}
	etcdSecret = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      etcdSecretName,
			Namespace: s.scope.Namespace,
		},
		Data:       map[string][]byte{},
		StringData: map[string]string{},
		Type:       "",
	}

	controllerRef := metav1.NewControllerRef(s.scope.K8sControlPlane, controlplanev1.GroupVersion.WithKind("K8sControlPlane"))
	sec := etcdKeyPair.AsSecret(util.ObjectKey(s.scope.K8sControlPlane), *controllerRef)
	etcdSecret.SetOwnerReferences([]metav1.OwnerReference{*metav1.NewControllerRef(s.scope.K8sControlPlane, controlplanev1.GroupVersion.WithKind("K8sControlPlane"))})
	etcdSecret.Data["ca.crt"] = etcdCert.KeyPair.Cert
	etcdSecret.Data["peer-ca.crt"] = etcdCert.KeyPair.Cert
	etcdSecret.Data["server-ca.crt"] = etcdCert.KeyPair.Cert
	etcdSecret.Data["etcd-client-ca.crt"] = etcdCert.KeyPair.Cert
	etcdSecret.Data["ca.key"] = etcdCert.KeyPair.Key
	etcdSecret.Data["server.key"] = sec.Data[secret.TLSKeyDataName]
	etcdSecret.Data["server.crt"] = sec.Data[secret.TLSCrtDataName]
	etcdSecret.Data["peer.key"] = sec.Data[secret.TLSKeyDataName]
	etcdSecret.Data["peer.crt"] = sec.Data[secret.TLSCrtDataName]
	etcdSecret.Data["etcd-client.key"] = sec.Data[secret.TLSKeyDataName]
	etcdSecret.Data["etcd-client.crt"] = sec.Data[secret.TLSCrtDataName]

	err = s.scope.ManagerClient.Create(ctx, etcdSecret)
	if err != nil {
		return err
	}
	return nil
}

func (s *ApiServerService) createOrUpdateApiServerEtcdCertificates(ctx context.Context) error {
	controlPlaneEtcdSecretName := fmt.Sprintf("%s-k8s-etcd", s.scope.Cluster.Name)
	namespaced := types.NamespacedName{Name: controlPlaneEtcdSecretName, Namespace: s.scope.Namespace}
	etcdSecret := &corev1.Secret{}
	err := s.scope.ManagerClient.Get(ctx, namespaced, etcdSecret)
	isNew := false
	if err != nil {
		// return error if error not equal not found
		if apierrors.IsNotFound(err) {
			isNew = true
		} else {
			return err
		}
	}
	if !isNew {
		return nil
	}
	certificates := secret.NewCertificatesForInitialControlPlane(nil)
	if err := certificates.Lookup(ctx, s.scope.ManagerClient, util.ObjectKey(s.scope.Cluster)); err != nil {
		return err
	}

	etcdCert := certificates.GetByPurpose(secret.EtcdCA)
	if etcdCert == nil {
		return fmt.Errorf("could not fetch EtcdCA")
	}

	etcdCrt, err := certs.DecodeCertPEM(etcdCert.KeyPair.Cert)
	if err != nil {
		return err
	}

	etcdKey, err := certs.DecodePrivateKeyPEM(etcdCert.KeyPair.Key)
	if err != nil {
		return err
	}

	//TODO replicas
	etcdKeyPair, err := certificate.NewEtcdServerCertAndKey(&certificate.KeyPair{Cert: etcdCrt, Key: etcdKey}, getEtcdServers(s.scope.Cluster.GetName(), s.scope.Cluster.GetNamespace(), 3))
	if err != nil {
		return err
	}

	apiServerEtcdSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      controlPlaneEtcdSecretName,
			Namespace: s.scope.Namespace,
		},
		Data:       map[string][]byte{},
		StringData: map[string]string{},
		Type:       "",
	}

	controllerRef := metav1.NewControllerRef(s.scope.K8sControlPlane, controlplanev1.GroupVersion.WithKind("K8sControlPlane"))
	etcdSecret = etcdKeyPair.AsSecret(util.ObjectKey(s.scope.K8sControlPlane), *controllerRef)

	apiServerEtcdSecret.Data["ca.crt"] = etcdCert.KeyPair.Cert
	apiServerEtcdSecret.Data["ca.key"] = etcdCert.KeyPair.Key
	apiServerEtcdSecret.Data["server.key"] = etcdSecret.Data[secret.TLSKeyDataName]
	apiServerEtcdSecret.Data["server.crt"] = etcdSecret.Data[secret.TLSCrtDataName]

	apiServerEtcdSecret.SetOwnerReferences([]metav1.OwnerReference{*metav1.NewControllerRef(s.scope.K8sControlPlane, controlplanev1.GroupVersion.WithKind("K8sControlPlane"))})
	err = s.scope.ManagerClient.Create(ctx, apiServerEtcdSecret)
	if err != nil {
		return err
	}
	return nil
}

func (s *ApiServerService) createOrUpdateApiServerCertificates(ctx context.Context) error {
	controlPlaneSecretName := fmt.Sprintf("%s-k8s", s.scope.Cluster.Name)
	namespaced := types.NamespacedName{Name: controlPlaneSecretName, Namespace: s.scope.Namespace}
	sec := &corev1.Secret{}
	err := s.scope.ManagerClient.Get(ctx, namespaced, sec)
	isNew := false
	if err != nil {
		// return error if error not equal not found
		if apierrors.IsNotFound(err) {
			isNew = true
		} else {
			return err
		}
	}
	if !isNew {
		return nil
	}
	certificates := secret.NewCertificatesForInitialControlPlane(nil)
	if err := certificates.Lookup(ctx, s.scope.ManagerClient, util.ObjectKey(s.scope.Cluster)); err != nil {
		return err
	}
	cacert := certificates.GetByPurpose(secret.ClusterCA)
	if cacert == nil {
		fmt.Errorf("could not fetch ClusterCA")
	}
	etcdCert := certificates.GetByPurpose(secret.EtcdCA)
	if etcdCert == nil {
		return fmt.Errorf("could not fetch EtcdCA")
	}

	etcdCrt, err := certs.DecodeCertPEM(etcdCert.KeyPair.Cert)
	if err != nil {
		return err
	}

	etcdKey, err := certs.DecodePrivateKeyPEM(etcdCert.KeyPair.Key)
	if err != nil {
		return err
	}

	serviceAccount := certificates.GetByPurpose(secret.ServiceAccount)
	if serviceAccount == nil {
		fmt.Errorf("could not fetch ServiceAccountCA")
	}

	frontProxy := certificates.GetByPurpose(secret.FrontProxyCA)
	if frontProxy == nil {
		fmt.Errorf("could not fetch FrontProxyCA")
	}

	frontProxyCrt, err := certs.DecodeCertPEM(frontProxy.KeyPair.Cert)
	if err != nil {
		return err
	}

	frontProxyKey, err := certs.DecodePrivateKeyPEM(frontProxy.KeyPair.Key)
	if err != nil {
		return err
	}

	cacrt, err := certs.DecodeCertPEM(cacert.KeyPair.Cert)
	if err != nil {
		return err
	}
	cakey, err := certs.DecodePrivateKeyPEM(cacert.KeyPair.Key)
	if err != nil {
		return err
	}

	apiKeyPair, err := certificate.NewAPIServerCrtAndKey(&certificate.KeyPair{Cert: cacrt, Key: cakey}, s.scope.Cluster.GetName(), "", s.scope.Cluster.Spec.ControlPlaneEndpoint.Host)
	if err != nil {
		return err
	}
	kubeletKeyPair, err := certificate.NewAPIServerKubeletClientCertAndKey(&certificate.KeyPair{Cert: cacrt, Key: cakey}, s.scope.Cluster.Namespace)
	if err != nil {
		return err
	}
	frontProxyKeyPair, err := certificate.NewFrontProxyClientCertAndKey(&certificate.KeyPair{Cert: frontProxyCrt, Key: frontProxyKey})
	if err != nil {
		return err
	}

	//TODO replicas
	etcdKeyPair, err := certificate.NewEtcdServerCertAndKey(&certificate.KeyPair{Cert: etcdCrt, Key: etcdKey}, getEtcdServers(s.scope.Cluster.GetName(), s.scope.Cluster.GetNamespace(), 3))
	if err != nil {
		return err
	}

	apiServerSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      controlPlaneSecretName,
			Namespace: s.scope.Namespace,
		},
		Data:       map[string][]byte{},
		StringData: map[string]string{},
		Type:       "",
	}

	controllerRef := metav1.NewControllerRef(s.scope.K8sControlPlane, controlplanev1.GroupVersion.WithKind("K8sMachine"))
	apiSecret := apiKeyPair.AsSecret(util.ObjectKey(s.scope.K8sControlPlane), *controllerRef)
	kubeletSecret := kubeletKeyPair.AsSecret(util.ObjectKey(s.scope.K8sControlPlane), *controllerRef)
	frontProxySecret := frontProxyKeyPair.AsSecret(util.ObjectKey(s.scope.K8sControlPlane), *controllerRef)
	etcdSecret := etcdKeyPair.AsSecret(util.ObjectKey(s.scope.K8sControlPlane), *controllerRef)

	apiServerSecret.Data["apiserver.key"] = apiSecret.Data[secret.TLSKeyDataName]
	apiServerSecret.Data["apiserver.crt"] = apiSecret.Data[secret.TLSCrtDataName]
	apiServerSecret.Data["apiserver-kubelet-client.key"] = kubeletSecret.Data[secret.TLSKeyDataName]
	apiServerSecret.Data["apiserver-kubelet-client.crt"] = kubeletSecret.Data[secret.TLSCrtDataName]

	apiServerSecret.Data["apiserver-etcd-client.key"] = etcdSecret.Data[secret.TLSKeyDataName]
	apiServerSecret.Data["apiserver-etcd-client.crt"] = etcdSecret.Data[secret.TLSCrtDataName]

	apiServerSecret.Data["ca.key"] = cacert.KeyPair.Key
	apiServerSecret.Data["ca.crt"] = cacert.KeyPair.Cert

	apiServerSecret.Data["front-proxy-ca.key"] = frontProxy.KeyPair.Key
	apiServerSecret.Data["front-proxy-ca.crt"] = frontProxy.KeyPair.Cert

	apiServerSecret.Data["front-proxy-client.key"] = frontProxySecret.Data[secret.TLSKeyDataName]
	apiServerSecret.Data["front-proxy-client.crt"] = frontProxySecret.Data[secret.TLSCrtDataName]

	apiServerSecret.Data["sa.key"] = serviceAccount.KeyPair.Key
	apiServerSecret.Data["sa.pub"] = serviceAccount.KeyPair.Cert

	apiServerSecret.SetOwnerReferences([]metav1.OwnerReference{*metav1.NewControllerRef(s.scope.K8sControlPlane, controlplanev1.GroupVersion.WithKind("K8sMachine"))})
	err = s.scope.ManagerClient.Create(ctx, apiServerSecret)
	if err != nil {
		return err
	}

	return nil
}

func (s *ApiServerService) createPod(ctx context.Context, pod *corev1.Pod) (res reconcile.Result, err error) {
	s.scope.SetReady(true)
	conditions.MarkTrue(s.scope.K8sControlPlane, infrav1.InstanceReadyCondition)
	err = s.scope.ManagerClient.Create(ctx, pod)
	return res, err
}
