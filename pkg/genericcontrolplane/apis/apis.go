/*
Copyright 2014 The Kubernetes Authors.

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

package apis

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	kcpkubernetesinformers "github.com/kcp-dev/client-go/informers"
	kcpkubernetesclientset "github.com/kcp-dev/client-go/kubernetes"
	"github.com/kcp-dev/logicalcluster/v3"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	admissionregistrationv1alpha1 "k8s.io/api/admissionregistration/v1alpha1"
	apiserverinternalv1alpha1 "k8s.io/api/apiserverinternal/v1alpha1"
	authenticationv1 "k8s.io/api/authentication/v1"
	authenticationv1alpha1 "k8s.io/api/authentication/v1alpha1"
	authorizationapiv1 "k8s.io/api/authorization/v1"
	certificatesapiv1 "k8s.io/api/certificates/v1"
	coordinationapiv1 "k8s.io/api/coordination/v1"
	apiv1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	discoveryv1beta1 "k8s.io/api/discovery/v1beta1"
	eventsv1 "k8s.io/api/events/v1"
	eventsv1beta1 "k8s.io/api/events/v1beta1"
	flowcontrolv1alpha1 "k8s.io/api/flowcontrol/v1alpha1"
	flowcontrolv1beta1 "k8s.io/api/flowcontrol/v1beta1"
	flowcontrolv1beta2 "k8s.io/api/flowcontrol/v1beta2"
	flowcontrolv1beta3 "k8s.io/api/flowcontrol/v1beta3"
	rbacv1 "k8s.io/api/rbac/v1"
	resourcev1alpha2 "k8s.io/api/resource/v1alpha2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/apiserver/pkg/endpoints/discovery"
	apiserverfeatures "k8s.io/apiserver/pkg/features"
	"k8s.io/apiserver/pkg/registry/generic"
	genericapiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/apiserver/pkg/server/dynamiccertificates"
	serverstorage "k8s.io/apiserver/pkg/server/storage"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	"k8s.io/component-helpers/apimachinery/lease"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/controlplane/controller/apiserverleasegc"
	"k8s.io/kubernetes/pkg/controlplane/controller/clusterauthenticationtrust"
	"k8s.io/kubernetes/pkg/controlplane/controller/legacytokentracking"

	// RESTStorage installers
	admissionregistrationrest "k8s.io/kubernetes/pkg/registry/admissionregistration/rest"
	apiserverinternalrest "k8s.io/kubernetes/pkg/registry/apiserverinternal/rest"
	authenticationrest "k8s.io/kubernetes/pkg/registry/authentication/rest"
	authorizationrest "k8s.io/kubernetes/pkg/registry/authorization/rest"
	certificatesrest "k8s.io/kubernetes/pkg/registry/certificates/rest"
	coordinationrest "k8s.io/kubernetes/pkg/registry/coordination/rest"
	corerest "k8s.io/kubernetes/pkg/registry/core/rest/genericcontrolplane"
	eventsrest "k8s.io/kubernetes/pkg/registry/events/rest"
	rbacrest "k8s.io/kubernetes/pkg/registry/rbac/rest"
	"k8s.io/kubernetes/pkg/routes"
	"k8s.io/kubernetes/pkg/serviceaccount"
	"k8s.io/utils/clock"
)

const (
	// DefaultEndpointReconcilerInterval is the default amount of time for how often the endpoints for
	// the kubernetes Service are reconciled.
	DefaultEndpointReconcilerInterval = 10 * time.Second
	// DefaultEndpointReconcilerTTL is the default TTL timeout for the storage layer
	DefaultEndpointReconcilerTTL = 15 * time.Second
	// IdentityLeaseComponentLabelKey is used to apply a component label to identity lease objects, indicating:
	//   1. the lease is an identity lease (different from leader election leases)
	//   2. which component owns this lease
	IdentityLeaseComponentLabelKey = "k8s.io/component"
	// KubeAPIServer defines variable used internally when referring to kube-apiserver component
	KubeAPIServer = "kube-apiserver"
	// KubeAPIServerIdentityLeaseLabelSelector selects kube-apiserver identity leases
	KubeAPIServerIdentityLeaseLabelSelector = IdentityLeaseComponentLabelKey + "=" + KubeAPIServer
)

var (
	// IdentityLeaseGCPeriod is the interval which the lease GC controller checks for expired leases
	// IdentityLeaseGCPeriod is exposed so integration tests can tune this value.
	IdentityLeaseGCPeriod = 3600 * time.Second
	// IdentityLeaseDurationSeconds is the duration of kube-apiserver lease in seconds
	// IdentityLeaseDurationSeconds is exposed so integration tests can tune this value.
	IdentityLeaseDurationSeconds = 3600
	// IdentityLeaseRenewIntervalSeconds is the interval of kube-apiserver renewing its lease in seconds
	// IdentityLeaseRenewIntervalSeconds is exposed so integration tests can tune this value.
	IdentityLeaseRenewIntervalPeriod = 10 * time.Second
)

// ExtraConfig defines extra configuration for the master
type ExtraConfig struct {
	ClusterAuthenticationInfo clusterauthenticationtrust.ClusterAuthenticationInfo

	APIResourceConfigSource serverstorage.APIResourceConfigSource
	StorageFactory          serverstorage.StorageFactory
	EventTTL                time.Duration

	EnableLogsSupport bool
	ProxyTransport    *http.Transport

	// Port for the apiserver service.
	APIServerServicePort int

	ServiceAccountIssuer        serviceaccount.TokenGenerator
	ServiceAccountMaxExpiration time.Duration
	ExtendExpiration            bool

	// ServiceAccountIssuerDiscovery
	ServiceAccountIssuerURL  string
	ServiceAccountJWKSURI    string
	ServiceAccountPublicKeys []interface{}

	VersionedInformers kcpkubernetesinformers.SharedInformerFactory
}

// Config defines configuration for the master
type Config struct {
	GenericConfig *genericapiserver.Config
	ExtraConfig   ExtraConfig
}

type completedConfig struct {
	GenericConfig genericapiserver.CompletedConfig
	ExtraConfig   *ExtraConfig
}

// CompletedConfig embeds a private pointer that cannot be instantiated outside of this package
type CompletedConfig struct {
	*completedConfig
}

// GenericControlPlane contains state for a Kubernetes cluster master/api server.
type GenericControlPlane struct {
	GenericAPIServer *genericapiserver.GenericAPIServer

	ClusterAuthenticationInfo clusterauthenticationtrust.ClusterAuthenticationInfo
}

// Complete fills in any fields not set that are required to have valid data. It's mutating the receiver.
func (c *Config) Complete() CompletedConfig {
	cfg := completedConfig{
		c.GenericConfig.Complete(nil), // TODO: when we have users of this, we will need a shared informer factory super-interface hack on both ends so as to not have to refactor the world
		&c.ExtraConfig,
	}

	discoveryAddresses := discovery.DefaultAddresses{DefaultAddress: cfg.GenericConfig.ExternalAddress}
	cfg.GenericConfig.DiscoveryAddresses = discoveryAddresses

	return CompletedConfig{&cfg}
}

// New returns a new instance of GenericControlPlane from the given config.
// Certain config fields will be set to a default value if unset.
// Certain config fields must be specified, including:
//
//	KubeletClientConfig
func (c completedConfig) New(delegationTarget genericapiserver.DelegationTarget) (*GenericControlPlane, error) {
	s, err := c.GenericConfig.New("kube-control-plane", delegationTarget)
	if err != nil {
		return nil, err
	}

	if c.ExtraConfig.EnableLogsSupport {
		routes.Logs{}.Install(s.Handler.GoRestfulContainer)
	}

	m := &GenericControlPlane{
		GenericAPIServer:          s,
		ClusterAuthenticationInfo: c.ExtraConfig.ClusterAuthenticationInfo,
	}

	// install legacy rest storage
	if err := m.InstallLegacyAPI(&c, c.GenericConfig.RESTOptionsGetter); err != nil {
		return nil, err
	}

	clientset, err := kcpkubernetesclientset.NewForConfig(c.GenericConfig.LoopbackClientConfig)
	if err != nil {
		return nil, err
	}

	// TODO: update to a version that caches success but will recheck on failure, unlike memcache discovery
	discoveryClientForAdmissionRegistration := clientset.Discovery()

	// The order here is preserved in discovery.
	// If resources with identical names exist in more than one of these groups (e.g. "deployments.apps"" and "deployments.extensions"),
	// the order of this list determines which group an unqualified resource name (e.g. "deployments") should prefer.
	// This priority order is used for local discovery, but it ends up aggregated in `k8s.io/kubernetes/cmd/kube-apiserver/app/aggregator.go
	// with specific priorities.
	// TODO: describe the priority all the way down in the RESTStorageProviders and plumb it back through the various discovery
	// handlers that we have.
	restStorageProviders := []RESTStorageProvider{
		apiserverinternalrest.StorageProvider{},
		authenticationrest.RESTStorageProvider{Authenticator: c.GenericConfig.Authentication.Authenticator, APIAudiences: c.GenericConfig.Authentication.APIAudiences},
		authorizationrest.RESTStorageProvider{Authorizer: c.GenericConfig.Authorization.Authorizer, RuleResolver: c.GenericConfig.RuleResolver},
		certificatesrest.RESTStorageProvider{},
		coordinationrest.RESTStorageProvider{},
		rbacrest.RESTStorageProvider{Authorizer: c.GenericConfig.Authorization.Authorizer},
		admissionregistrationrest.RESTStorageProvider{Authorizer: c.GenericConfig.Authorization.Authorizer, DiscoveryClient: discoveryClientForAdmissionRegistration},
		eventsrest.RESTStorageProvider{TTL: c.ExtraConfig.EventTTL},
	}
	if err := m.InstallAPIs(c.ExtraConfig.APIResourceConfigSource, c.GenericConfig.RESTOptionsGetter, restStorageProviders...); err != nil {
		return nil, err
	}

	rootCluster := logicalcluster.Name("root")

	// Have to create this outside the post-start hook to ensure the informer is started correctly
	rootConfigMapsInformer := c.ExtraConfig.VersionedInformers.Core().V1().ConfigMaps().Cluster(rootCluster)
	// Register the informer for starting when the factory is started in a different post-start hook
	_ = rootConfigMapsInformer.Informer()

	m.GenericAPIServer.AddPostStartHookOrDie("start-cluster-authentication-info-controller", func(hookContext genericapiserver.PostStartHookContext) error {
		kubeClient, err := kcpkubernetesclientset.NewForConfig(hookContext.LoopbackClientConfig)
		if err != nil {
			return err
		}

		controller := clusterauthenticationtrust.NewClusterAuthenticationTrustController(m.ClusterAuthenticationInfo, kubeClient.Cluster(rootCluster.Path()))

		// generate a context  from stopCh. This is to avoid modifying files which are relying on apiserver
		// TODO: See if we can pass ctx to the current method
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			select {
			case <-hookContext.StopCh:
				cancel() // stopCh closed, so cancel our context
			case <-ctx.Done():
			}
		}()

		// prime values and start listeners
		if m.ClusterAuthenticationInfo.ClientCA != nil {
			m.ClusterAuthenticationInfo.ClientCA.AddListener(controller)
			if controller, ok := m.ClusterAuthenticationInfo.ClientCA.(dynamiccertificates.ControllerRunner); ok {
				// runonce to be sure that we have a value.
				if err := controller.RunOnce(ctx); err != nil {
					runtime.HandleError(err)
				}
				go controller.Run(ctx, 1)
			}
		}
		if m.ClusterAuthenticationInfo.RequestHeaderCA != nil {
			m.ClusterAuthenticationInfo.RequestHeaderCA.AddListener(controller)
			if controller, ok := m.ClusterAuthenticationInfo.RequestHeaderCA.(dynamiccertificates.ControllerRunner); ok {
				// runonce to be sure that we have a value.
				if err := controller.RunOnce(ctx); err != nil {
					runtime.HandleError(err)
				}
				go controller.Run(ctx, 1)
			}
		}

		go controller.Run(ctx, 1)
		return nil
	})

	if utilfeature.DefaultFeatureGate.Enabled(apiserverfeatures.APIServerIdentity) {
		m.GenericAPIServer.AddPostStartHookOrDie("start-kube-apiserver-identity-lease-controller", func(hookContext genericapiserver.PostStartHookContext) error {
			kubeClient, err := kcpkubernetesclientset.NewForConfig(hookContext.LoopbackClientConfig)
			if err != nil {
				return err
			}

			leaseName := m.GenericAPIServer.APIServerID
			holderIdentity := m.GenericAPIServer.APIServerID + "_" + string(uuid.NewUUID())

			controller := lease.NewController(
				clock.RealClock{},
				kubeClient.Cluster(rootCluster.Path()),
				holderIdentity,
				int32(IdentityLeaseDurationSeconds),
				nil,
				IdentityLeaseRenewIntervalPeriod,
				leaseName,
				metav1.NamespaceSystem,
				labelAPIServerHeartbeat)
			go controller.Run(hookContext.StopCh)
			return nil
		})
		m.GenericAPIServer.AddPostStartHookOrDie("start-kube-apiserver-identity-lease-garbage-collector", func(hookContext genericapiserver.PostStartHookContext) error {
			kubeClient, err := kcpkubernetesclientset.NewForConfig(hookContext.LoopbackClientConfig)
			if err != nil {
				return err
			}
			go apiserverleasegc.NewAPIServerLeaseGC(
				kubeClient.Cluster(rootCluster.Path()),
				IdentityLeaseGCPeriod,
				metav1.NamespaceSystem,
				KubeAPIServerIdentityLeaseLabelSelector,
			).Run(hookContext.StopCh)
			return nil
		})
	}

	m.GenericAPIServer.AddPostStartHookOrDie("start-legacy-token-tracking-controller", func(hookContext genericapiserver.PostStartHookContext) error {
		kubeClient, err := kcpkubernetesclientset.NewForConfig(hookContext.LoopbackClientConfig)
		if err != nil {
			return err
		}
		go legacytokentracking.NewController(kubeClient.Cluster(rootCluster.Path())).Run(hookContext.StopCh)
		return nil
	})

	return m, nil
}

func labelAPIServerHeartbeat(lease *coordinationapiv1.Lease) error {
	if lease.Labels == nil {
		lease.Labels = map[string]string{}
	}
	// This label indicates that kube-apiserver owns this identity lease object
	lease.Labels[IdentityLeaseComponentLabelKey] = KubeAPIServer

	hostname, err := os.Hostname()
	if err != nil {
		return err
	}

	// convenience label to easily map a lease object to a specific apiserver
	lease.Labels[apiv1.LabelHostname] = hostname
	return nil
}

// InstallLegacyAPI will install the legacy APIs for the restStorageProviders if they are enabled.
func (m *GenericControlPlane) InstallLegacyAPI(c *completedConfig, restOptionsGetter generic.RESTOptionsGetter) error {
	legacyRESTStorageProvider := corerest.LegacyRESTStorageProvider{
		StorageFactory:              c.ExtraConfig.StorageFactory,
		EventTTL:                    c.ExtraConfig.EventTTL,
		ServiceAccountIssuer:        c.ExtraConfig.ServiceAccountIssuer,
		ExtendExpiration:            c.ExtraConfig.ExtendExpiration,
		ServiceAccountMaxExpiration: c.ExtraConfig.ServiceAccountMaxExpiration,
		APIAudiences:                c.GenericConfig.Authentication.APIAudiences,
	}
	_, apiGroupInfo, err := legacyRESTStorageProvider.NewLegacyRESTStorage(c.ExtraConfig.APIResourceConfigSource, restOptionsGetter)
	if err != nil {
		return fmt.Errorf("error building core storage: %v", err)
	}
	if len(apiGroupInfo.VersionedResourcesStorageMap) == 0 { // if all core storage is disabled, return.
		return nil
	}

	if err := m.GenericAPIServer.InstallLegacyAPIGroup(genericapiserver.DefaultLegacyAPIPrefix, &apiGroupInfo); err != nil {
		return fmt.Errorf("error in registering group versions: %v", err)
	}
	return nil
}

// RESTStorageProvider is a factory type for REST storage.
type RESTStorageProvider interface {
	GroupName() string
	NewRESTStorage(apiResourceConfigSource serverstorage.APIResourceConfigSource, restOptionsGetter generic.RESTOptionsGetter) (genericapiserver.APIGroupInfo, error)
}

// InstallAPIs will install the APIs for the restStorageProviders if they are enabled.
func (m *GenericControlPlane) InstallAPIs(apiResourceConfigSource serverstorage.APIResourceConfigSource, restOptionsGetter generic.RESTOptionsGetter, restStorageProviders ...RESTStorageProvider) error {
	apiGroupsInfo := []*genericapiserver.APIGroupInfo{}

	// used later in the loop to filter the served resource by those that have expired.
	resourceExpirationEvaluator, err := genericapiserver.NewResourceExpirationEvaluator(*m.GenericAPIServer.Version)
	if err != nil {
		return err
	}

	for _, restStorageBuilder := range restStorageProviders {
		groupName := restStorageBuilder.GroupName()
		apiGroupInfo, err := restStorageBuilder.NewRESTStorage(apiResourceConfigSource, restOptionsGetter)
		if err != nil {
			return fmt.Errorf("problem initializing API group %q : %v", groupName, err)
		}
		if len(apiGroupInfo.VersionedResourcesStorageMap) == 0 {
			// If we have no storage for any resource configured, this API group is effectively disabled.
			// This can happen when an entire API group, version, or development-stage (alpha, beta, GA) is disabled.
			klog.Infof("API group %q is not enabled, skipping.", groupName)
			continue
		}

		// Remove resources that serving kinds that are removed.
		// We do this here so that we don't accidentally serve versions without resources or openapi information that for kinds we don't serve.
		// This is a spot above the construction of individual storage handlers so that no sig accidentally forgets to check.
		resourceExpirationEvaluator.RemoveDeletedKinds(groupName, apiGroupInfo.Scheme, apiGroupInfo.VersionedResourcesStorageMap)
		if len(apiGroupInfo.VersionedResourcesStorageMap) == 0 {
			klog.V(1).Infof("Removing API group %v because it is time to stop serving it because it has no versions per APILifecycle.", groupName)
			continue
		}

		klog.V(1).Infof("Enabling API group %q.", groupName)

		if postHookProvider, ok := restStorageBuilder.(genericapiserver.PostStartHookProvider); ok {
			name, hook, err := postHookProvider.PostStartHook()
			if err != nil {
				klog.Fatalf("Error building PostStartHook: %v", err)
			}
			m.GenericAPIServer.AddPostStartHookOrDie(name, hook)
		}

		apiGroupsInfo = append(apiGroupsInfo, &apiGroupInfo)
	}

	if err := m.GenericAPIServer.InstallAPIGroups(apiGroupsInfo...); err != nil {
		return fmt.Errorf("error in registering group versions: %v", err)
	}
	return nil
}

var (
	// stableAPIGroupVersionsEnabledByDefault is a list of our stable versions.
	stableAPIGroupVersionsEnabledByDefault = []schema.GroupVersion{
		admissionregistrationv1.SchemeGroupVersion,
		apiv1.SchemeGroupVersion,
		//appsv1.SchemeGroupVersion,
		authenticationv1.SchemeGroupVersion,
		authorizationapiv1.SchemeGroupVersion,
		//autoscalingapiv1.SchemeGroupVersion,
		//autoscalingapiv2.SchemeGroupVersion,
		//batchapiv1.SchemeGroupVersion,
		certificatesapiv1.SchemeGroupVersion,
		coordinationapiv1.SchemeGroupVersion,
		discoveryv1.SchemeGroupVersion,
		eventsv1.SchemeGroupVersion,
		//networkingapiv1.SchemeGroupVersion,
		//nodev1.SchemeGroupVersion,
		//policyapiv1.SchemeGroupVersion,
		rbacv1.SchemeGroupVersion,
		//storageapiv1.SchemeGroupVersion,
		//schedulingapiv1.SchemeGroupVersion,
	}

	// legacyBetaEnabledByDefaultResources is the list of beta resources we enable.  You may only add to this list
	// if your resource is already enabled by default in a beta level we still serve AND there is no stable API for it.
	// see https://github.com/kubernetes/enhancements/tree/master/keps/sig-architecture/3136-beta-apis-off-by-default
	// for more details.
	legacyBetaEnabledByDefaultResources = []schema.GroupVersionResource{
		//autoscalingapiv2beta2.SchemeGroupVersion.WithResource("horizontalpodautoscalers"), // remove in 1.26
		//storageapiv1beta1.SchemeGroupVersion.WithResource("csistoragecapacities"),         // remove in 1.27
		flowcontrolv1beta1.SchemeGroupVersion.WithResource("flowschemas"),                 // remove in 1.26
		flowcontrolv1beta1.SchemeGroupVersion.WithResource("prioritylevelconfigurations"), // remove in 1.26
		flowcontrolv1beta2.SchemeGroupVersion.WithResource("flowschemas"),                 // remove in 1.29
		flowcontrolv1beta2.SchemeGroupVersion.WithResource("prioritylevelconfigurations"), // remove in 1.29
		flowcontrolv1beta3.SchemeGroupVersion.WithResource("flowschemas"),                 // deprecate in 1.29, remove in 1.32
		flowcontrolv1beta3.SchemeGroupVersion.WithResource("prioritylevelconfigurations"), // deprecate in 1.29, remove in 1.32
	}
	// betaAPIGroupVersionsDisabledByDefault is for all future beta groupVersions.
	betaAPIGroupVersionsDisabledByDefault = []schema.GroupVersion{
		//autoscalingapiv2beta1.SchemeGroupVersion,
		//autoscalingapiv2beta2.SchemeGroupVersion,
		//batchapiv1beta1.SchemeGroupVersion,
		discoveryv1beta1.SchemeGroupVersion,
		eventsv1beta1.SchemeGroupVersion,
		//nodev1beta1.SchemeGroupVersion, // remove in 1.26
		//policyapiv1beta1.SchemeGroupVersion,
		//storageapiv1beta1.SchemeGroupVersion,
		flowcontrolv1beta1.SchemeGroupVersion,
		flowcontrolv1beta2.SchemeGroupVersion,
		flowcontrolv1beta3.SchemeGroupVersion,
	}

	// alphaAPIGroupVersionsDisabledByDefault holds the alpha APIs we have.  They are always disabled by default.
	alphaAPIGroupVersionsDisabledByDefault = []schema.GroupVersion{
		admissionregistrationv1alpha1.SchemeGroupVersion,
		apiserverinternalv1alpha1.SchemeGroupVersion,
		authenticationv1alpha1.SchemeGroupVersion,
		resourcev1alpha2.SchemeGroupVersion,
		//networkingapiv1alpha1.SchemeGroupVersion,
		//storageapiv1alpha1.SchemeGroupVersion,
		flowcontrolv1alpha1.SchemeGroupVersion,
	}
)

// DefaultAPIResourceConfigSource returns default configuration for an APIResource.
func DefaultAPIResourceConfigSource() *serverstorage.ResourceConfig {
	ret := serverstorage.NewResourceConfig()
	// NOTE: GroupVersions listed here will be enabled by default. Don't put alpha or beta versions in the list.
	ret.EnableVersions(stableAPIGroupVersionsEnabledByDefault...)
	// disable alpha and beta versions explicitly so we have a full list of what's possible to serve
	ret.DisableVersions(betaAPIGroupVersionsDisabledByDefault...)
	ret.DisableVersions(alphaAPIGroupVersionsDisabledByDefault...)

	// enable the legacy beta resources that were present before stopped serving new beta APIs by default.
	ret.EnableResources(legacyBetaEnabledByDefaultResources...)

	return ret
}
