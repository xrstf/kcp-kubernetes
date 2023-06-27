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
	kcpcache "github.com/kcp-dev/apimachinery/v2/pkg/cache"
	"github.com/kcp-dev/logicalcluster/v3"

	policyv1beta1 "k8s.io/api/policy/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	policyv1beta1listers "k8s.io/client-go/listers/policy/v1beta1"
	"k8s.io/client-go/tools/cache"
)

// PodDisruptionBudgetClusterLister can list PodDisruptionBudgets across all workspaces, or scope down to a PodDisruptionBudgetLister for one workspace.
// All objects returned here must be treated as read-only.
type PodDisruptionBudgetClusterLister interface {
	// List lists all PodDisruptionBudgets in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*policyv1beta1.PodDisruptionBudget, err error)
	// Cluster returns a lister that can list and get PodDisruptionBudgets in one workspace.
	Cluster(clusterName logicalcluster.Name) policyv1beta1listers.PodDisruptionBudgetLister
	PodDisruptionBudgetClusterListerExpansion
}

type podDisruptionBudgetClusterLister struct {
	indexer cache.Indexer
}

// NewPodDisruptionBudgetClusterLister returns a new PodDisruptionBudgetClusterLister.
// We assume that the indexer:
// - is fed by a cross-workspace LIST+WATCH
// - uses kcpcache.MetaClusterNamespaceKeyFunc as the key function
// - has the kcpcache.ClusterIndex as an index
// - has the kcpcache.ClusterAndNamespaceIndex as an index
func NewPodDisruptionBudgetClusterLister(indexer cache.Indexer) *podDisruptionBudgetClusterLister {
	return &podDisruptionBudgetClusterLister{indexer: indexer}
}

// List lists all PodDisruptionBudgets in the indexer across all workspaces.
func (s *podDisruptionBudgetClusterLister) List(selector labels.Selector) (ret []*policyv1beta1.PodDisruptionBudget, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*policyv1beta1.PodDisruptionBudget))
	})
	return ret, err
}

// Cluster scopes the lister to one workspace, allowing users to list and get PodDisruptionBudgets.
func (s *podDisruptionBudgetClusterLister) Cluster(clusterName logicalcluster.Name) policyv1beta1listers.PodDisruptionBudgetLister {
	return &podDisruptionBudgetLister{indexer: s.indexer, clusterName: clusterName}
}

// podDisruptionBudgetLister implements the policyv1beta1listers.PodDisruptionBudgetLister interface.
type podDisruptionBudgetLister struct {
	indexer     cache.Indexer
	clusterName logicalcluster.Name
}

// List lists all PodDisruptionBudgets in the indexer for a workspace.
func (s *podDisruptionBudgetLister) List(selector labels.Selector) (ret []*policyv1beta1.PodDisruptionBudget, err error) {
	err = kcpcache.ListAllByCluster(s.indexer, s.clusterName, selector, func(i interface{}) {
		ret = append(ret, i.(*policyv1beta1.PodDisruptionBudget))
	})
	return ret, err
}

// PodDisruptionBudgets returns an object that can list and get PodDisruptionBudgets in one namespace.
func (s *podDisruptionBudgetLister) PodDisruptionBudgets(namespace string) policyv1beta1listers.PodDisruptionBudgetNamespaceLister {
	return &podDisruptionBudgetNamespaceLister{indexer: s.indexer, clusterName: s.clusterName, namespace: namespace}
}

// podDisruptionBudgetNamespaceLister implements the policyv1beta1listers.PodDisruptionBudgetNamespaceLister interface.
type podDisruptionBudgetNamespaceLister struct {
	indexer     cache.Indexer
	clusterName logicalcluster.Name
	namespace   string
}

// List lists all PodDisruptionBudgets in the indexer for a given workspace and namespace.
func (s *podDisruptionBudgetNamespaceLister) List(selector labels.Selector) (ret []*policyv1beta1.PodDisruptionBudget, err error) {
	err = kcpcache.ListAllByClusterAndNamespace(s.indexer, s.clusterName, s.namespace, selector, func(i interface{}) {
		ret = append(ret, i.(*policyv1beta1.PodDisruptionBudget))
	})
	return ret, err
}

// Get retrieves the PodDisruptionBudget from the indexer for a given workspace, namespace and name.
func (s *podDisruptionBudgetNamespaceLister) Get(name string) (*policyv1beta1.PodDisruptionBudget, error) {
	key := kcpcache.ToClusterAwareKey(s.clusterName.String(), s.namespace, name)
	obj, exists, err := s.indexer.GetByKey(key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(policyv1beta1.Resource("poddisruptionbudgets"), name)
	}
	return obj.(*policyv1beta1.PodDisruptionBudget), nil
}
