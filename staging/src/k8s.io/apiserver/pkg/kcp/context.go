/*
Copyright 2022 The KCP Authors.

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

package kcp

import (
	"context"

	"github.com/kcp-dev/logicalcluster/v3"
)

type key int

const (
	clusterKey key = iota
	crdKey
)

// WithCustomResourceIndicator wraps ctx and returns a new context.Context that indicates the current request is for a
// CustomResource. This is required to support wildcard (cross-cluster) partial metadata requests, as the keys in
// storage for built-in types and custom resources differ in format. Built-in types have the format
// /registry/$group/$resource/$cluster/[$namespace]/$name, whereas custom resources have the format
// /registry/$group/$resource/$identity/$cluster/[$namespace]/$name.
func WithCustomResourceIndicator(ctx context.Context) context.Context {
	return context.WithValue(ctx, crdKey, true)
}

// CustomResourceIndicatorFrom returns true if this is a custom resource request.
func CustomResourceIndicatorFrom(ctx context.Context) bool {
	v := ctx.Value(crdKey)

	if v == nil {
		return false
	}

	return v.(bool)
}

// WithCluster injects a cluster name into a context.
func WithCluster(ctx context.Context, cluster logicalcluster.Name) context.Context {
	return context.WithValue(ctx, clusterKey, cluster)
}

// ClusterFrom extracts a cluster name from the context.
func ClusterFrom(ctx context.Context) (logicalcluster.Name, bool) {
	s, ok := ctx.Value(clusterKey).(logicalcluster.Name)
	return s, ok
}
