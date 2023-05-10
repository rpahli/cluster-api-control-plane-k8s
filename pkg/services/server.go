package server

import (
	"context"
	"encoding/base64"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	infrav1 "sigs.k8s.io/cluster-api-provider-nested/api/infrastructure/v1beta1"
	controlplanev1 "sigs.k8s.io/cluster-api-provider-nested/controlplane/nested/api/v1beta1"
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

// Service defines struct with machine scope to reconcile HCloudMachines.
type Service struct {
	scope *scope.MachineScope
}

// NewService outs a new service with machine scope.
func NewService(scope *scope.MachineScope) *Service {
	return &Service{
		scope: scope,
	}
}

type BaseUserData struct {
	WriteFiles []bootstrapv1.File `yaml:"write_files"`
}

// Reconcile implements reconcilement of HCloudMachines.
func (s *Service) Reconcile(ctx context.Context) (res reconcile.Result, err error) {

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

	machine, err := util.GetOwnerMachine(ctx, s.scope.Client, s.scope.K8sMachine.ObjectMeta)

	namespaced := types.NamespacedName{Name: machine.Spec.Bootstrap.ConfigRef.Name, Namespace: machine.Spec.Bootstrap.ConfigRef.Namespace}
	kubeadmConfig := &bootstrapv1.KubeadmConfig{}
	err = s.scope.Client.Get(ctx, namespaced, kubeadmConfig)
	if err != nil {
		return reconcile.Result{}, err
	}

	namespaced = types.NamespacedName{Name: *machine.Spec.Bootstrap.DataSecretName, Namespace: machine.Spec.Bootstrap.ConfigRef.Namespace}
	k8sSecret := &corev1.Secret{}
	err = s.scope.Client.Get(ctx, namespaced, k8sSecret)
	if err != nil {
		return reconcile.Result{}, err
	}
	controlPlaneSecretName := fmt.Sprintf("%s-k8s", machine.Spec.Bootstrap.ConfigRef.Name)
	kubeadmConfig.Spec.ClusterConfiguration.CertificatesDir = "/etc/kubernetes/pki"
	kubeadmConfig.Spec.ClusterConfiguration.APIServer.ExtraArgs["etcd-servers"] = "https://pl-etcd:2379"
	podSpecs := controlplane.GetStaticPodSpecs(kubeadmConfig.Spec.ClusterConfiguration, &bootstrapv1.APIEndpoint{
		AdvertiseAddress: "0.0.0.0",
		BindPort:         6443,
	}, controlPlaneSecretName)

	kubeAPIServerPodSpec := podSpecs[kubeadmconstants.KubeAPIServer]
	kubeAPIServerPodSpec.Name = machine.Spec.Bootstrap.ConfigRef.Name
	kubeAPIServerPodSpec.Namespace = s.scope.Namespace

	kubeAPIServerPodSpec.SetOwnerReferences([]metav1.OwnerReference{*metav1.NewControllerRef(s.scope.K8sMachine, controlplanev1.GroupVersion.WithKind("K8sMachine"))})
	res, err = s.createPod(ctx, &kubeAPIServerPodSpec)
	if err == nil {
		return res, nil
	}
	return res, nil
}
func decodeString(encoded string) (string, error) {
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}

	return string(decoded), nil
}

// Delete implements delete method of server.
func (s *Service) Delete(ctx context.Context) (res reconcile.Result, err error) {
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

func (s *Service) createOrUpdateEtcdServerCertificates(ctx context.Context) error {
	etcdSecretName := fmt.Sprintf("%s-etcd-server", s.scope.Cluster.Name)
	namespaced := types.NamespacedName{Name: etcdSecretName, Namespace: s.scope.Namespace}
	etcdSecret := &corev1.Secret{}
	err := s.scope.Client.Get(ctx, namespaced, etcdSecret)
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
	if err := certificates.Lookup(ctx, s.scope.Client, util.ObjectKey(s.scope.Cluster)); err != nil {
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

	controllerRef := metav1.NewControllerRef(s.scope.K8sMachine, controlplanev1.GroupVersion.WithKind("K8sMachine"))
	sec := etcdKeyPair.AsSecret(util.ObjectKey(s.scope.K8sMachine), *controllerRef)
	etcdSecret.SetOwnerReferences([]metav1.OwnerReference{*metav1.NewControllerRef(s.scope.K8sMachine, controlplanev1.GroupVersion.WithKind("K8sMachine"))})
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

	err = s.scope.Client.Create(ctx, etcdSecret)
	if err != nil {
		return err
	}
	return nil
}

func (s *Service) createOrUpdateApiServerEtcdCertificates(ctx context.Context) error {
	controlPlaneEtcdSecretName := fmt.Sprintf("%s-k8s-etcd", s.scope.Machine.Spec.Bootstrap.ConfigRef.Name)
	namespaced := types.NamespacedName{Name: controlPlaneEtcdSecretName, Namespace: s.scope.Namespace}
	etcdSecret := &corev1.Secret{}
	err := s.scope.Client.Get(ctx, namespaced, etcdSecret)
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
	if err := certificates.Lookup(ctx, s.scope.Client, util.ObjectKey(s.scope.Cluster)); err != nil {
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

	controllerRef := metav1.NewControllerRef(s.scope.K8sMachine, controlplanev1.GroupVersion.WithKind("K8sMachine"))
	etcdSecret = etcdKeyPair.AsSecret(util.ObjectKey(s.scope.K8sMachine), *controllerRef)

	apiServerEtcdSecret.Data["ca.crt"] = etcdCert.KeyPair.Cert
	apiServerEtcdSecret.Data["ca.key"] = etcdCert.KeyPair.Key
	apiServerEtcdSecret.Data["server.key"] = etcdSecret.Data[secret.TLSKeyDataName]
	apiServerEtcdSecret.Data["server.crt"] = etcdSecret.Data[secret.TLSCrtDataName]

	apiServerEtcdSecret.SetOwnerReferences([]metav1.OwnerReference{*metav1.NewControllerRef(s.scope.K8sMachine, controlplanev1.GroupVersion.WithKind("K8sMachine"))})
	err = s.scope.Client.Create(ctx, apiServerEtcdSecret)
	if err != nil {
		return err
	}
	return nil
}

func (s *Service) createOrUpdateApiServerCertificates(ctx context.Context) error {
	controlPlaneSecretName := fmt.Sprintf("%s-k8s", s.scope.Machine.Spec.Bootstrap.ConfigRef.Name)
	namespaced := types.NamespacedName{Name: controlPlaneSecretName, Namespace: s.scope.Namespace}
	sec := &corev1.Secret{}
	err := s.scope.Client.Get(ctx, namespaced, sec)
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
	if err := certificates.Lookup(ctx, s.scope.Client, util.ObjectKey(s.scope.Cluster)); err != nil {
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

	controllerRef := metav1.NewControllerRef(s.scope.K8sMachine, controlplanev1.GroupVersion.WithKind("K8sMachine"))
	apiSecret := apiKeyPair.AsSecret(util.ObjectKey(s.scope.K8sMachine), *controllerRef)
	kubeletSecret := kubeletKeyPair.AsSecret(util.ObjectKey(s.scope.K8sMachine), *controllerRef)
	frontProxySecret := frontProxyKeyPair.AsSecret(util.ObjectKey(s.scope.K8sMachine), *controllerRef)
	etcdSecret := etcdKeyPair.AsSecret(util.ObjectKey(s.scope.K8sMachine), *controllerRef)

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

	apiServerSecret.SetOwnerReferences([]metav1.OwnerReference{*metav1.NewControllerRef(s.scope.K8sMachine, controlplanev1.GroupVersion.WithKind("K8sMachine"))})
	err = s.scope.Client.Create(ctx, apiServerSecret)
	if err != nil {
		return err
	}

	return nil
}

func (s *Service) createPod(ctx context.Context, pod *corev1.Pod) (res reconcile.Result, err error) {
	s.scope.SetReady(true)
	conditions.MarkTrue(s.scope.K8sMachine, infrav1.InstanceReadyCondition)
	err = s.scope.Client.Create(ctx, pod)
	return res, err
}
