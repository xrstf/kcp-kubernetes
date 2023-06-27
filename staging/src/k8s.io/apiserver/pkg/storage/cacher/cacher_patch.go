/*
Copyright 2023 The KCP Authors.

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

package cacher

func (lw *cacherListerWatcher) kcpAwareResourcePrefix() string {
	if lw.kcpExtraStorageMetadata.Cluster.Wildcard {
		return lw.resourcePrefix
	}

	// This is a request for normal (non-bound) CRs outside of system:system-crds. Make sure we only list in the
	// specific logical cluster.
	return lw.resourcePrefix + "/" + lw.kcpExtraStorageMetadata.Cluster.Name.String()
}
