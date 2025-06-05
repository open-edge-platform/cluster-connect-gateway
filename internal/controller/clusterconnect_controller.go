// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/tools/record"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/controllers/external"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	cutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	v1alpha1 "github.com/open-edge-platform/cluster-connect-gateway/api/v1alpha1"
	"github.com/open-edge-platform/cluster-connect-gateway/internal/agentconfig"
	"github.com/open-edge-platform/cluster-connect-gateway/internal/auth"
	"github.com/open-edge-platform/cluster-connect-gateway/internal/provider"
	"github.com/open-edge-platform/cluster-connect-gateway/internal/utils/kubeutil"
)

const (
	FinalizerConnectController = "cluster.edge-orchestrator.intel.com/connect-controller"

	clusterRefKey = ".spec.clusterRef"

	agentManifestPath   = "connect-agent.yaml"
	privateCAEnabledEnv = "PRIVATE_CA_ENABLED"
)

var (
	createOnlyPredicate = predicate.Funcs{
		CreateFunc: func(_ event.CreateEvent) bool {
			return true
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			// no action
			return false
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			// no action
			return false
		},
		GenericFunc: func(_ event.GenericEvent) bool {
			// no action
			return false
		},
	}

	clusterPredicate = predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			return true
		},
		CreateFunc: func(e event.CreateEvent) bool {
			return false
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return false
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return false
		},
	}
)

var (
	clusterConnectConnectionProbeTimeout = 5 * time.Minute
)

type ConnectAgentConfig struct {
	Path    string `json:"path"`
	Owner   string `json:"owner"`
	Content string `json:"content"`
}

// +kubebuilder:rbac:groups=cluster.edge-orchestrator.intel.com,resources=clusterconnects,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cluster.edge-orchestrator.intel.com,resources=clusterconnects/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cluster.edge-orchestrator.intel.com,resources=clusterconnects/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;patch;update
// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=clusters;clusters/status,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=apiextensions.k8s.io,resources=customresourcedefinitions,verbs=get;list;watch

// ClusterConnectReconciler reconciles a ClusterConnect object
type ClusterConnectReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	// TODO: Add pointer to connect-gateway client to request disconnect when
	// ClusterConnect object is removed
	// gatewayClient *gateway.Client
	tokenManager    auth.TokenManager
	providerManager provider.ProviderManager

	controlPlaneEndpointHost string
	controlPlaneEndpointPort int32

	externalTracker external.ObjectTracker
	recorder        record.EventRecorder
}

func clusterRefIdxFunc(o client.Object) []string {
	ref := o.(*v1alpha1.ClusterConnect).Spec.ClusterRef
	if ref != nil {
		return []string{ref.Namespace + ref.Name}
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterConnectReconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager, connectionTimeout time.Duration) error {
	if r.Client == nil {
		return errors.New("Client must not be nil")
	}

	clusterConnectConnectionProbeTimeout = connectionTimeout

	c, err := ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.ClusterConnect{}).
		Named("cluster/clusterconnect").
		Build(r)

	if err != nil {
		return errors.Wrap(err, "failed setting up with a controller manager")
	}

	// TODO: initialize connect-gateway client

	if r.tokenManager, err = auth.NewTokenManager(); err != nil {
		return errors.Wrap(err, "failed to initialize token manager")
	}

	// Initialize provider manager with RKE2ControlPlane provider.
	// Add KubeadmControlPlane when implemented.
	r.providerManager = provider.NewProviderManager().
		WithProvider("RKE2ControlPlane", "/var/lib/rancher/rke2/agent/pod-manifests/connect-agent.yaml").
		WithProvider("KThreesControlPlane", "/var/lib/rancher/k3s/agent/pod-manifests/connect-agent.yaml").
		Build()

	// Get the hostname of the control plane endpoint from environment variable.
	parsedURL, err := url.Parse(os.Getenv("GATEWAY_INTERNAL_URL"))
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("invalid GATEWAY_INTERNAL_URL: %s", os.Getenv("GATEWAY_INTERNAL_URL")))
	}
	port, err := strconv.Atoi(parsedURL.Port())
	if err != nil || (port < 0 || port > 65535) {
		return errors.Wrap(err, fmt.Sprintf("invalid GATEWAY_INTERNAL_URL: %s", os.Getenv("GATEWAY_INTERNAL_URL")))
	}

	r.controlPlaneEndpointHost = parsedURL.Hostname()
	r.controlPlaneEndpointPort = int32(port) // nolint: gosec

	// Add field indexer for spec.clusterRef field.
	if err = mgr.GetFieldIndexer().IndexField(ctx, &v1alpha1.ClusterConnect{}, clusterRefKey, clusterRefIdxFunc); err != nil {
		return errors.Wrap(err, "failed to add field indexer for spec.clusterRef")
	}

	// Setup external tracker to watch unstructured ControlPlane object.
	predicateLog := ctrl.LoggerFrom(ctx).WithValues("controller", "cluster/clusterconnect")
	r.externalTracker = external.ObjectTracker{
		Controller:      c,
		Cache:           mgr.GetCache(),
		Scheme:          mgr.GetScheme(),
		PredicateLogger: &predicateLog,
	}
	r.recorder = mgr.GetEventRecorderFor("cluster/clusterconnect")
	return nil
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.4/pkg/reconcile
func (r *ClusterConnectReconciler) Reconcile(ctx context.Context, req ctrl.Request) (retres ctrl.Result, reterr error) {
	_ = log.FromContext(ctx)

	cc := &v1alpha1.ClusterConnect{}
	if err := r.Client.Get(ctx, req.NamespacedName, cc); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}

	// Add finalizer first to avoid the race condition between init and delete.
	if !cutil.ContainsFinalizer(cc, FinalizerConnectController) {
		cutil.AddFinalizer(cc, FinalizerConnectController)
		if err := r.Update(ctx, cc); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to add finalizer (%s)", err)
		}
	}

	patchHelper, err := patch.NewHelper(cc, r.Client)
	if err != nil {
		return ctrl.Result{}, err
	}

	defer func() {
		// Always reconcile the status.
		r.updateStatus(ctx, cc)

		// Patch the updates after each reconciliation.
		patchOpts := []patch.Option{patch.WithStatusObservedGeneration{}}
		if err := patchHelper.Patch(ctx, cc, patchOpts...); err != nil {
			reterr = kerrors.NewAggregate([]error{reterr, err})
		}

		if reterr != nil {
			retres = ctrl.Result{}
		}
	}()

	// Handle finalizers if the deletion timestamp is non-zero.
	if !cc.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.delete(ctx, cc)
	}

	// Handle normal reconciliation loop.
	result, err := r.reconcile(ctx, cc)
	if err != nil {
		r.recorder.Eventf(cc, corev1.EventTypeWarning, "ReconcileError", "%v", err)
	}

	return result, err
}

func (r *ClusterConnectReconciler) updateStatus(ctx context.Context, cc *v1alpha1.ClusterConnect) {
	_ = log.FromContext(ctx)

	// Check if conditions are initialized, if not, initialize them with Unknown.
	if len(cc.Status.Conditions) == 0 {
		initConditions(cc)
		cc.Status.ConnectionProbe = v1alpha1.ConnectionProbeState{
			LastProbeTimestamp:        metav1.Time{},
			LastProbeSuccessTimestamp: metav1.Time{},
		}
	}

	// Set status.ready to true if all the conditions are true.
	status := true
	for _, condition := range cc.Status.Conditions {
		// skip the condition that is not a part of the provisioning.
		// Status.Ready value should be based only on the conditions
		// that are part of the provisioning.
		if condition.Type == v1alpha1.ConnectionProbeCondition {
			continue
		}

		if condition.Status != metav1.ConditionTrue {
			status = false
			break
		}
	}

	cc.Status.Ready = status
}

//nolint:unparam
func (r *ClusterConnectReconciler) delete(ctx context.Context, cc *v1alpha1.ClusterConnect) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	if !cutil.ContainsFinalizer(cc, FinalizerConnectController) {
		return ctrl.Result{}, nil
	}

	// TODO: Add finalizer logic
	// 1) get connection from remotedialer server
	// 2) close the connection if exists
	cutil.RemoveFinalizer(cc, FinalizerConnectController)

	return ctrl.Result{}, nil
}

//nolint:unparam
func (r *ClusterConnectReconciler) reconcile(ctx context.Context, cc *v1alpha1.ClusterConnect) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// Normal reconcile logic consists of three phases, each dependent on the previous phase.
	// Setp 4 to 6 is valid only when ClusterRef is set in the ClusterConnect object.
	// 1) Ensure the auth token
	// 2) Generate the connect-agent pod manifest
	// 3) Initialize the connection probe state
	// 4) Set control plane endpoint
	// 5) Set the connect-agent config to Cluster object
	// 6) Wait until the Cluster object update is reconciled by Topology controller
	// 7) Update kubeconfig secret
	phases := []func(context.Context, *v1alpha1.ClusterConnect) error{
		r.reconcileAuthToken,
		r.reconcileConnectAgentManifest,
		r.reconcileConnectionProbe,
		r.reconcileControlPlaneEndpoint,
		r.reconcileClusterSpec,
		r.reconcileTopology,
		r.reconcileKubeconfig,
	}

	errs := []error{}
	for _, phase := range phases {
		if err := phase(ctx, cc); err != nil {
			errs = append(errs, err)
			break
		}
	}

	return ctrl.Result{}, kerrors.NewAggregate(errs)
}

func (r *ClusterConnectReconciler) reconcileAuthToken(ctx context.Context, cc *v1alpha1.ClusterConnect) error {
	// TODO: Return early if JWT auth is enabled.

	// Attempt to retrieve token for the cluster connect.
	tunnelId := cc.GetTunnelID()
	exist, err := r.tokenManager.TokenExist(ctx, tunnelId)

	// Return with error for another try, if it failed to check the existence of the token.
	// Keep the condition unknown.
	if err != nil {
		return fmt.Errorf("failed to get token: %v", err)
	}

	// Return early, if the token already exists.
	if exist {
		setAuthTokenReadyConditionTrue(cc)
		return nil
	}

	// Token doesn't exist. Create a new one.
	if err := r.tokenManager.CreateAndStoreToken(ctx, tunnelId, cc); err != nil {
		msg := "failed to create token"
		setAuthTokenReadyConditionFalse(cc, msg)
		return fmt.Errorf("%s: %v", msg, err)
	} else {
		setAuthTokenReadyConditionTrue(cc)
		return nil
	}
}

func (r *ClusterConnectReconciler) reconcileConnectAgentManifest(ctx context.Context, cc *v1alpha1.ClusterConnect) error {
	tunnelId := cc.GetTunnelID()

	// Retrieve the token for the cluster connect.
	// Keep the condition unknown.
	token, err := r.tokenManager.GetToken(ctx, tunnelId)
	if err != nil || token == nil {
		return fmt.Errorf("failed to retrieve token: %v", err)
	}

	manifest, err := agentconfig.GenerateAgentConfig(tunnelId, token.Value)
	if err != nil {
		msg := "failed to generate agent manifest"
		setAgentManifestGeneratedConditionFalse(cc, msg)
		return fmt.Errorf("%s: %v", msg, err)
	}

	cc.Status.AgentManifest = manifest
	setAgentManifestGeneratedConditionTrue(cc)

	return nil
}

func (r *ClusterConnectReconciler) reconcileControlPlaneEndpoint(ctx context.Context, cc *v1alpha1.ClusterConnect) error {
	// Cluster API doesn't allow sub-path in the ControlPlaneEndpoint API URL.
	// Value here is to just pass the contract and won't be used.
	// TODO: Create another field to provide the control plane endpoint that actually works.
	cc.Status.ControlPlaneEndpoint = clusterv1.APIEndpoint{
		Host: r.controlPlaneEndpointHost,
		Port: r.controlPlaneEndpointPort,
	}

	setControlPlaneEndpointSetConditionTrue(cc)
	return nil
}

func (r *ClusterConnectReconciler) reconcileClusterSpec(ctx context.Context, cc *v1alpha1.ClusterConnect) error {
	// Return early, if the ClusterConnect doesn't have associated Cluster-API resources.
	if cc.Spec.ClusterRef == nil {
		return nil
	}

	cluster := &clusterv1.Cluster{}
	err := r.Client.Get(ctx, client.ObjectKey{
		Namespace: cc.Spec.ClusterRef.Namespace,
		Name:      cc.Spec.ClusterRef.Name,
	}, cluster)

	if err != nil {
		setClusterSpecUpdatedConditionFalse(cc)
		return fmt.Errorf("failed to get Cluster object: %v", err)
	}

	// Now update the Cluster object with the agent config.
	agentConfig := &ConnectAgentConfig{
		Path:    r.providerManager.StaticPodManifestPath(cluster.Spec.ControlPlaneRef.Kind),
		Owner:   "root:root",
		Content: cc.Status.AgentManifest,
	}

	agentConfigJson, err := json.Marshal(agentConfig)
	if err != nil {
		setClusterSpecUpdatedConditionFalse(cc)
		return fmt.Errorf("failed to marshal agent config: %v", err)
	}

	// Inject the pod manifest.
	patchHelper, err := patch.NewHelper(cluster, r.Client)
	if err != nil {
		setClusterSpecUpdatedConditionFalse(cc)
		return fmt.Errorf("failed to create patch helper for Cluster: %v", err)
	}

	cluster.Spec.Topology.Variables = []clusterv1.ClusterVariable{
		{
			Name: "connectAgentManifest",
			Value: v1.JSON{
				Raw: agentConfigJson,
			},
		},
	}

	// Patch the updates after each reconciliation.
	patchOpts := []patch.Option{patch.WithStatusObservedGeneration{}}
	if err := patchHelper.Patch(ctx, cluster, patchOpts...); err != nil {
		setClusterSpecUpdatedConditionFalse(cc, "failed to patch Cluster")
		return fmt.Errorf("failed to patch Cluster object: %v", err)
	}

	setClusterSpecReayConditionTrue(cc)
	return nil
}

func (r *ClusterConnectReconciler) reconcileTopology(ctx context.Context, cc *v1alpha1.ClusterConnect) error {
	log := log.FromContext(ctx)

	// Return early, if the ClusterConnect spec doesn't have cluster-api ClusterRef.
	if cc.Spec.ClusterRef == nil {
		return nil
	}

	cluster := &clusterv1.Cluster{}
	err := r.Client.Get(ctx, client.ObjectKey{
		Namespace: cc.Spec.ClusterRef.Namespace,
		Name:      cc.Spec.ClusterRef.Name,
	}, cluster)

	if err != nil {
		setTopologyReconciledConditionFalse(cc)
		return fmt.Errorf("failed to get Cluster object: %v", err)
	}

	// Confirm the Cluster object has connectAgentManifest variable in the topology.
	// This step is redundant as the Cluster object is already updated in reconcileClusterSpec.
	// But, this is to ensure the Cluster object is updated by the Topology controller.
	exists := false
	for _, variable := range cluster.Spec.Topology.Variables {
		if variable.Name == "connectAgentManifest" {
			exists = true
			break
		}
	}

	// If the Generation does not match to the observedGeneration, this means the Cluster spec update in reconcileClusterSpec
	// is not yet reconciled by the Topology controller.
	// Set tracker to watch the Cluster object updates and return.
	if !exists || cluster.Generation != cluster.Status.ObservedGeneration {
		setClusterSpecUpdatedConditionFalse(cc)
		if err := r.externalTracker.Watch(log, cluster, handler.EnqueueRequestsFromMapFunc(r.clusterToClusterConnectMapper),
			clusterPredicate); err != nil {
			return fmt.Errorf("failed to add watch on ClusterRef: %v", err)
		}
		return nil
	}

	setTopologyReconciledConditionTrue(cc)
	return nil
}

// TODO: Improve this function in general to support both CAPI and non-CAPI managed clusters.
func (r *ClusterConnectReconciler) reconcileKubeconfig(ctx context.Context, cc *v1alpha1.ClusterConnect) error {
	log := log.FromContext(ctx)

	// Return early, if the ClusterConnect doesn't have associated Cluster-API resources.
	// TODO: support this feature for non-CAPI managed cluster as well.
	if cc.Spec.ClusterRef == nil {
		return nil
	}

	// Get cluster name and namespace from ClusterRef.
	clusterNamespace := cc.Spec.ClusterRef.Namespace
	clusterName := cc.Spec.ClusterRef.Name

	// Set the labels with kubeconfig Secret name and namespce for use in secretToClusteConnectMapper.
	labels := cc.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}
	labels["cluster.x-k8s.io/kubeconfig-name"] = clusterName + "-kubeconfig"
	labels["cluster.x-k8s.io/kubeconfig-namespace"] = clusterNamespace
	cc.SetLabels(labels)

	// Fetch kubeconfig Secret.
	kc := &corev1.Secret{}
	err := r.Client.Get(ctx, client.ObjectKey{
		Namespace: clusterNamespace,
		Name:      clusterName + "-kubeconfig",
	}, kc)

	// Return early, if kubeconfig Secret object doesn't exist yet.
	// Setup dynamic watch to the kubeconfig Secret object so that the controller can be notificed.
	if apierrors.IsNotFound(err) {
		if err := r.externalTracker.Watch(log, kc, handler.EnqueueRequestsFromMapFunc(r.secretToClusterConnectMapper),
			createOnlyPredicate); err != nil {
			return fmt.Errorf("failed to add watch on kubeconfig secret: %v", err)
		}
		return nil
	}

	if err != nil {
		return fmt.Errorf("failed to fetch kubeconfig Secret: %v", err)
	}

	// Kubeconfig Secret exists, update the server URL.
	patchHelper, err := patch.NewHelper(kc, r.Client)
	if err != nil {
		return fmt.Errorf("failed to create patch helper for ControlPlane: %v", err)
	}

	// Update kubeconfig Secret.
	// Better approach would be create a kubeconfig secret along with required certificates before the ControlPlane creates.
	// But that requires cluster-connect-gateway to manages certificates which is not implemented.
	// So just update the existing kubeconfig Secret now.
	data, err := kubeutil.GenerateKubeconfig(ctx, r.Client, clusterName, clusterNamespace, getControlPlaneEndpointUrl(cc))
	if err != nil || data == nil {
		return fmt.Errorf("failed to generate kubeconfig: %v", err)
	}

	kc.Data = map[string][]byte{
		kubeutil.KubeconfigDataName: data,
	}

	// Enabling private CA will set an orchestration self-signed certificate in the kubeConfig secret
	// which can be used by the downstream cluster fleet-agent to access the Kubernetes API service in the orchestration cluster
	privateCaEnabled := os.Getenv(privateCAEnabledEnv)
	if privateCaEnabled == "true" {
		caCrt, err := kubeutil.GetAPIServerCA(ctx, r.Client)
		if err != nil {
			return fmt.Errorf("failed to get APIServer CA: %v", err)
		}
		kc.Data[kubeutil.ApiServerCA] = caCrt
	}

	// Patch the updates after each reconciliation.
	patchOpts := []patch.Option{patch.WithStatusObservedGeneration{}}
	if err := patchHelper.Patch(ctx, kc, patchOpts...); err != nil {
		return fmt.Errorf("failed to patch ControlPlane object: %v", err)
	}

	setKubeconfigReadyConditionTrue(cc)
	return nil
}

func (r *ClusterConnectReconciler) reconcileConnectionProbe(ctx context.Context, cc *v1alpha1.ClusterConnect) error {
	log.FromContext(ctx)
	// Initialize ConnectionProbe if not already set.
	if cc.Status.ConnectionProbe == (v1alpha1.ConnectionProbeState{}) {
		cc.Status.ConnectionProbe = v1alpha1.ConnectionProbeState{
			LastProbeTimestamp:        metav1.Time{},
			LastProbeSuccessTimestamp: metav1.Time{},
		}
	}

	if cc.Status.ConnectionProbe.LastProbeSuccessTimestamp.IsZero() {
		// initConditions will keep ConnectionProbeCondition Unknown
		return nil
	}

	timeDiff := cc.Status.ConnectionProbe.LastProbeTimestamp.Time.Sub(cc.Status.ConnectionProbe.LastProbeSuccessTimestamp.Time)
	if timeDiff > clusterConnectConnectionProbeTimeout {
		msg := fmt.Sprintf("Remote connection probe failed. Time since last successful probe: %s. Last probe: %s, Last successful probe: %s",
			timeDiff.String(),
			cc.Status.ConnectionProbe.LastProbeTimestamp.Time.Format(time.RFC3339),
			cc.Status.ConnectionProbe.LastProbeSuccessTimestamp.Time.Format(time.RFC3339))
		setConnectionProbeConditionFalse(cc, msg)
	} else {
		setConnectionProbeConditionTrue(cc)
	}
	return nil
}

func (r *ClusterConnectReconciler) clusterToClusterConnectMapper(ctx context.Context, obj client.Object) []ctrl.Request {
	var ccList v1alpha1.ClusterConnectList

	// Find ClusterConnect object that has a given Cluster object in ClusterRef field.
	if err := r.Client.List(ctx, &ccList, client.MatchingFields{
		clusterRefKey: obj.GetNamespace() + obj.GetName(),
	}); err != nil {
		log.FromContext(ctx).Error(err, "failed to list Cluster objects")
		return nil
	}

	// Return if there is no associated ClusterConnect object.
	if len(ccList.Items) == 0 {
		return nil
	}

	return []ctrl.Request{
		{
			NamespacedName: client.ObjectKeyFromObject(&ccList.Items[0]),
		},
	}
}

func (r *ClusterConnectReconciler) secretToClusterConnectMapper(ctx context.Context, obj client.Object) []ctrl.Request {
	var ccList v1alpha1.ClusterConnectList

	// Find ClusterConnect object that has a given Secret object in kubeconfig-name and kubeconfig-namespace labels.
	if err := r.Client.List(ctx, &ccList, client.MatchingLabels{
		"cluster.x-k8s.io/kubeconfig-name":      obj.GetName(),
		"cluster.x-k8s.io/kubeconfig-namespace": obj.GetNamespace(),
	}); err != nil {
		log.FromContext(ctx).Error(err, "failed to list ClusterConnect objects")
		return nil
	}

	// Return if there is no associated ClusterConnect object.
	if len(ccList.Items) == 0 {
		return nil
	}

	return []ctrl.Request{
		{
			NamespacedName: client.ObjectKeyFromObject(&ccList.Items[0]),
		},
	}
}

func getControlPlaneEndpointUrl(cc *v1alpha1.ClusterConnect) string {
	// TODO: Get scheme separately.
	return fmt.Sprintf("http://%s:%d/kubernetes/%s",
		cc.Status.ControlPlaneEndpoint.Host,
		cc.Status.ControlPlaneEndpoint.Port,
		cc.GetTunnelID(),
	)
}
