//go:build !ignore_autogenerated
// +build !ignore_autogenerated

/*
Copyright The KCP Authors.

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

// Code generated by kcp code-generator. DO NOT EDIT.

package v1beta1

import (
	"context"
	"time"

	kcpcache "github.com/kcp-dev/apimachinery/v2/pkg/cache"
	kcpinformers "github.com/kcp-dev/apimachinery/v2/third_party/informers"
	"github.com/kcp-dev/logicalcluster/v3"

	policyv1beta1 "k8s.io/api/policy/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	upstreampolicyv1beta1informers "k8s.io/client-go/informers/policy/v1beta1"
	upstreampolicyv1beta1listers "k8s.io/client-go/listers/policy/v1beta1"
	"k8s.io/client-go/tools/cache"

	"github.com/kcp-dev/client-go/informers/internalinterfaces"
	clientset "github.com/kcp-dev/client-go/kubernetes"
	policyv1beta1listers "github.com/kcp-dev/client-go/listers/policy/v1beta1"
)

// PodSecurityPolicyClusterInformer provides access to a shared informer and lister for
// PodSecurityPolicies.
type PodSecurityPolicyClusterInformer interface {
	Cluster(logicalcluster.Name) upstreampolicyv1beta1informers.PodSecurityPolicyInformer
	Informer() kcpcache.ScopeableSharedIndexInformer
	Lister() policyv1beta1listers.PodSecurityPolicyClusterLister
}

type podSecurityPolicyClusterInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// NewPodSecurityPolicyClusterInformer constructs a new informer for PodSecurityPolicy type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewPodSecurityPolicyClusterInformer(client clientset.ClusterInterface, resyncPeriod time.Duration, indexers cache.Indexers) kcpcache.ScopeableSharedIndexInformer {
	return NewFilteredPodSecurityPolicyClusterInformer(client, resyncPeriod, indexers, nil)
}

// NewFilteredPodSecurityPolicyClusterInformer constructs a new informer for PodSecurityPolicy type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredPodSecurityPolicyClusterInformer(client clientset.ClusterInterface, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) kcpcache.ScopeableSharedIndexInformer {
	return kcpinformers.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.PolicyV1beta1().PodSecurityPolicies().List(context.TODO(), options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.PolicyV1beta1().PodSecurityPolicies().Watch(context.TODO(), options)
			},
		},
		&policyv1beta1.PodSecurityPolicy{},
		resyncPeriod,
		indexers,
	)
}

func (f *podSecurityPolicyClusterInformer) defaultInformer(client clientset.ClusterInterface, resyncPeriod time.Duration) kcpcache.ScopeableSharedIndexInformer {
	return NewFilteredPodSecurityPolicyClusterInformer(client, resyncPeriod, cache.Indexers{
		kcpcache.ClusterIndexName: kcpcache.ClusterIndexFunc,
	},
		f.tweakListOptions,
	)
}

func (f *podSecurityPolicyClusterInformer) Informer() kcpcache.ScopeableSharedIndexInformer {
	return f.factory.InformerFor(&policyv1beta1.PodSecurityPolicy{}, f.defaultInformer)
}

func (f *podSecurityPolicyClusterInformer) Lister() policyv1beta1listers.PodSecurityPolicyClusterLister {
	return policyv1beta1listers.NewPodSecurityPolicyClusterLister(f.Informer().GetIndexer())
}

func (f *podSecurityPolicyClusterInformer) Cluster(clusterName logicalcluster.Name) upstreampolicyv1beta1informers.PodSecurityPolicyInformer {
	return &podSecurityPolicyInformer{
		informer: f.Informer().Cluster(clusterName),
		lister:   f.Lister().Cluster(clusterName),
	}
}

type podSecurityPolicyInformer struct {
	informer cache.SharedIndexInformer
	lister   upstreampolicyv1beta1listers.PodSecurityPolicyLister
}

func (f *podSecurityPolicyInformer) Informer() cache.SharedIndexInformer {
	return f.informer
}

func (f *podSecurityPolicyInformer) Lister() upstreampolicyv1beta1listers.PodSecurityPolicyLister {
	return f.lister
}
