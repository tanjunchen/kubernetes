/*
Copyright 2020 The Kubernetes Authors.

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

package corev1

import (
	v1 "k8s.io/api/core/v1"
)

// PodPriority returns priority of the given pod.
func PodPriority(pod *v1.Pod) int32 {
	if pod.Spec.Priority != nil {
		return *pod.Spec.Priority
	}
	// When priority of a running pod is nil, it means it was created at a time
	// that there was no global default priority class and the priority class
	// name of the pod was empty. So, we resolve to the static default priority.
	return 0
}

// GetZoneKey is a helper function that builds a string identifier that is unique per failure-zone;
// it returns empty-string for no zone.
// Since there are currently two separate zone keys:
//   * "failure-domain.beta.kubernetes.io/zone"
//   * "topology.kubernetes.io/zone"
// GetZoneKey will first check failure-domain.beta.kubernetes.io/zone and if not exists, will then check
// topology.kubernetes.io/zone
func GetZoneKey(node *v1.Node) string {
	labels := node.Labels
	if labels == nil {
		return ""
	}

	// TODO: prefer stable labels for zone in v1.18
	zone, ok := labels[v1.LabelZoneFailureDomain]
	if !ok {
		zone, _ = labels[v1.LabelZoneFailureDomainStable]
	}

	// TODO: prefer stable labels for region in v1.18
	region, ok := labels[v1.LabelZoneRegion]
	if !ok {
		region, _ = labels[v1.LabelZoneRegionStable]
	}

	if region == "" && zone == "" {
		return ""
	}

	// We include the null character just in case region or failureDomain has a colon
	// (We do assume there's no null characters in a region or failureDomain)
	// As a nice side-benefit, the null character is not printed by fmt.Print or glog
	return region + ":\x00:" + zone
}
