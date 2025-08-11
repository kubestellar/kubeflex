/*
Copyright 2023 The KubeStellar Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package shared

import (
	"context"
	"strconv"
	"time"

	"github.com/pkg/errors"

	tenancyv1alpha1 "github.com/kubestellar/kubeflex/api/v1alpha1"
	"github.com/kubestellar/kubeflex/pkg/util"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	clog "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	// FieldOwner is the identifier used for server-side apply operations across the kubeflex system.
	// This field owner name is used to claim ownership of specific fields in Kubernetes resources
	// when using server-side apply, enabling proper field management and conflict resolution.
	// The domain-based naming follows Kubernetes conventions for field ownership.
	FieldOwner = "kubeflex.kubestellar.io"
)

// BaseReconciler provides common reconciliation functionality and shared resources
// used by other controller reconcilers in the kubeflex system. This struct serves
// as the foundation for all control plane reconcilers, providing consistent access
// to Kubernetes APIs, configuration, and operational utilities.
//
// The BaseReconciler implements common patterns such as:
// - Status management and condition updates
// - Event recording for operational visibility
// - Configuration retrieval from system ConfigMaps
// - Error handling and retry logic
// - Resource ownership and lifecycle management
type BaseReconciler struct {
	// Client provides the controller-runtime client for interacting with Kubernetes APIs
	// using strongly-typed objects. This is the primary interface for CRUD operations
	// on custom resources and standard Kubernetes resources.
	client.Client

	// Scheme contains the runtime type information for all API types that this
	// reconciler needs to work with. It's used for serialization, deserialization,
	// and type conversion operations.
	Scheme *runtime.Scheme

	// Version represents the current version of the kubeflex controller.
	// This can be used for compatibility checks, feature gates, or operational visibility.
	Version string

	// ClientSet provides access to the standard Kubernetes API using the traditional
	// client-go typed interfaces. This is used for operations that require specific
	// client-go functionality not available in the controller-runtime client.
	ClientSet *kubernetes.Clientset

	// DynamicClient enables interaction with arbitrary Kubernetes resources without
	// requiring compile-time knowledge of their types. This is essential for working
	// with custom resources defined by post-create hooks or other dynamic scenarios.
	DynamicClient *dynamic.DynamicClient

	// EventRecorder is used to generate Kubernetes events for operational visibility.
	// Events help operators understand what actions the controller is taking and
	// provide debugging information when issues occur.
	EventRecorder record.EventRecorder
}

// SharedConfig represents the system-wide configuration parameters that are
// shared across all control plane instances. This configuration is typically
// stored in a system ConfigMap and contains infrastructure-level settings
// that affect how control planes are deployed and accessed.
//
// These settings are generally set once during kubeflex installation and
// remain stable across the lifecycle of the system.
type SharedConfig struct {
	// ExternalPort is the port number used for external access to control planes.
	// This port is used when generating external URLs and configuring ingress
	// or load balancer resources for control plane API servers.
	ExternalPort int

	// Domain is the base DNS domain used for control plane external access.
	// Individual control planes will typically get subdomains under this base domain
	// for their external API endpoints (e.g., cp1.example.com, cp2.example.com).
	Domain string

	// HostContainer specifies the container runtime or hosting environment.
	// This affects how networking, storage, and other infrastructure concerns
	// are handled when deploying control plane components.
	HostContainer string

	// IsOpenShift indicates whether the system is running on OpenShift rather than
	// standard Kubernetes. This flag enables OpenShift-specific behaviors such as
	// using different security contexts, route objects instead of ingress, or
	// OpenShift-specific networking configurations.
	IsOpenShift bool

	// ExternalURL is the complete external URL base for accessing control planes.
	// This is typically constructed from the Domain and ExternalPort but may be
	// overridden for complex networking scenarios or when using custom ingress controllers.
	ExternalURL string
}

// UpdateStatusForSyncingError handles reconciliation failures by updating the ControlPlane
// status with error conditions and generating appropriate events. This function provides
// standardized error handling across all reconcilers, ensuring consistent behavior for
// failure scenarios.
//
// The function performs several important operations:
// 1. Records a warning event for operational visibility
// 2. Updates the ControlPlane status with error condition
// 3. Implements specific retry logic for certain error types
// 4. Returns appropriate controller.Result to influence requeue behavior
//
// Special handling is provided for ErrPostCreateHookNotFound, which is treated as a
// temporary condition that should be retried rather than a permanent failure.
//
// Parameters:
//   - hcp: The ControlPlane resource being reconciled
//   - e: The error that occurred during reconciliation
//
// Returns:
//   - ctrl.Result: Controller result indicating requeue behavior
//   - error: Any error that occurred during status update
func (r *BaseReconciler) UpdateStatusForSyncingError(hcp *tenancyv1alpha1.ControlPlane, e error) (ctrl.Result, error) {
	// Record warning event for operational visibility
	// Events appear in kubectl describe and monitoring systems
	if r.EventRecorder != nil {
		r.EventRecorder.Event(hcp, "Warning", "SyncFail", e.Error())
	}

	// Update the ControlPlane status with the error condition
	// This provides structured status information that can be queried programmatically
	tenancyv1alpha1.EnsureCondition(hcp, tenancyv1alpha1.ConditionReconcileError(e))
	
	// Persist the status update to the cluster
	err := r.Status().Update(context.Background(), hcp)
	if err != nil {
		// If status update fails, wrap the original error with the update error
		// This preserves the root cause while indicating the status update problem
		return ctrl.Result{}, errors.Wrap(e, err.Error())
	}

	// Special handling for post-create hook not found errors
	// These are often temporary conditions (hooks being created, network issues)
	if errors.Is(e, ErrPostCreateHookNotFound) {
		// Requeue after 10 seconds instead of using exponential backoff
		// This provides faster recovery when hooks become available
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	// For other errors, return the original error to trigger standard retry behavior
	// The controller-runtime will handle exponential backoff automatically
	return ctrl.Result{}, err
}

// UpdateStatusForSyncingSuccess handles successful reconciliation by updating the
// ControlPlane status with success conditions and generating appropriate events.
// This function provides standardized success handling across all reconcilers,
// ensuring consistent status reporting for successful operations.
//
// The function:
// 1. Records a normal event indicating successful reconciliation
// 2. Updates the ControlPlane status with success condition
// 3. Clears any previous error conditions
//
// Parameters:
//   - ctx: Context for the operation (used for logging and cancellation)
//   - hcp: The ControlPlane resource that was successfully reconciled
//
// Returns:
//   - ctrl.Result: Controller result (typically empty for success cases)
//   - error: Any error that occurred during status update
func (r *BaseReconciler) UpdateStatusForSyncingSuccess(ctx context.Context, hcp *tenancyv1alpha1.ControlPlane) (ctrl.Result, error) {
	// Record success event for operational visibility
	// Empty message is acceptable for success events as the event type conveys the meaning
	if r.EventRecorder != nil {
		r.EventRecorder.Event(hcp, "Normal", "SyncSuccess", "")
	}

	// Get logger from context for potential debugging (currently unused but available)
	_ = clog.FromContext(ctx)

	// Update the ControlPlane status with success condition
	// This clears any previous error conditions and indicates healthy state
	tenancyv1alpha1.EnsureCondition(hcp, tenancyv1alpha1.ConditionReconcileSuccess())
	
	// Persist the status update to the cluster
	err := r.Status().Update(context.Background(), hcp)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Success case - no requeue needed, no error to report
	return ctrl.Result{}, err
}

// GetConfig retrieves the system-wide configuration from the kubeflex system ConfigMap.
// This function provides access to infrastructure-level settings that are shared across
// all control plane instances in the cluster.
//
// The configuration is stored in a well-known ConfigMap in the system namespace and
// includes settings such as:
// - External access configuration (domain, port)
// - Platform-specific settings (OpenShift vs Kubernetes)
// - Container runtime and networking settings
//
// This function handles the parsing and type conversion of ConfigMap data into
// strongly-typed configuration values, providing error handling for malformed
// or missing configuration data.
//
// Parameters:
//   - ctx: Context for the operation (currently unused but available for future use)
//
// Returns:
//   - *SharedConfig: Parsed configuration object with all system settings
//   - error: Any error encountered during ConfigMap retrieval or parsing
func (r *BaseReconciler) GetConfig(ctx context.Context) (*SharedConfig, error) {
	// Create template for the system ConfigMap
	cmap := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      util.SystemConfigMap,
			Namespace: util.SystemNamespace,
		},
	}

	// Retrieve the ConfigMap from the cluster
	err := r.Client.Get(context.TODO(), client.ObjectKeyFromObject(cmap), cmap, &client.GetOptions{})
	if err != nil {
		return nil, err
	}

	// Parse and validate the external port setting
	// This port is used for generating external URLs and ingress configurations
	port, err := strconv.Atoi(cmap.Data["externalPort"])
	if err != nil {
		return nil, err
	}

	// Parse the OpenShift flag
	// This boolean determines platform-specific behavior throughout the system
	isOpenShift, err := strconv.ParseBool(cmap.Data["isOpenShift"])
	if err != nil {
		return nil, err
	}

	// Construct and return the configuration object
	return &SharedConfig{
		Domain:        cmap.Data["domain"],        // Base DNS domain for control planes
		HostContainer: cmap.Data["hostContainer"], // Container runtime information
		ExternalPort:  port,                       // Parsed external access port
		IsOpenShift:   isOpenShift,                // Parsed platform flag
	}, nil
}

// UpdateStatusWithSecretRef updates the ControlPlane status to include a reference
// to a secret containing important configuration or credential information.
// This function provides a standardized way to link ControlPlanes to their
// associated secrets for kubeconfig access, certificates, or other sensitive data.
//
// The SecretRef in the ControlPlane status serves multiple purposes:
// 1. Provides clients with information about where to find access credentials
// 2. Enables garbage collection and lifecycle management of related secrets
// 3. Supports both external access patterns and in-cluster access patterns
//
// The function automatically determines the correct namespace for the secret
// based on the ControlPlane name, ensuring proper isolation between different
// control plane instances.
//
// Parameters:
//   - hcp: The ControlPlane resource to update
//   - secretName: The name of the secret containing the referenced data
//   - key: The key within the secret for external client access
//   - inClusterKey: The key within the secret for in-cluster access (may be different)
func (r *BaseReconciler) UpdateStatusWithSecretRef(hcp *tenancyv1alpha1.ControlPlane, secretName, key, inClusterKey string) {
	// Generate the appropriate namespace for this control plane's resources
	// This ensures secrets are created in the correct namespace for isolation
	namespace := util.GenerateNamespaceFromControlPlaneName(hcp.Name)

	// Update the status with complete secret reference information
	hcp.Status.SecretRef = &tenancyv1alpha1.SecretReference{
		Name:      secretName,   // Name of the secret resource
		Namespace: namespace,    // Namespace where the secret exists
		Key:       key,          // Key for external client access (e.g., "kubeconfig")
		InClusterKey: inClusterKey, // Key for in-cluster access (e.g., "kubeconfig-incluster")
	}
}