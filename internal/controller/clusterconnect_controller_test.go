// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"time"

	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"

	v1alpha1 "github.com/open-edge-platform/cluster-connect-gateway/api/v1alpha1"
	"github.com/open-edge-platform/cluster-connect-gateway/internal/utils/certutil"
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
					cc.Status.ConnectionProbe.LastProbeSuccessTimestamp == metav1.Time{}
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
					ControlPlaneRef: clusterv1.ContractVersionedObjectReference{
						APIGroup: "controlplane.cluster.x-k8s.io",
						Kind:     "KThreesControlPlane",
						Name:     testName,
					},
					Topology: clusterv1.Topology{
						ClassRef: clusterv1.ClusterClassRef{Name: "baseline-v0.0.1"},
						Version:  "v1.30.6+rke2r1",
						Variables: []clusterv1.ClusterVariable{
							{
								Name: "airGapped",
								Value: v1.JSON{
									Raw: []byte("true"),
								},
							},
						},
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
			Expect(k8sClient.Create(ctx, kc)).To(Succeed())

			caCert, caKey, err := certutil.GenerateTestCertificate()
			Expect(err).NotTo(HaveOccurred())

			ca = &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testName + "-ca",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"tls.crt": caCert,
					"tls.key": caKey,
				},
			}
			Expect(k8sClient.Create(ctx, ca)).To(Succeed())

			caCert1, caKey1, err := certutil.GenerateTestCertificate()
			Expect(err).NotTo(HaveOccurred())

			cca = &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testName + "-cca",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"tls.crt": caCert1,
					"tls.key": caKey1,
				},
			}
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
				return err == nil && len(cl.Spec.Topology.Variables) == 2 &&
					cl.Spec.Topology.Variables[0].Name == "airGapped" &&
					cl.Spec.Topology.Variables[1].Name == "connectAgentManifest"
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
