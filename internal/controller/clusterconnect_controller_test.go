// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"

	v1alpha1 "github.com/open-edge-platform/cluster-connect-gateway/api/v1alpha1"
)

var _ = Describe("ClusterConnect Controller", Ordered, func() {
	const (
		timeout  = time.Second * 5
		interval = time.Second * 1
	)

	var (
		cc  = &v1alpha1.ClusterConnect{}
		cl  = &clusterv1.Cluster{}
		kc  = &corev1.Secret{}
		ca  = &corev1.Secret{}
		cca = &corev1.Secret{}
	)

	Context("When reconciling a ClusterConnect resource without CAPI ClusterRef", func() {
		var (
			testName           = "test1"
			testClusterConnect = types.NamespacedName{Name: testName}
			testAuthSecret     = types.NamespacedName{Name: testName + "-agent-token", Namespace: "default"}
		)

		BeforeEach(func() {
			By("creating the custom resource for the Kind ClusterConnect")
			resource := &v1alpha1.ClusterConnect{
				ObjectMeta: metav1.ObjectMeta{
					Name: testName,
				},
			}
			Expect(k8sClient.Create(ctx, resource)).To(Succeed())
		})

		AfterEach(func() {
			resource := &v1alpha1.ClusterConnect{}
			err := k8sClient.Get(ctx, testClusterConnect, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance ClusterConnect")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})

		It("should successfully reconcile the resource", func() {
			// Ensure auth-token secret is created.
			Eventually(func() bool {
				return k8sClient.Get(ctx, testAuthSecret, &corev1.Secret{}) == nil
			}, timeout, interval).Should(BeTrue())

			// Ensure status.agentManifest is set.
			Eventually(func() bool {
				err := k8sClient.Get(ctx, testClusterConnect, cc)
				return err == nil && cc.Status.AgentManifest != ""
			}, timeout, interval).Should(BeTrue())

			// Ensure status.connectionProbe is set.
			Eventually(func() bool {
				err := k8sClient.Get(ctx, testClusterConnect, cc)
				return err == nil &&
					cc.Status.ConnectionProbe.LastProbeTimestamp == metav1.Time{} &&
					cc.Status.ConnectionProbe.LastProbeSuccessTimestamp == metav1.Time{} &&
					cc.Status.ConnectionProbe.ConsecutiveFailures == 0
			}, timeout, interval).Should(BeTrue())

			// Ensure ControlPlaneEndpoint is set.
			Eventually(func() bool {
				err := k8sClient.Get(ctx, testClusterConnect, cc)
				return err == nil &&
					cc.Status.ControlPlaneEndpoint.Host == "connect-gateway.default.svc" &&
					cc.Status.ControlPlaneEndpoint.Port == 8080
			}, timeout, interval).Should(BeTrue())

			// Ensure there are four conditions and status.ready is true.
			Expect(cc.Status.Conditions).To(HaveLen(4))
			Expect(cc.Status.Ready).To(BeTrue())
		})
	})

	Context("When reconciling a resource with CAPI ClusterRef", func() {
		var (
			testName           = "test2"
			testClusterConnect = types.NamespacedName{Name: testName}
			testAuthSecret     = types.NamespacedName{Name: testName + "-agent-token", Namespace: "default"}
			testCluster        = types.NamespacedName{Name: testName, Namespace: "default"}
			testKubeconfig     = types.NamespacedName{Name: testName + "-kubeconfig", Namespace: "default"}
			testCa             = types.NamespacedName{Name: testName + "-ca", Namespace: "default"}
			testCca            = types.NamespacedName{Name: testName + "-cca", Namespace: "default"}
		)

		BeforeEach(func() {
			By("creating Cluster object and kubeconfig Secrets before ClusterConnect object")
			cl = &clusterv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testName,
					Namespace: "default",
				},
				Spec: clusterv1.ClusterSpec{
					ControlPlaneRef: &corev1.ObjectReference{
						APIVersion: "controlplane.cluster.x-k8s.io/v1beta1",
						Kind:       "RKE2ControlPlane",
						Name:       testName,
						Namespace:  "default",
					},
					Topology: &clusterv1.Topology{
						Class:   "baseline-v0.0.1",
						Version: "v1.30.6+rke2r1",
					},
				},
			}
			Expect(k8sClient.Create(ctx, cl)).To(Succeed())

			kc = &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testName + "-kubeconfig",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"value": []byte("test-data"),
				},
			}
			ca = &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testName + "-ca",
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
			cca = &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testName + "-cca",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"tls.crt": []byte(`
-----BEGIN CERTIFICATE-----
MIIC6jCCAdKgAwIBAgIBADANBgkqhkiG9w0BAQsFADAVMRMwEQYDVQQDEwprdWJl
cm5ldGVzMB4XDTI1MDIxODAyMTczOVoXDTM1MDIxNjAyMjIzOVowFTETMBEGA1UE
AxMKa3ViZXJuZXRlczCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBALje
vxpGa0dHKVeoF9mZDp43Xfnqae4Cd+T7fZngC12CeSW64OTl9xiTQ8/tsQwTf32c
SB6GDXSetTn3umKip17MgWYo5EUnIcxDTD2ZG2jmZUIidl6KU8FmW9warwp9e2NA
FNeGlYpBAiWLrekUGpMD76uvD1M/k5V3y94XZjXB7biWGDVB+WOHhirXRSQ5sroZ
obcuHjVbPXDMJyIl5xf6G1U5FEru8ucXMLRH/W80WWT0Y3twkhcHlqggumckbFEe
faBhkZx2+I2WSRVRKoBcYQfMfthNgwwvWZaP9t/ZIAWyycd0Aujl3fsTHqQG9oOk
kuUoU/ZxYv7iNEdJEzECAwEAAaNFMEMwDgYDVR0PAQH/BAQDAgKkMBIGA1UdEwEB
/wQIMAYBAf8CAQAwHQYDVR0OBBYEFDux1+2JrwueABqvqxK0d2yR47KjMA0GCSqG
SIb3DQEBCwUAA4IBAQA67lG7TV93cEud/z4Ny245qAQGFAmObIhFu2g5dHz9QV+6
+hStcXTCUvgAvUnFca1uK3ZcfvIEcGqXVt3Na54v0hyJw1SGLRXdegN9ebksZFRX
evzHKPswjOGiOOhpkJNYs/VY2/RkXLcrWfzBAyQZL9IpqBDC5LXo18nN/Up2oZ+Q
u2mxxsbn5lb+8Wq3ZrOA2kR5LIxCiJ8bRf4I01/B5OfaGmY4C1y/G4ZusLkMw5Um
WT+bXLGKr1Nd3HBtanJModAhndTgGPqV7fYuidwsJzIgtbFuXLtE58wc0/w9Is/J
KIZD2cJgNml9oeaRlWnKbkWSjPgmpjcoWJvGFF4H
-----END CERTIFICATE-----	
					`),
					"tls.key": []byte(`
-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEAuN6/GkZrR0cpV6gX2ZkOnjdd+epp7gJ35Pt9meALXYJ5Jbrg
5OX3GJNDz+2xDBN/fZxIHoYNdJ61Ofe6YqKnXsyBZijkRSchzENMPZkbaOZlQiJ2
XopTwWZb3BqvCn17Y0AU14aVikECJYut6RQakwPvq68PUz+TlXfL3hdmNcHtuJYY
NUH5Y4eGKtdFJDmyuhmhty4eNVs9cMwnIiXnF/obVTkUSu7y5xcwtEf9bzRZZPRj
e3CSFweWqCC6ZyRsUR59oGGRnHb4jZZJFVEqgFxhB8x+2E2DDC9Zlo/239kgBbLJ
x3QC6OXd+xMepAb2g6SS5ShT9nFi/uI0R0kTMQIDAQABAoIBACmlpI+axZgtI9cn
exf+LxjdiEJA2NRRogjnzwfXPTg3qGsjin6Gf//9TQ2OkpAR5Gq7xlN6B5X6GwGQ
GWEXG3VzxMLq+2HIywhlMMu/rjkXjkAz2065At1S3ebb7AsE8cMOMKPNIesfcghb
SzZy3Pm5rUiUC3PJosnNTxQdWV1KjDwbFn180VqYiTwHGDTbXjvEnTg8GgkEdjkv
DMB0CXYBr6gGQcRmFSMYepl4gdgi82Td9LF6L0SDZkiAc7SHY1mQTUZNEcYPJ1ZT
9jeWj+l5y5AcW0Ef0ZM3Ig+9NjgrzPj3PymgSmCgnvtEj60+ijeqwhgen7wHEoA+
YRPfo0ECgYEAwVyMnObmMmEciNEMCxSLlBuJWFlVY2jl0MME+XhPZnYBjkjxCcpx
hjjJTpUft/hdkNbGO0Acq6hzvaPJSJOjcuHshXSiBYD5tSubwqpG5Waml40CWQL/
B6eTZ+4SSNHlDvmKBBpxaAHE87vPNGKZfdmdSIw2gRXpRhrot6+1CvUCgYEA9MIB
QJ6MWRZcOCgO/EOi+ACEVwPvlBPaoYBB1sQ/5K5DFPW0+KGBxHexCSLrCkpvqAea
QlH1NrOcm5FMe4Tv4TZCJmMxo8YGOImGV1T4heFK43SwAsglB038d9DerXnFF2pG
LX1NKi4rHg81TIEhDFu03NnMlMF65BkoJEKr+c0CgYB911m8E5nlV8ZWVJbd7zUX
4wPz98/WrsJCp2lpYp9qM4q3q0kT8ikIDZUWC39lH8LlLKI+lW8TWMJavWmWTl+l
uNtRz6JSqqnY+tSYUHFrgu8Q2riYbZbEyI7qqMpFkjTdFhJ4sSz1X5FqfuRPV4xF
l4cBXB9sy3HSKlkikrvksQKBgANZmGzK/r7FaJO4MKRRRKzCTQ1n12KedvKY783B
t+IoZHCoAEca5szKmpi2aqMka6zQFdNsjb+m67DRZ0uVgTcC6BrgZrCIKFOjxjvx
xMkIUV8VnMcc+CzGa3s0V6fmL8VnRfifa2v1X8+0g5U9ezdRFgOj04IBt+3hTkRY
yUPNAoGBALHJ5Bjsiw/NSnxSC9RevCMPB0BqtVcgk74Pgnk6VaHrGPP6+09UkfRC
kmV/gLc0YHh9vpvPJyx53f8YNzJLjLmsQf4SO9Oz4TiY8i9pQm1IPm7Ide4u8H2R
alqk7z9FdSWcdNlg08Oo/j8W/yQrvpsipgAwcxA3XnVSC6CKtysc
-----END RSA PRIVATE KEY-----
					`),
				},
			}
			Expect(k8sClient.Create(ctx, kc)).To(Succeed())
			Expect(k8sClient.Create(ctx, ca)).To(Succeed())
			Expect(k8sClient.Create(ctx, cca)).To(Succeed())

			cc = &v1alpha1.ClusterConnect{
				ObjectMeta: metav1.ObjectMeta{
					Name: testName,
				},
				Spec: v1alpha1.ClusterConnectSpec{
					ClusterRef: &corev1.ObjectReference{
						Name:       testName,
						Namespace:  "default",
						APIVersion: "cluser.cluster.x-k8s.io/v1beta1",
						Kind:       "Cluster",
					},
				},
			}
			Expect(k8sClient.Create(ctx, cc)).To(Succeed())
		})

		AfterEach(func() {
			err := k8sClient.Get(ctx, testClusterConnect, cc)
			Expect(err).NotTo(HaveOccurred())

			err = k8sClient.Get(ctx, testCluster, cl)
			Expect(err).NotTo(HaveOccurred())

			err = k8sClient.Get(ctx, testKubeconfig, kc)
			Expect(err).NotTo(HaveOccurred())

			err = k8sClient.Get(ctx, testCa, ca)
			Expect(err).NotTo(HaveOccurred())

			err = k8sClient.Get(ctx, testCca, cca)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance ClusterConnect and ControlPlane")
			Expect(k8sClient.Delete(ctx, cc)).To(Succeed())
			Expect(k8sClient.Delete(ctx, cl)).To(Succeed())
			Expect(k8sClient.Delete(ctx, kc)).To(Succeed())
			Expect(k8sClient.Delete(ctx, ca)).To(Succeed())
			Expect(k8sClient.Delete(ctx, cca)).To(Succeed())
		})

		It("should successfully reconcile the resource", func() {
			// Ensure auth-token secret is created.
			tokenSecret := &corev1.Secret{}
			Eventually(func() bool {
				return k8sClient.Get(ctx, testAuthSecret, tokenSecret) == nil
			}, timeout, interval).Should(BeTrue())

			// Ensure status.agentManifest is set.
			Eventually(func() bool {
				err := k8sClient.Get(ctx, testClusterConnect, cc)
				return err == nil && cc.Status.AgentManifest != ""
			}, timeout, interval).Should(BeTrue())

			// Ensure ControlPlaneEndpoint is set.
			Eventually(func() bool {
				err := k8sClient.Get(ctx, testClusterConnect, cc)
				return err == nil &&
					cc.Status.ControlPlaneEndpoint.Host == "connect-gateway.default.svc" &&
					cc.Status.ControlPlaneEndpoint.Port == 8080
			}, timeout, interval).Should(BeTrue())

			// Ensure Cluster spec is patched with the agent config.
			// TODO: improve the validation here
			Eventually(func() bool {
				err := k8sClient.Get(ctx, testCluster, cl)
				return err == nil && len(cl.Spec.Topology.Variables) == 1 &&
					cl.Spec.Topology.Variables[0].Name == "connectAgentManifest"
			}, timeout, interval).Should(BeTrue())

			// Now set ObservedGeneration to Generation.
			cl.Status.ObservedGeneration = cl.Generation
			Expect(k8sClient.Status().Update(ctx, cl)).To(Succeed())

			// Ensure ClusterConnect status is ready.
			Eventually(func() bool {
				err := k8sClient.Get(ctx, testClusterConnect, cc)
				return err == nil && cc.Status.Ready
			}, timeout, interval).Should(BeTrue())

			// Ensure kubeconfig secret is with expected server url.
			Eventually(func() bool {
				err := k8sClient.Get(ctx, testKubeconfig, kc)
				Expect(err).NotTo(HaveOccurred())
				kubeconfig, err := clientcmd.Load(kc.Data["value"])
				return err == nil &&
					kubeconfig.Clusters[testName].Server == "http://connect-gateway.default.svc:8080/kubernetes/test2"
			}, timeout, interval).Should(BeTrue())
		})
	})
})
