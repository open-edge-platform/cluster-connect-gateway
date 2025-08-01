// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/open-edge-platform/cluster-connect-gateway/test/utils"
)

var (
	// Optional Environment Variables:
	skipKindCleanup        = os.Getenv("SKIP_KIND_CLEANUP") == "true"
	skipCertManagerInstall = os.Getenv("CERT_MANAGER_INSTALL_SKIP") == "true"
	skipClusterAPIInstall  = os.Getenv("CLUSTER_API_INSTALL_SKIP") == "true"
	skipDockerBuild        = os.Getenv("SKIP_DOCKER_BUILD") == "true"

	isCertManagerAlreadyInstalled        = false
	isClusterAPIOperatorAlreadyInstalled = false
	isClusterAPIProviderAlreadyInstalled = false

	namespace = os.Getenv("NAMESPACE")
)

const (
	// timeout and interval for Gomega Eventually
	timeout  = time.Second * 120
	interval = time.Second * 1

	// namespace where the project is deployed in

	testDataPath = "test/resources/testdata/"
)

// TestE2E runs the end-to-end (e2e) test suite for the project. These tests execute in an isolated,
// temporary environment to validate project changes with the the purposed to be used in CI jobs.
// The default setup requires Kind, builds/loads the Manager Docker image locally, and installs
// CertManager and Cluster API operator and providers.
func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	_, _ = fmt.Fprintf(GinkgoWriter, "Starting cluster-connect-gateway integration test suite\n")
	RunSpecs(t, "e2e suite")
}

var _ = BeforeSuite(func() {
	if !skipDockerBuild {
		By("building the manager image")
		cmd := exec.Command("make", "docker-build")
		_, err := utils.Run(cmd)
		ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Failed to build the images")
	} else {
		_, _ = fmt.Fprintf(GinkgoWriter, "Skipping Docker build step...\n")
	}

	By("loading the manager image on Kind")
	cmd := exec.Command("make", "docker-load")
	_, err := utils.Run(cmd)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Failed to load the images into Kind")

	// The tests-e2e are intended to run on a temporary cluster that is created and destroyed for testing.
	// To prevent errors when tests run in environments with CertManager and Cluster API already installed,
	// we check for their presence before execution.
	// Setup CertManager and Cluster API before the suite if not skipped and if not already installed
	if !skipCertManagerInstall {
		By("checking if cert manager is installed already")
		isCertManagerAlreadyInstalled = utils.IsCertManagerCRDsInstalled()
		if !isCertManagerAlreadyInstalled {
			_, _ = fmt.Fprintf(GinkgoWriter, "Installing CertManager...\n")
			Expect(utils.InstallCertManager()).To(Succeed(), "Failed to install CertManager")
		} else {
			_, _ = fmt.Fprintf(GinkgoWriter, "WARNING: CertManager is already installed. Skipping installation...\n")
		}
	}

	// Setup Cluster API before the suite if not skipped and if not already installed
	if !skipClusterAPIInstall {
		By("checking if cluster api operator is installed already")
		isClusterAPIOperatorAlreadyInstalled = utils.IsClusterAPIOperatorCRDsInstalled()
		if !isClusterAPIOperatorAlreadyInstalled {
			_, _ = fmt.Fprintf(GinkgoWriter, "Installing Cluster API Operator...\n")
			Eventually(utils.InstallClusterAPIOperator(), timeout, interval).Should(Succeed(), "Failed to install Cluster API Operator")
		} else {
			_, _ = fmt.Fprintf(GinkgoWriter, "WARNING: Cluster API Operator is already installed. Skipping installation...\n")
		}

		By("checking if cluster api provider is installed already")
		isClusterAPIProviderAlreadyInstalled = utils.IsClusterAPIProviderCRDsInstalled()
		if !isClusterAPIProviderAlreadyInstalled {
			_, _ = fmt.Fprintf(GinkgoWriter, "Installing Cluster API Provider...\n")
			Eventually(utils.InstallClusterAPIProvider(), timeout, interval).Should(Succeed(), "Failed to install Cluster API provider")
		} else {
			_, _ = fmt.Fprintf(GinkgoWriter, "WARNING: Cluster API Provider is already installed. Skipping installation...\n")
		}
	}

	By("installing cluster-connect-gateway helm charts")
	Expect(utils.InstallEdgeConnectGateway(namespace)).To(Succeed(), "Failed to install Edge Connect Gateway")

	By("creating namespace for test resources")
	cmd = exec.Command("kubectl", "apply", "-f", testDataPath+"namespace.yaml")
	_, err = utils.Run(cmd)
	Expect(err).NotTo(HaveOccurred(), "Failed to create test namespace")
})

var _ = AfterSuite(func() {
	// Teardown CertManager after the suite if not skipped and if they were not already installed
	if !skipKindCleanup && !skipCertManagerInstall && !isCertManagerAlreadyInstalled {
		_, _ = fmt.Fprintf(GinkgoWriter, "Uninstalling CertManager...\n")
		utils.UninstallCertManager()
	}

	// Teardown Cluster API operator and providers after the suite if not skipped and if they were not already installed
	if !skipKindCleanup && !skipClusterAPIInstall && !isClusterAPIOperatorAlreadyInstalled {
		_, _ = fmt.Fprintf(GinkgoWriter, "Uninstalling Cluster API Operator...\n")
		utils.UninstallClusterAPIOperator()
	}
	if !skipKindCleanup && !skipClusterAPIInstall && !isClusterAPIProviderAlreadyInstalled {
		_, _ = fmt.Fprintf(GinkgoWriter, "Uninstalling Cluster API Provider...\n")
		utils.UninstallClusterAPIProvider()
	}

	// Teardown the helm chart.
	if !skipKindCleanup {
		By("uninstalling cluster-connect-gateway helm charts")
		cmd := exec.Command("make", "helm-uninstall")
		_, _ = utils.Run(cmd)
	}
})
