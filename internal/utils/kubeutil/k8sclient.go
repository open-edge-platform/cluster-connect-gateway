// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package kubeutil

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"sync"

	"github.com/atomix/dazl"
	_ "github.com/atomix/dazl/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1alpha1 "github.com/open-edge-platform/cluster-connect-gateway/api/v1alpha1"
)

var (
	log = dazl.GetPackageLogger()

	scheme = runtime.NewScheme()
)

// Certkubeclient is an interface that defines the methods for managing client and server certificates
// for a given tunnel ID. It provides methods to retrieve and invalidate certificates.
type Kubeclient interface {
	GetCerts(tunnelId string) (*x509.CertPool, tls.Certificate, error)
	InvalidateCerts(tunnelId string) error
	GetKubeconfig(tunnelId string) (*api.Config, error)
	InvalidateKubeconfig(tunnelId string) error
	UpdateConnectionProbe(tunnelId string, hasSession bool) error
}

func NewInClusterClient() (Kubeclient, error) {
	// Initialize the scheme with default Kubernetes and clusterconnects types
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(v1alpha1.AddToScheme(scheme))

	config := ctrl.GetConfigOrDie()
	client, err := client.New(config, client.Options{Scheme: scheme})
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %v", err)
	}

	return &kubeclient{certStore: sync.Map{}, kcStore: sync.Map{}, client: client}, nil
}

// kubeclient is a struct that implements the Certkubeclient interface
type kubeclient struct {
	certStore sync.Map
	kcStore   sync.Map
	client    client.Client
}

type Certs struct {
	caPool     *x509.CertPool
	clientCert tls.Certificate
}

// GetCerts retrieves the client and server certificates for a given tunnel ID
// Currently not being used in preference to GetKubeconfig, but may be useful in the future
func (k *kubeclient) GetCerts(tunnelId string) (*x509.CertPool, tls.Certificate, error) {
	// Get the cluster connect object from the API server
	cc, err := k.getClusterConnect(tunnelId)
	if err != nil {
		log.Errorf("Failed to get cluster connect for tunnel %s: %v", tunnelId, err)
		return nil, tls.Certificate{}, err
	}

	// Return early if there are no client or server certificates
	if cc.Spec.ServerCertRef == nil && cc.Spec.ClientCertRef == nil {
		return nil, tls.Certificate{}, nil
	}

	// Get the client certificate from the cert store
	cached, ok := k.certStore.Load(tunnelId)
	if ok {
		return cached.(*Certs).caPool, cached.(*Certs).clientCert, nil
	}

	// Get the server certificate from the secret reference
	var caPool *x509.CertPool
	if cc.Spec.ServerCertRef != nil {
		ca, err := k.getServerCertFromSecret(cc)
		if err != nil {
			log.Errorf("Failed to get server certificate for tunnel %s: %v", tunnelId, err)
			return nil, tls.Certificate{}, err
		}
		// Create a new cert pool and append the server certificate
		caPool = x509.NewCertPool()
		caPool.AppendCertsFromPEM(ca["tls.crt"])
		if ok = caPool.AppendCertsFromPEM(ca["tls.crt"]); !ok {
			log.Warnf("Failed to append server cert for tunnel %s", tunnelId)
			return nil, tls.Certificate{}, nil
		}
	}

	// Get the client certificate from the secret reference
	var clientCert tls.Certificate
	if cc.Spec.ClientCertRef != nil {
		cca, err := k.getClientCertFromSecret(cc)
		if err != nil {
			log.Errorf("Failed to get client certificate for tunnel %s: %v", tunnelId, err)
			return nil, tls.Certificate{}, err
		}
		// Create a new client certificate from the client certificate and key
		clientCert, err = tls.X509KeyPair(cca["tls.crt"], cca["tls.key"])
		if err != nil {
			log.Errorf("Failed to create client cert keypair for tunnel %s: %v", tunnelId, err)
			return nil, tls.Certificate{}, err
		}
	}

	// Store the certs in the cert store
	k.certStore.Store(tunnelId, &Certs{caPool: caPool, clientCert: clientCert})
	return caPool, clientCert, nil
}

// InvalidateCerts invalidates the client and server certificates for a given tunnel ID
// Currently not being used in preference to InvalidateKubeconfig, but may be useful in the future
func (k *kubeclient) InvalidateCerts(tunnelId string) error {
	// Delete the certs from the cert store
	_, ok := k.certStore.Load(tunnelId)
	if ok {
		k.certStore.Delete(tunnelId)
		log.Debugf("Invalidated certs for tunnel %s", tunnelId)
		return nil
	}
	return nil
}

// GetKubeconfig retrieves the kubeconfig for a given tunnel ID
func (k *kubeclient) GetKubeconfig(tunnelId string) (*api.Config, error) {
	// Get the client certificate from the cert store
	cached, ok := k.kcStore.Load(tunnelId)
	if ok {
		return cached.(*api.Config), nil
	}

	// Get the cluster connect object from the API server
	cc, err := k.getClusterConnect(tunnelId)
	if err != nil {
		log.Errorf("Failed to get cluster connect for tunnel %s: %v", tunnelId, err)
		return nil, err
	}

	// Get the kubeconfig from the secret reference
	kubeconfig := &corev1.Secret{}
	err = k.client.Get(context.Background(), types.NamespacedName{
		Name:      cc.GetLabels()["cluster.x-k8s.io/kubeconfig-name"],
		Namespace: cc.GetLabels()["cluster.x-k8s.io/kubeconfig-namespace"],
	}, kubeconfig)
	if err != nil {
		log.Errorf("Failed to get kubeconfig for tunnel %s: %v", tunnelId, err)
		return nil, err
	}

	config, err := clientcmd.Load(kubeconfig.Data["value"])
	if err != nil {
		log.Errorf("Failed to load kubeconfig for tunnel %s: %v", tunnelId, err)
		return nil, err
	}

	k.kcStore.Store(tunnelId, config)
	return config, nil
}

// InvalidateKubeconfig invalidates the kubeconfig for a given tunnel ID
func (k *kubeclient) InvalidateKubeconfig(tunnelId string) error {
	// Delete the kubeconfig from the kubeconfig store
	_, ok := k.kcStore.Load(tunnelId)
	if ok {
		k.kcStore.Delete(tunnelId)
		log.Debugf("Invalidated kubeconfig for tunnel %s", tunnelId)
		return nil
	}
	return nil
}

func (m *kubeclient) getClusterConnect(tunnelId string) (*v1alpha1.ClusterConnect, error) {
	cc := &v1alpha1.ClusterConnect{}
	err := m.client.Get(context.Background(), types.NamespacedName{Name: tunnelId}, cc)
	if err != nil {
		return nil, err
	}
	return cc, nil
}

func (m *kubeclient) getClientCertFromSecret(cc *v1alpha1.ClusterConnect) (map[string][]byte, error) {
	secret := &corev1.Secret{}
	err := m.client.Get(context.Background(), types.NamespacedName{
		Name:      cc.Spec.ClientCertRef.Name,
		Namespace: cc.Spec.ClientCertRef.Namespace,
	}, secret)
	if err != nil {
		return nil, err
	}
	return secret.Data, nil
}

func (m *kubeclient) getServerCertFromSecret(cc *v1alpha1.ClusterConnect) (map[string][]byte, error) {
	secret := &corev1.Secret{}
	err := m.client.Get(context.Background(), types.NamespacedName{
		Name:      cc.Spec.ServerCertRef.Name,
		Namespace: cc.Spec.ServerCertRef.Namespace,
	}, secret)
	if err != nil {
		return nil, err
	}
	return secret.Data, nil
}

func (m *kubeclient) UpdateConnectionProbe(tunnelId string, hasSession bool) error {
	cc, err := m.getClusterConnect(tunnelId)
	if err != nil {
		log.Errorf("Failed to get cluster connect for tunnel %s: %v", tunnelId, err)
		return err
	}

	cc.Status.ConnectionProbe.LastProbeTimestamp = metav1.Now()

	if hasSession {
		cc.Status.ConnectionProbe.LastProbeSuccessTimestamp = cc.Status.ConnectionProbe.LastProbeTimestamp
		cc.Status.ConnectionProbe.ConsecutiveFailures = 0
		log.Debug("Connection probe successful for tunnel", tunnelId)
	} else {
		cc.Status.ConnectionProbe.ConsecutiveFailures++
		log.Debugf("Connection probe failed for tunnel %s, consecutive failures: %d", tunnelId, cc.Status.ConnectionProbe.ConsecutiveFailures)
	}

	// modify clusterconnection with the health info
	err = m.client.Status().Patch(context.Background(), cc, client.MergeFrom(cc.DeepCopy()))
	if err != nil {
		log.Errorf("Failed to patch cluster connect status for tunnel %s: %v", tunnelId, err)
		return fmt.Errorf("failed to patch cluster connect status for tunnel %s: %v", tunnelId, err)
	}

	log.Debugf("Updated connection probe for tunnel %s", tunnelId)

	return nil
}
