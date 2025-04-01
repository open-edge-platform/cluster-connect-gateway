// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package kubeutil

import (
	"sync"
	"testing"

	v1alpha1 "github.com/open-edge-platform/cluster-connect-gateway/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestGetCerts(t *testing.T) {
	scheme := runtime.NewScheme()
	clientgoscheme.AddToScheme(scheme)
	v1alpha1.AddToScheme(scheme)

	// Create a fake client with a ClusterConnect object and secrets
	cc := &v1alpha1.ClusterConnect{
		ObjectMeta: ctrl.ObjectMeta{
			Name: "test-tunnel",
		},
		Spec: v1alpha1.ClusterConnectSpec{
			ClientCertRef: &corev1.ObjectReference{
				Name:      "client-cert",
				Namespace: "default",
			},
			ServerCertRef: &corev1.ObjectReference{
				Name:      "server-cert",
				Namespace: "default",
			},
		},
	}
	clientCertSecret := &corev1.Secret{
		ObjectMeta: ctrl.ObjectMeta{
			Name:      "client-cert",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"tls.crt": []byte(`
-----BEGIN CERTIFICATE-----
MIIC6jCCAdKgAwIBAgIBADANBgkqhkiG9w0BAQsFADAVMRMwEQYDVQQDEwprdWJl
cm5ldGVzMB4XDTI1MDIxODAyMTczOVoXDTM1MDIxNjAyMjIzOVowFTETMBEGA1UE
AxMKa3ViZXJuZXRlczCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBALDO
aOWKcc3dEoGN5zYxD/8OV8d7BCiwMtkV5fWLOZenTveiiqzMfGfWe7WVSGYaohie
28yQL3T8oJ6GrU1So40AG38oLkwji6WqV9UftSVfMWN9nYQsUk245k0F8Ii1lgcY
x9NWl7EFoqy5ppUtbMruWtHsGbC52weT0OvA0BhXCAEzEs8ZQOLcM9ovi29xQLZi
YIUcyVKFCSxDrC+FlSBCFLq5FqwuWmH2iAL1jjr1OSEi9H1lE+V0QpATDiVh987r
TA2g0paY8Vut38wywFwv/KxjL+RGbRgieET/e83q3qhO7A9Xht3VBgYc3FGJH6wL
lul7/ynVkGGyOjKzi6cCAwEAAaNFMEMwDgYDVR0PAQH/BAQDAgKkMBIGA1UdEwEB
/wQIMAYBAf8CAQAwHQYDVR0OBBYEFPJ1wChRjATKQVP6whgJiMucGeUAMA0GCSqG
SIb3DQEBCwUAA4IBAQAKXzExUriJ1xdq62ZtTxjUXz9Zcj86/a4yPYloZw17cn8a
Imm/BZHaok4XyUIF5jhk0diaBt6wp0Pm7KBVvceIs1ChIHevPOaCc6s3PIGF98/0
FZ3vPnx/dtH4uM8I2L+enFaK3pNkSM6cbJIV6pXAHIS8qo61k1FjQwaSyTURgCJw
89ip1aqitCeuoEjraZYaxV4d4+tZgoY8qZOWOHs8PZAfg4x3ZPXSQM6tnoF1cLzS
5vWXi+gKXSXA1QHRYSnMldjAzMjnsJJe/haeKOlYohQcBuDXOZUacOwCbz7/RuNg
KY8DZC3hx+gPswLs+uSsG2g9FIYikrM9AbClJRSS
-----END CERTIFICATE-----
            `),
			"tls.key": []byte(`
-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEAsM5o5Ypxzd0SgY3nNjEP/w5Xx3sEKLAy2RXl9Ys5l6dO96KK
rMx8Z9Z7tZVIZhqiGJ7bzJAvdPygnoatTVKjjQAbfyguTCOLpapX1R+1JV8xY32d
hCxSTbjmTQXwiLWWBxjH01aXsQWirLmmlS1syu5a0ewZsLnbB5PQ68DQGFcIATMS
zxlA4twz2i+Lb3FAtmJghRzJUoUJLEOsL4WVIEIUurkWrC5aYfaIAvWOOvU5ISL0
fWUT5XRCkBMOJWH3zutMDaDSlpjxW63fzDLAXC/8rGMv5EZtGCJ4RP97zereqE7s
D1eG3dUGBhzcUYkfrAuW6Xv/KdWQYbI6MrOLpwIDAQABAoIBADL4Pjs2DxrGyYf5
rZbsy+y+aMXEC+3i5bW5X2LK8R5sCBLRk+K+zHpu1ZkCYS22LdalLT4qrtOt5Gvu
7VTmJuoGBudAYSBn+uEWW13AV0tfxvAkjV1GHa/0Rsgblz0CBC8lkK23P+GzezMK
DiDhSISz9BCiXMGawq7LiSX9nr/1E7mOsdH26BJvO5T3P40fQhR6/hqQ8ZQl2MA5
fPQXj9fM13dLnSX5gi8BiqxStdZWwToqzgmN404doc/G4hSQhzOUCR2ff7kmNK1w
FiohZy2UvSAAkHXbTqFpxo/rxtQSimv0IpcSmKWImJiZuXc+9fQ0s+joLBwaYFXp
tPQuf/kCgYEAwEXk3tfy2Q5m7/DvUGeoZH1QbpoYQaPH7JFo/BpA8oXsNCKkUfKq
4NkOHn5cqbEkqn/o824gDNACj5ZLxUdL2jATGPkgv43P9I45RB/q1z6xmUr7l8dc
Buz78Zt7K99OTexgOLVk4NsSl93J00fYQE47iuCIEGhssERekoXVl6MCgYEA62gv
JCu20ibuyAjPpnaZWeqMAkrbLGiZI0wxfJXSzXTjNrPRHvIypZsfbfNc/G0ihuTQ
QdnQ4QkL934SN9DJHUDRY1YS7Kpepkbyq0eVTdGn0IQAg1y5xm52Gv+AIPFX+28A
Zy8dIaqPLra6/W0InpvZZRq3PfOUJe8KGAiGzC0CgYAq0mY6y1WmyfJbFgn2ml+C
ofY7683jMJriMTB0lVRJr1H/+ocmSSmNkkn2uKXilTVZU8uKC8jPkbRATnTppwtZ
uMNIGJQWlXrvOI3AgmtHLQtY3L5T+26fjEBAeyRfjQhfinmTp7Kj8aaedCLzD1k2
WTYhpAgpv1gVmeSGNZBwiQKBgQCAldoaMd6dEDMiBN4YGXROjzWHEwiBS2lKxJXL
bbNGEvEBslsqQjW0C/WxA1vpbluLv3SaY7YbFev5dl3RKzSPzBYT4rJXoAAvZ1Wq
hWFiroCx/0igeIfpgfD1cla0p9/dMZbQxgVtnFK1u46MW4B30r1+4obxShnEVrv2
wMGQyQKBgCzPCG+yErwqtUMmod0/IFVn9y6CTecOY06obkhNz52YZf5iuDFVN1u9
NrVCdA1dvnlbD/XDcwLk/vK1NYnerlny7dIKtfltGqjX38kh7XKLOFickBO7Qo8f
pqff57pweBjeLyDZRtfZyMJoLzH6/KXImRBc1BU31AlEzQg+4Vkq
-----END RSA PRIVATE KEY-----
			`),
		},
	}
	serverCertSecret := &corev1.Secret{
		ObjectMeta: ctrl.ObjectMeta{
			Name:      "server-cert",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"tls.crt": []byte(`
-----BEGIN CERTIFICATE-----
MIIC6jCCAdKgAwIBAgIBADANBgkqhkiG9w0BAQsFADAVMRMwEQYDVQQDEwprdWJl
cm5ldGVzMB4XDTI1MDIxODAyMTczOVoXDTM1MDIxNjAyMjIzOVowFTETMBEGA1UE
AxMKa3ViZXJuZXRlczCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBALDO
aOWKcc3dEoGN5zYxD/8OV8d7BCiwMtkV5fWLOZenTveiiqzMfGfWe7WVSGYaohie
28yQL3T8oJ6GrU1So40AG38oLkwji6WqV9UftSVfMWN9nYQsUk245k0F8Ii1lgcY
x9NWl7EFoqy5ppUtbMruWtHsGbC52weT0OvA0BhXCAEzEs8ZQOLcM9ovi29xQLZi
YIUcyVKFCSxDrC+FlSBCFLq5FqwuWmH2iAL1jjr1OSEi9H1lE+V0QpATDiVh987r
TA2g0paY8Vut38wywFwv/KxjL+RGbRgieET/e83q3qhO7A9Xht3VBgYc3FGJH6wL
lul7/ynVkGGyOjKzi6cCAwEAAaNFMEMwDgYDVR0PAQH/BAQDAgKkMBIGA1UdEwEB
/wQIMAYBAf8CAQAwHQYDVR0OBBYEFPJ1wChRjATKQVP6whgJiMucGeUAMA0GCSqG
SIb3DQEBCwUAA4IBAQAKXzExUriJ1xdq62ZtTxjUXz9Zcj86/a4yPYloZw17cn8a
Imm/BZHaok4XyUIF5jhk0diaBt6wp0Pm7KBVvceIs1ChIHevPOaCc6s3PIGF98/0
FZ3vPnx/dtH4uM8I2L+enFaK3pNkSM6cbJIV6pXAHIS8qo61k1FjQwaSyTURgCJw
89ip1aqitCeuoEjraZYaxV4d4+tZgoY8qZOWOHs8PZAfg4x3ZPXSQM6tnoF1cLzS
5vWXi+gKXSXA1QHRYSnMldjAzMjnsJJe/haeKOlYohQcBuDXOZUacOwCbz7/RuNg
KY8DZC3hx+gPswLs+uSsG2g9FIYikrM9AbClJRSS
-----END CERTIFICATE-----
			`),
			"tls.key": []byte(`
-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEAsM5o5Ypxzd0SgY3nNjEP/w5Xx3sEKLAy2RXl9Ys5l6dO96KK
rMx8Z9Z7tZVIZhqiGJ7bzJAvdPygnoatTVKjjQAbfyguTCOLpapX1R+1JV8xY32d
hCxSTbjmTQXwiLWWBxjH01aXsQWirLmmlS1syu5a0ewZsLnbB5PQ68DQGFcIATMS
zxlA4twz2i+Lb3FAtmJghRzJUoUJLEOsL4WVIEIUurkWrC5aYfaIAvWOOvU5ISL0
fWUT5XRCkBMOJWH3zutMDaDSlpjxW63fzDLAXC/8rGMv5EZtGCJ4RP97zereqE7s
D1eG3dUGBhzcUYkfrAuW6Xv/KdWQYbI6MrOLpwIDAQABAoIBADL4Pjs2DxrGyYf5
rZbsy+y+aMXEC+3i5bW5X2LK8R5sCBLRk+K+zHpu1ZkCYS22LdalLT4qrtOt5Gvu
7VTmJuoGBudAYSBn+uEWW13AV0tfxvAkjV1GHa/0Rsgblz0CBC8lkK23P+GzezMK
DiDhSISz9BCiXMGawq7LiSX9nr/1E7mOsdH26BJvO5T3P40fQhR6/hqQ8ZQl2MA5
fPQXj9fM13dLnSX5gi8BiqxStdZWwToqzgmN404doc/G4hSQhzOUCR2ff7kmNK1w
FiohZy2UvSAAkHXbTqFpxo/rxtQSimv0IpcSmKWImJiZuXc+9fQ0s+joLBwaYFXp
tPQuf/kCgYEAwEXk3tfy2Q5m7/DvUGeoZH1QbpoYQaPH7JFo/BpA8oXsNCKkUfKq
4NkOHn5cqbEkqn/o824gDNACj5ZLxUdL2jATGPkgv43P9I45RB/q1z6xmUr7l8dc
Buz78Zt7K99OTexgOLVk4NsSl93J00fYQE47iuCIEGhssERekoXVl6MCgYEA62gv
JCu20ibuyAjPpnaZWeqMAkrbLGiZI0wxfJXSzXTjNrPRHvIypZsfbfNc/G0ihuTQ
QdnQ4QkL934SN9DJHUDRY1YS7Kpepkbyq0eVTdGn0IQAg1y5xm52Gv+AIPFX+28A
Zy8dIaqPLra6/W0InpvZZRq3PfOUJe8KGAiGzC0CgYAq0mY6y1WmyfJbFgn2ml+C
ofY7683jMJriMTB0lVRJr1H/+ocmSSmNkkn2uKXilTVZU8uKC8jPkbRATnTppwtZ
uMNIGJQWlXrvOI3AgmtHLQtY3L5T+26fjEBAeyRfjQhfinmTp7Kj8aaedCLzD1k2
WTYhpAgpv1gVmeSGNZBwiQKBgQCAldoaMd6dEDMiBN4YGXROjzWHEwiBS2lKxJXL
bbNGEvEBslsqQjW0C/WxA1vpbluLv3SaY7YbFev5dl3RKzSPzBYT4rJXoAAvZ1Wq
hWFiroCx/0igeIfpgfD1cla0p9/dMZbQxgVtnFK1u46MW4B30r1+4obxShnEVrv2
wMGQyQKBgCzPCG+yErwqtUMmod0/IFVn9y6CTecOY06obkhNz52YZf5iuDFVN1u9
NrVCdA1dvnlbD/XDcwLk/vK1NYnerlny7dIKtfltGqjX38kh7XKLOFickBO7Qo8f
pqff57pweBjeLyDZRtfZyMJoLzH6/KXImRBc1BU31AlEzQg+4Vkq
-----END RSA PRIVATE KEY-----
			`),
		},
	}

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cc, clientCertSecret, serverCertSecret).Build()

	kc := &kubeclient{
		certStore: sync.Map{},
		kcStore:   sync.Map{},
		client:    fakeClient,
	}

	caPool, clientCert, err := kc.GetCerts("test-tunnel")
	assert.NoError(t, err)
	assert.NotNil(t, caPool)
	assert.NotNil(t, clientCert)
}

func TestInvalidateCerts(t *testing.T) {
	scheme := runtime.NewScheme()
	clientgoscheme.AddToScheme(scheme)
	v1alpha1.AddToScheme(scheme)

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	kc := &kubeclient{
		certStore: sync.Map{},
		client:    fakeClient,
	}

	// Store a dummy cert in the cert store
	kc.certStore.Store("test-tunnel", &Certs{})

	err := kc.InvalidateCerts("test-tunnel")
	assert.NoError(t, err)

	_, ok := kc.certStore.Load("test-tunnel")
	assert.False(t, ok)
}
