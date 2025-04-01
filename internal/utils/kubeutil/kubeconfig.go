// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package kubeutil

import (
	"context"
	"crypto"
	"crypto/x509"
	"os"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/cluster-api/util/certs"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// ClusterCA is the secret name suffix for APIServer CA.
	ClusterCA = "ca"

	// ClientClusterCA is the secret name suffix for APIServer CA.
	ClientClusterCA = "cca"

	// TLSKeyDataName is the key used to store a TLS private key in the secret's data field.
	TLSKeyDataName = "tls.key"

	// TLSCrtDataName is the key used to store a TLS certificate in the secret's data field.
	TLSCrtDataName = "tls.crt"

	KubeconfigDataName = "value"

	ApiServerCA = "apiServerCA"
)

const (
	privateCASecNameEnv      = "PRIVATE_CA_SECRET_NAME"
	privateCASecNamespaceEnv = "PRIVATE_CA_SECRET_NAMESPACE"
)

func GenerateKubeconfig(ctx context.Context, c client.Client, clusterName, clusterNamespace, server string) ([]byte, error) {
	serverCA, err := getCertSecret(ctx, c, clusterName, clusterNamespace, ClusterCA)
	if err != nil {
		return nil, err
	}

	clientClusterCA, err := getCertSecret(ctx, c, clusterName, clusterNamespace, ClientClusterCA)
	if err != nil {
		return nil, err
	}

	clientCACert, err := certs.DecodeCertPEM(clientClusterCA.Data[TLSCrtDataName])
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode CA Cert")
	} else if clientCACert == nil {
		return nil, errors.New("certificate not found in config")
	}

	clientCAKey, err := certs.DecodePrivateKeyPEM(clientClusterCA.Data[TLSKeyDataName])
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode private key")
	} else if clientCAKey == nil {
		return nil, errors.New("CA private key not found")
	}

	serverCACert, err := certs.DecodeCertPEM(serverCA.Data[TLSCrtDataName])
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode CA Cert")
	} else if serverCACert == nil {
		return nil, errors.New("certificate not found in config")
	}

	cfg, err := newKubeconfig(clusterName, server, clientCACert, clientCAKey, serverCACert)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate a kubeconfig")
	}

	out, err := clientcmd.Write(*cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to serialize config to yaml")
	}

	return out, nil
}

func GetAPIServerCA(ctx context.Context, c client.Client) ([]byte, error) {

	privateCASecretName := os.Getenv(privateCASecNameEnv)
	privateCASecretNamespace := os.Getenv(privateCASecNamespaceEnv)
	if privateCASecretName == "" || privateCASecretNamespace == "" {
		return nil, errors.New("private CA secret name or namespace not set")
	}

	secret := &corev1.Secret{}
	secretKey := client.ObjectKey{Namespace: privateCASecretNamespace, Name: privateCASecretName}

	if err := c.Get(ctx, secretKey, secret); err != nil {
		return nil, err
	}

	caCrt, exists := secret.Data["ca.crt"]
	if !exists {
		caCrt, exists = secret.Data["tls.crt"]
		if !exists {
			return nil, errors.New("neither ca.crt nor tls.crt found in secret")
		}
	}

	return caCrt, nil
}

func newKubeconfig(clusterName, server string, clientCACert *x509.Certificate, clientCAKey crypto.Signer, serverCACert *x509.Certificate) (*api.Config, error) {
	cfg := &certs.Config{
		CommonName:   "kubernetes-admin",
		Organization: []string{"system:masters"},
		Usages:       []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	clientKey, err := certs.NewPrivateKey()
	if err != nil {
		return nil, errors.Wrap(err, "unable to create private key")
	}

	clientCert, err := cfg.NewSignedCert(clientKey, clientCACert, clientCAKey)
	if err != nil {
		return nil, errors.Wrap(err, "unable to sign certificate")
	}

	userName := clusterName + "-admin"
	contextName := userName + "@" + clusterName

	return &api.Config{
		Clusters: map[string]*api.Cluster{
			clusterName: {
				Server:                   server,
				CertificateAuthorityData: certs.EncodeCertPEM(serverCACert),
			},
		},
		Contexts: map[string]*api.Context{
			contextName: {
				Cluster:  clusterName,
				AuthInfo: userName,
			},
		},
		AuthInfos: map[string]*api.AuthInfo{
			userName: {
				ClientKeyData:         certs.EncodePrivateKeyPEM(clientKey),
				ClientCertificateData: certs.EncodeCertPEM(clientCert),
			},
		},
		CurrentContext: contextName,
	}, nil
}

func getCertSecret(ctx context.Context, c client.Client, clusterName, clusterNamespace, purpose string) (*corev1.Secret, error) {
	secret := &corev1.Secret{}
	secretKey := client.ObjectKey{Namespace: clusterNamespace, Name: clusterName + "-" + purpose}

	if err := c.Get(ctx, secretKey, secret); err != nil {
		return nil, err
	}

	return secret, nil
}
