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

package predicates

import (
	"fmt"
	"math/rand"
	"strconv"
	"sync"
	"time"

	"github.com/golang/glog"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/client/cache"
	"k8s.io/kubernetes/pkg/kubelet/qos"
	"k8s.io/kubernetes/pkg/labels"
	utilruntime "k8s.io/kubernetes/pkg/util/runtime"
	"k8s.io/kubernetes/pkg/util/workqueue"
	"k8s.io/kubernetes/plugin/pkg/scheduler/algorithm"
	priorityutil "k8s.io/kubernetes/plugin/pkg/scheduler/algorithm/priorities/util"
	"k8s.io/kubernetes/plugin/pkg/scheduler/schedulercache"
)

type NodeInfo interface {
	GetNodeInfo(nodeID string) (*api.Node, error)
}

type PersistentVolumeInfo interface {
	GetPersistentVolumeInfo(pvID string) (*api.PersistentVolume, error)
}

type PersistentVolumeClaimInfo interface {
	GetPersistentVolumeClaimInfo(namespace string, pvcID string) (*api.PersistentVolumeClaim, error)
}

type CachedNodeInfo struct {
	*cache.StoreToNodeLister
}

// GetNodeInfo returns cached data for the node 'id'.
func (c *CachedNodeInfo) GetNodeInfo(id string) (*api.Node, error) {
	node, exists, err := c.Get(&api.Node{ObjectMeta: api.ObjectMeta{Name: id}})

	if err != nil {
		return nil, fmt.Errorf("error retrieving node '%v' from cache: %v", id, err)
	}

	if !exists {
		return nil, fmt.Errorf("node '%v' not found", id)
	}

	return node.(*api.Node), nil
}

// podMetadata is a type that is passed as metadata for predicate functions
type predicateMetadata struct {
	podBestEffort             bool
	podRequest                *schedulercache.Resource
	podPorts                  map[int]bool
	matchingAntiAffinityTerms []matchingPodAntiAffinityTerm
}

type matchingPodAntiAffinityTerm struct {
	term *api.PodAffinityTerm
	node *api.Node
}

func PredicateMetadata(pod *api.Pod, nodeInfoMap map[string]*schedulercache.NodeInfo) interface{} {
	// If we cannot compute metadata, just return nil
	if pod == nil {
		return nil
	}
	matchingTerms, err := getMatchingAntiAffinityTerms(pod, nodeInfoMap)
	if err != nil {
		return nil
	}
	return &predicateMetadata{
		podBestEffort:             isPodBestEffort(pod),
		podRequest:                getResourceRequest(pod),
		podPorts:                  getUsedPorts(pod),
		matchingAntiAffinityTerms: matchingTerms,
	}
}

func isVolumeConflict(volume api.Volume, pod *api.Pod) bool {
	// fast path if there is no conflict checking targets.
	if volume.GCEPersistentDisk == nil && volume.AWSElasticBlockStore == nil && volume.RBD == nil {
		return false
	}

	for _, existingVolume := range pod.Spec.Volumes {
		// Same GCE disk mounted by multiple pods conflicts unless all pods mount it read-only.
		if volume.GCEPersistentDisk != nil && existingVolume.GCEPersistentDisk != nil {
			disk, existingDisk := volume.GCEPersistentDisk, existingVolume.GCEPersistentDisk
			if disk.PDName == existingDisk.PDName && !(disk.ReadOnly && existingDisk.ReadOnly) {
				return true
			}
		}

		if volume.AWSElasticBlockStore != nil && existingVolume.AWSElasticBlockStore != nil {
			if volume.AWSElasticBlockStore.VolumeID == existingVolume.AWSElasticBlockStore.VolumeID {
				return true
			}
		}

		if volume.RBD != nil && existingVolume.RBD != nil {
			mon, pool, image := volume.RBD.CephMonitors, volume.RBD.RBDPool, volume.RBD.RBDImage
			emon, epool, eimage := existingVolume.RBD.CephMonitors, existingVolume.RBD.RBDPool, existingVolume.RBD.RBDImage
			// two RBDs images are the same if they share the same Ceph monitor, are in the same RADOS Pool, and have the same image name
			// only one read-write mount is permitted for the same RBD image.
			// same RBD image mounted by multiple Pods conflicts unless all Pods mount the image read-only
			if haveSame(mon, emon) && pool == epool && image == eimage && !(volume.RBD.ReadOnly && existingVolume.RBD.ReadOnly) {
				return true
			}
		}
	}

	return false
}

// NoDiskConflict evaluates if a pod can fit due to the volumes it requests, and those that
// are already mounted. If there is already a volume mounted on that node, another pod that uses the same volume
// can't be scheduled there.
// This is GCE, Amazon EBS, and Ceph RBD specific for now:
// - GCE PD allows multiple mounts as long as they're all read-only
// - AWS EBS forbids any two pods mounting the same volume ID
// - Ceph RBD forbids if any two pods share at least same monitor, and match pool and image.
// TODO: migrate this into some per-volume specific code?
func NoDiskConflict(pod *api.Pod, meta interface{}, nodeInfo *schedulercache.NodeInfo) (bool, []algorithm.PredicateFailureReason, error) {
	for _, v := range pod.Spec.Volumes {
		for _, ev := range nodeInfo.Pods() {
			if isVolumeConflict(v, ev) {
				return false, []algorithm.PredicateFailureReason{ErrDiskConflict}, nil
			}
		}
	}
	return true, nil, nil
}

type MaxPDVolumeCountChecker struct {
	filter     VolumeFilter
	maxVolumes int
	pvInfo     PersistentVolumeInfo
	pvcInfo    PersistentVolumeClaimInfo
}

// VolumeFilter contains information on how to filter PD Volumes when checking PD Volume caps
type VolumeFilter struct {
	// Filter normal volumes
	FilterVolume           func(vol *api.Volume) (id string, relevant bool)
	FilterPersistentVolume func(pv *api.PersistentVolume) (id string, relevant bool)
}

// NewMaxPDVolumeCountPredicate creates a predicate which evaluates whether a pod can fit based on the
// number of volumes which match a filter that it requests, and those that are already present.  The
// maximum number is configurable to accommodate different systems.
//
// The predicate looks for both volumes used directly, as well as PVC volumes that are backed by relevant volume
// types, counts the number of unique volumes, and rejects the new pod if it would place the total count over
// the maximum.
func NewMaxPDVolumeCountPredicate(filter VolumeFilter, maxVolumes int, pvInfo PersistentVolumeInfo, pvcInfo PersistentVolumeClaimInfo) algorithm.FitPredicate {
	c := &MaxPDVolumeCountChecker{
		filter:     filter,
		maxVolumes: maxVolumes,
		pvInfo:     pvInfo,
		pvcInfo:    pvcInfo,
	}

	return c.predicate
}

func (c *MaxPDVolumeCountChecker) filterVolumes(volumes []api.Volume, namespace string, filteredVolumes map[string]bool) error {
	for _, vol := range volumes {
		if id, ok := c.filter.FilterVolume(&vol); ok {
			filteredVolumes[id] = true
		} else if vol.PersistentVolumeClaim != nil {
			pvcName := vol.PersistentVolumeClaim.ClaimName
			if pvcName == "" {
				return fmt.Errorf("PersistentVolumeClaim had no name")
			}
			pvc, err := c.pvcInfo.GetPersistentVolumeClaimInfo(namespace, pvcName)
			if err != nil {
				// if the PVC is not found, log the error and count the PV towards the PV limit
				// generate a random volume ID since its required for de-dup
				utilruntime.HandleError(fmt.Errorf("Unable to look up PVC info for %s/%s, assuming PVC matches predicate when counting limits: %v", namespace, pvcName, err))
				source := rand.NewSource(time.Now().UnixNano())
				generatedID := "missingPVC" + strconv.Itoa(rand.New(source).Intn(1000000))
				filteredVolumes[generatedID] = true
				return nil
			}

			if pvc == nil {
				return fmt.Errorf("PersistentVolumeClaim not found: %q", pvcName)
			}

			pvName := pvc.Spec.VolumeName
			if pvName == "" {
				return fmt.Errorf("PersistentVolumeClaim is not bound: %q", pvcName)
			}

			pv, err := c.pvInfo.GetPersistentVolumeInfo(pvName)
			if err != nil {
				// if the PV is not found, log the error
				// and count the PV towards the PV limit
				// generate a random volume ID since its required for de-dup
				utilruntime.HandleError(fmt.Errorf("Unable to look up PV info for %s/%s/%s, assuming PV matches predicate when counting limits: %v", namespace, pvcName, pvName, err))
				source := rand.NewSource(time.Now().UnixNano())
				generatedID := "missingPV" + strconv.Itoa(rand.New(source).Intn(1000000))
				filteredVolumes[generatedID] = true
				return nil
			}

			if pv == nil {
				return fmt.Errorf("PersistentVolume not found: %q", pvName)
			}

			if id, ok := c.filter.FilterPersistentVolume(pv); ok {
				filteredVolumes[id] = true
			}
		}
	}

	return nil
}

func (c *MaxPDVolumeCountChecker) predicate(pod *api.Pod, meta interface{}, nodeInfo *schedulercache.NodeInfo) (bool, []algorithm.PredicateFailureReason, error) {
	// If a pod doesn't have any volume attached to it, the predicate will always be true.
	// Thus we make a fast path for it, to avoid unnecessary computations in this case.
	if len(pod.Spec.Volumes) == 0 {
		return true, nil, nil
	}

	newVolumes := make(map[string]bool)
	if err := c.filterVolumes(pod.Spec.Volumes, pod.Namespace, newVolumes); err != nil {
		return false, nil, err
	}

	// quick return
	if len(newVolumes) == 0 {
		return true, nil, nil
	}

	// count unique volumes
	existingVolumes := make(map[string]bool)
	for _, existingPod := range nodeInfo.Pods() {
		if err := c.filterVolumes(existingPod.Spec.Volumes, existingPod.Namespace, existingVolumes); err != nil {
			return false, nil, err
		}
	}
	numExistingVolumes := len(existingVolumes)

	// filter out already-mounted volumes
	for k := range existingVolumes {
		if _, ok := newVolumes[k]; ok {
			delete(newVolumes, k)
		}
	}

	numNewVolumes := len(newVolumes)

	if numExistingVolumes+numNewVolumes > c.maxVolumes {
		// violates MaxEBSVolumeCount or MaxGCEPDVolumeCount
		return false, []algorithm.PredicateFailureReason{ErrMaxVolumeCountExceeded}, nil
	}

	return true, nil, nil
}

// EBSVolumeFilter is a VolumeFilter for filtering AWS ElasticBlockStore Volumes
var EBSVolumeFilter VolumeFilter = VolumeFilter{
	FilterVolume: func(vol *api.Volume) (string, bool) {
		if vol.AWSElasticBlockStore != nil {
			return vol.AWSElasticBlockStore.VolumeID, true
		}
		return "", false
	},

	FilterPersistentVolume: func(pv *api.PersistentVolume) (string, bool) {
		if pv.Spec.AWSElasticBlockStore != nil {
			return pv.Spec.AWSElasticBlockStore.VolumeID, true
		}
		return "", false
	},
}

// GCEPDVolumeFilter is a VolumeFilter for filtering GCE PersistentDisk Volumes
var GCEPDVolumeFilter VolumeFilter = VolumeFilter{
	FilterVolume: func(vol *api.Volume) (string, bool) {
		if vol.GCEPersistentDisk != nil {
			return vol.GCEPersistentDisk.PDName, true
		}
		return "", false
	},

	FilterPersistentVolume: func(pv *api.PersistentVolume) (string, bool) {
		if pv.Spec.GCEPersistentDisk != nil {
			return pv.Spec.GCEPersistentDisk.PDName, true
		}
		return "", false
	},
}

type VolumeZoneChecker struct {
	pvInfo  PersistentVolumeInfo
	pvcInfo PersistentVolumeClaimInfo
}

// VolumeZonePredicate evaluates if a pod can fit due to the volumes it requests, given
// that some volumes may have zone scheduling constraints.  The requirement is that any
// volume zone-labels must match the equivalent zone-labels on the node.  It is OK for
// the node to have more zone-label constraints (for example, a hypothetical replicated
// volume might allow region-wide access)
//
// Currently this is only supported with PersistentVolumeClaims, and looks to the labels
// only on the bound PersistentVolume.
//
// Working with volumes declared inline in the pod specification (i.e. not
// using a PersistentVolume) is likely to be harder, as it would require
// determining the zone of a volume during scheduling, and that is likely to
// require calling out to the cloud provider.  It seems that we are moving away
// from inline volume declarations anyway.
func NewVolumeZonePredicate(pvInfo PersistentVolumeInfo, pvcInfo PersistentVolumeClaimInfo) algorithm.FitPredicate {
	c := &VolumeZoneChecker{
		pvInfo:  pvInfo,
		pvcInfo: pvcInfo,
	}
	return c.predicate
}

func (c *VolumeZoneChecker) predicate(pod *api.Pod, meta interface{}, nodeInfo *schedulercache.NodeInfo) (bool, []algorithm.PredicateFailureReason, error) {
	// If a pod doesn't have any volume attached to it, the predicate will always be true.
	// Thus we make a fast path for it, to avoid unnecessary computations in this case.
	if len(pod.Spec.Volumes) == 0 {
		return true, nil, nil
	}

	node := nodeInfo.Node()
	if node == nil {
		return false, nil, fmt.Errorf("node not found")
	}

	nodeConstraints := make(map[string]string)
	for k, v := range node.ObjectMeta.Labels {
		if k != unversioned.LabelZoneFailureDomain && k != unversioned.LabelZoneRegion {
			continue
		}
		nodeConstraints[k] = v
	}

	if len(nodeConstraints) == 0 {
		// The node has no zone constraints, so we're OK to schedule.
		// In practice, when using zones, all nodes must be labeled with zone labels.
		// We want to fast-path this case though.
		return true, nil, nil
	}

	namespace := pod.Namespace
	manifest := &(pod.Spec)
	for i := range manifest.Volumes {
		volume := &manifest.Volumes[i]
		if volume.PersistentVolumeClaim != nil {
			pvcName := volume.PersistentVolumeClaim.ClaimName
			if pvcName == "" {
				return false, nil, fmt.Errorf("PersistentVolumeClaim had no name")
			}
			pvc, err := c.pvcInfo.GetPersistentVolumeClaimInfo(namespace, pvcName)
			if err != nil {
				return false, nil, err
			}

			if pvc == nil {
				return false, nil, fmt.Errorf("PersistentVolumeClaim was not found: %q", pvcName)
			}

			pvName := pvc.Spec.VolumeName
			if pvName == "" {
				return false, nil, fmt.Errorf("PersistentVolumeClaim is not bound: %q", pvcName)
			}

			pv, err := c.pvInfo.GetPersistentVolumeInfo(pvName)
			if err != nil {
				return false, nil, err
			}

			if pv == nil {
				return false, nil, fmt.Errorf("PersistentVolume not found: %q", pvName)
			}

			for k, v := range pv.ObjectMeta.Labels {
				if k != unversioned.LabelZoneFailureDomain && k != unversioned.LabelZoneRegion {
					continue
				}
				nodeV, _ := nodeConstraints[k]
				if v != nodeV {
					glog.V(2).Infof("Won't schedule pod %q onto node %q due to volume %q (mismatch on %q)", pod.Name, node.Name, pvName, k)
					return false, []algorithm.PredicateFailureReason{ErrVolumeZoneConflict}, nil
				}
			}
		}
	}

	return true, nil, nil
}

func getResourceRequest(pod *api.Pod) *schedulercache.Resource {
	result := schedulercache.Resource{}
	for _, container := range pod.Spec.Containers {
		requests := container.Resources.Requests
		result.Memory += requests.Memory().Value()
		result.MilliCPU += requests.Cpu().MilliValue()
		result.NvidiaGPU += requests.NvidiaGPU().Value()
	}
	// take max_resource(sum_pod, any_init_container)
	for _, container := range pod.Spec.InitContainers {
		requests := container.Resources.Requests
		if mem := requests.Memory().Value(); mem > result.Memory {
			result.Memory = mem
		}
		if cpu := requests.Cpu().MilliValue(); cpu > result.MilliCPU {
			result.MilliCPU = cpu
		}
	}
	return &result
}

func podName(pod *api.Pod) string {
	return pod.Namespace + "/" + pod.Name
}

func PodFitsResources(pod *api.Pod, meta interface{}, nodeInfo *schedulercache.NodeInfo) (bool, []algorithm.PredicateFailureReason, error) {
	node := nodeInfo.Node()
	if node == nil {
		return false, nil, fmt.Errorf("node not found")
	}

	var predicateFails []algorithm.PredicateFailureReason
	allowedPodNumber := nodeInfo.AllowedPodNumber()
	if len(nodeInfo.Pods())+1 > allowedPodNumber {
		predicateFails = append(predicateFails, NewInsufficientResourceError(api.ResourcePods, 1, int64(len(nodeInfo.Pods())), int64(allowedPodNumber)))
	}

	var podRequest *schedulercache.Resource
	if predicateMeta, ok := meta.(*predicateMetadata); ok {
		podRequest = predicateMeta.podRequest
	} else {
		// We couldn't parse metadata - fallback to computing it.
		podRequest = getResourceRequest(pod)
	}
	if podRequest.MilliCPU == 0 && podRequest.Memory == 0 && podRequest.NvidiaGPU == 0 {
		return len(predicateFails) == 0, predicateFails, nil
	}

	allocatable := nodeInfo.AllocatableResource()
	if allocatable.MilliCPU < podRequest.MilliCPU+nodeInfo.RequestedResource().MilliCPU {
		predicateFails = append(predicateFails, NewInsufficientResourceError(api.ResourceCPU, podRequest.MilliCPU, nodeInfo.RequestedResource().MilliCPU, allocatable.MilliCPU))
	}
	if allocatable.Memory < podRequest.Memory+nodeInfo.RequestedResource().Memory {
		predicateFails = append(predicateFails, NewInsufficientResourceError(api.ResourceMemory, podRequest.Memory, nodeInfo.RequestedResource().Memory, allocatable.Memory))
	}
	if allocatable.NvidiaGPU < podRequest.NvidiaGPU+nodeInfo.RequestedResource().NvidiaGPU {
		predicateFails = append(predicateFails, NewInsufficientResourceError(api.ResourceNvidiaGPU, podRequest.NvidiaGPU, nodeInfo.RequestedResource().NvidiaGPU, allocatable.NvidiaGPU))
	}
	if glog.V(10) {
		// We explicitly don't do glog.V(10).Infof() to avoid computing all the parameters if this is
		// not logged. There is visible performance gain from it.
		glog.Infof("Schedule Pod %+v on Node %+v is allowed, Node is running only %v out of %v Pods.",
			podName(pod), node.Name, len(nodeInfo.Pods()), allowedPodNumber)
	}
	return len(predicateFails) == 0, predicateFails, nil
}

// nodeMatchesNodeSelectorTerms checks if a node's labels satisfy a list of node selector terms,
// terms are ORed, and an empty list of terms will match nothing.
func nodeMatchesNodeSelectorTerms(node *api.Node, nodeSelectorTerms []api.NodeSelectorTerm) bool {
	for _, req := range nodeSelectorTerms {
		nodeSelector, err := api.NodeSelectorRequirementsAsSelector(req.MatchExpressions)
		if err != nil {
			glog.V(10).Infof("Failed to parse MatchExpressions: %+v, regarding as not match.", req.MatchExpressions)
			return false
		}
		if nodeSelector.Matches(labels.Set(node.Labels)) {
			return true
		}
	}
	return false
}

// The pod can only schedule onto nodes that satisfy requirements in both NodeAffinity and nodeSelector.
func podMatchesNodeLabels(pod *api.Pod, node *api.Node) bool {
	// Check if node.Labels match pod.Spec.NodeSelector.
	if len(pod.Spec.NodeSelector) > 0 {
		selector := labels.SelectorFromSet(pod.Spec.NodeSelector)
		if !selector.Matches(labels.Set(node.Labels)) {
			return false
		}
	}

	// Parse required node affinity scheduling requirements
	// and check if the current node match the requirements.
	affinity, err := api.GetAffinityFromPodAnnotations(pod.Annotations)
	if err != nil {
		glog.V(10).Infof("Failed to get Affinity from Pod %+v, err: %+v", podName(pod), err)
		return false
	}

	// 1. nil NodeSelector matches all nodes (i.e. does not filter out any nodes)
	// 2. nil []NodeSelectorTerm (equivalent to non-nil empty NodeSelector) matches no nodes
	// 3. zero-length non-nil []NodeSelectorTerm matches no nodes also, just for simplicity
	// 4. nil []NodeSelectorRequirement (equivalent to non-nil empty NodeSelectorTerm) matches no nodes
	// 5. zero-length non-nil []NodeSelectorRequirement matches no nodes also, just for simplicity
	// 6. non-nil empty NodeSelectorRequirement is not allowed
	nodeAffinityMatches := true
	if affinity != nil && affinity.NodeAffinity != nil {
		nodeAffinity := affinity.NodeAffinity
		// if no required NodeAffinity requirements, will do no-op, means select all nodes.
		// TODO: Replace next line with subsequent commented-out line when implement RequiredDuringSchedulingRequiredDuringExecution.
		if nodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution == nil {
			// if nodeAffinity.RequiredDuringSchedulingRequiredDuringExecution == nil && nodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution == nil {
			return true
		}

		// Match node selector for requiredDuringSchedulingRequiredDuringExecution.
		// TODO: Uncomment this block when implement RequiredDuringSchedulingRequiredDuringExecution.
		// if nodeAffinity.RequiredDuringSchedulingRequiredDuringExecution != nil {
		// 	nodeSelectorTerms := nodeAffinity.RequiredDuringSchedulingRequiredDuringExecution.NodeSelectorTerms
		// 	glog.V(10).Infof("Match for RequiredDuringSchedulingRequiredDuringExecution node selector terms %+v", nodeSelectorTerms)
		// 	nodeAffinityMatches = nodeMatchesNodeSelectorTerms(node, nodeSelectorTerms)
		// }

		// Match node selector for requiredDuringSchedulingIgnoredDuringExecution.
		if nodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution != nil {
			nodeSelectorTerms := nodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms
			glog.V(10).Infof("Match for RequiredDuringSchedulingIgnoredDuringExecution node selector terms %+v", nodeSelectorTerms)
			nodeAffinityMatches = nodeAffinityMatches && nodeMatchesNodeSelectorTerms(node, nodeSelectorTerms)
		}

	}
	return nodeAffinityMatches
}

func PodSelectorMatches(pod *api.Pod, meta interface{}, nodeInfo *schedulercache.NodeInfo) (bool, []algorithm.PredicateFailureReason, error) {
	node := nodeInfo.Node()
	if node == nil {
		return false, nil, fmt.Errorf("node not found")
	}
	if podMatchesNodeLabels(pod, node) {
		return true, nil, nil
	}
	return false, []algorithm.PredicateFailureReason{ErrNodeSelectorNotMatch}, nil
}

func PodFitsHost(pod *api.Pod, meta interface{}, nodeInfo *schedulercache.NodeInfo) (bool, []algorithm.PredicateFailureReason, error) {
	if len(pod.Spec.NodeName) == 0 {
		return true, nil, nil
	}
	node := nodeInfo.Node()
	if node == nil {
		return false, nil, fmt.Errorf("node not found")
	}
	if pod.Spec.NodeName == node.Name {
		return true, nil, nil
	}
	return false, []algorithm.PredicateFailureReason{ErrPodNotMatchHostName}, nil
}

type NodeLabelChecker struct {
	labels   []string
	presence bool
}

func NewNodeLabelPredicate(labels []string, presence bool) algorithm.FitPredicate {
	labelChecker := &NodeLabelChecker{
		labels:   labels,
		presence: presence,
	}
	return labelChecker.CheckNodeLabelPresence
}

// CheckNodeLabelPresence checks whether all of the specified labels exists on a node or not, regardless of their value
// If "presence" is false, then returns false if any of the requested labels matches any of the node's labels,
// otherwise returns true.
// If "presence" is true, then returns false if any of the requested labels does not match any of the node's labels,
// otherwise returns true.
//
// Consider the cases where the nodes are placed in regions/zones/racks and these are identified by labels
// In some cases, it is required that only nodes that are part of ANY of the defined regions/zones/racks be selected
//
// Alternately, eliminating nodes that have a certain label, regardless of value, is also useful
// A node may have a label with "retiring" as key and the date as the value
// and it may be desirable to avoid scheduling new pods on this node
func (n *NodeLabelChecker) CheckNodeLabelPresence(pod *api.Pod, meta interface{}, nodeInfo *schedulercache.NodeInfo) (bool, []algorithm.PredicateFailureReason, error) {
	node := nodeInfo.Node()
	if node == nil {
		return false, nil, fmt.Errorf("node not found")
	}

	var exists bool
	nodeLabels := labels.Set(node.Labels)
	for _, label := range n.labels {
		exists = nodeLabels.Has(label)
		if (exists && !n.presence) || (!exists && n.presence) {
			return false, []algorithm.PredicateFailureReason{ErrNodeLabelPresenceViolated}, nil
		}
	}
	return true, nil, nil
}

type ServiceAffinity struct {
	podLister     algorithm.PodLister
	serviceLister algorithm.ServiceLister
	nodeInfo      NodeInfo
	labels        []string
}

func NewServiceAffinityPredicate(podLister algorithm.PodLister, serviceLister algorithm.ServiceLister, nodeInfo NodeInfo, labels []string) algorithm.FitPredicate {
	affinity := &ServiceAffinity{
		podLister:     podLister,
		serviceLister: serviceLister,
		nodeInfo:      nodeInfo,
		labels:        labels,
	}
	return affinity.CheckServiceAffinity
}

// CheckServiceAffinity ensures that only the nodes that match the specified labels are considered for scheduling.
// The set of labels to be considered are provided to the struct (ServiceAffinity).
// The pod is checked for the labels and any missing labels are then checked in the node
// that hosts the service pods (peers) for the given pod.
//
// We add an implicit selector requiring some particular value V for label L to a pod, if:
// - L is listed in the ServiceAffinity object that is passed into the function
// - the pod does not have any NodeSelector for L
// - some other pod from the same service is already scheduled onto a node that has value V for label L
func (s *ServiceAffinity) CheckServiceAffinity(pod *api.Pod, meta interface{}, nodeInfo *schedulercache.NodeInfo) (bool, []algorithm.PredicateFailureReason, error) {
	node := nodeInfo.Node()
	if node == nil {
		return false, nil, fmt.Errorf("node not found")
	}

	var affinitySelector labels.Selector

	// check if the pod being scheduled has the affinity labels specified in its NodeSelector
	affinityLabels := map[string]string{}
	nodeSelector := labels.Set(pod.Spec.NodeSelector)
	labelsExist := true
	for _, l := range s.labels {
		if nodeSelector.Has(l) {
			affinityLabels[l] = nodeSelector.Get(l)
		} else {
			// the current pod does not specify all the labels, look in the existing service pods
			labelsExist = false
		}
	}

	// skip looking at other pods in the service if the current pod defines all the required affinity labels
	if !labelsExist {
		services, err := s.serviceLister.GetPodServices(pod)
		if err == nil {
			// just use the first service and get the other pods within the service
			// TODO: a separate predicate can be created that tries to handle all services for the pod
			selector := labels.SelectorFromSet(services[0].Spec.Selector)
			servicePods, err := s.podLister.List(selector)
			if err != nil {
				return false, nil, err
			}
			// consider only the pods that belong to the same namespace
			nsServicePods := []*api.Pod{}
			for _, nsPod := range servicePods {
				if nsPod.Namespace == pod.Namespace {
					nsServicePods = append(nsServicePods, nsPod)
				}
			}
			if len(nsServicePods) > 0 {
				// consider any service pod and fetch the node its hosted on
				otherNode, err := s.nodeInfo.GetNodeInfo(nsServicePods[0].Spec.NodeName)
				if err != nil {
					return false, nil, err
				}
				for _, l := range s.labels {
					// If the pod being scheduled has the label value specified, do not override it
					if _, exists := affinityLabels[l]; exists {
						continue
					}
					if labels.Set(otherNode.Labels).Has(l) {
						affinityLabels[l] = labels.Set(otherNode.Labels).Get(l)
					}
				}
			}
		}
	}

	// if there are no existing pods in the service, consider all nodes
	if len(affinityLabels) == 0 {
		affinitySelector = labels.Everything()
	} else {
		affinitySelector = labels.Set(affinityLabels).AsSelector()
	}

	// check if the node matches the selector
	if affinitySelector.Matches(labels.Set(node.Labels)) {
		return true, nil, nil
	}
	return false, []algorithm.PredicateFailureReason{ErrServiceAffinityViolated}, nil
}

func PodFitsHostPorts(pod *api.Pod, meta interface{}, nodeInfo *schedulercache.NodeInfo) (bool, []algorithm.PredicateFailureReason, error) {
	var wantPorts map[int]bool
	if predicateMeta, ok := meta.(*predicateMetadata); ok {
		wantPorts = predicateMeta.podPorts
	} else {
		// We couldn't parse metadata - fallback to computing it.
		wantPorts = getUsedPorts(pod)
	}
	if len(wantPorts) == 0 {
		return true, nil, nil
	}

	// TODO: Aggregate it at the NodeInfo level.
	existingPorts := getUsedPorts(nodeInfo.Pods()...)
	for wport := range wantPorts {
		if wport != 0 && existingPorts[wport] {
			return false, []algorithm.PredicateFailureReason{ErrPodNotFitsHostPorts}, nil
		}
	}
	return true, nil, nil
}

func getUsedPorts(pods ...*api.Pod) map[int]bool {
	ports := make(map[int]bool)
	for _, pod := range pods {
		for j := range pod.Spec.Containers {
			container := &pod.Spec.Containers[j]
			for k := range container.Ports {
				podPort := &container.Ports[k]
				// "0" is explicitly ignored in PodFitsHostPorts,
				// which is the only function that uses this value.
				if podPort.HostPort != 0 {
					ports[int(podPort.HostPort)] = true
				}
			}
		}
	}
	return ports
}

// search two arrays and return true if they have at least one common element; return false otherwise
func haveSame(a1, a2 []string) bool {
	for _, val1 := range a1 {
		for _, val2 := range a2 {
			if val1 == val2 {
				return true
			}
		}
	}
	return false
}

func GeneralPredicates(pod *api.Pod, meta interface{}, nodeInfo *schedulercache.NodeInfo) (bool, []algorithm.PredicateFailureReason, error) {
	var predicateFails []algorithm.PredicateFailureReason
	fit, reasons, err := PodFitsResources(pod, meta, nodeInfo)
	if err != nil {
		return false, predicateFails, err
	}
	if !fit {
		predicateFails = append(predicateFails, reasons...)
	}

	fit, reasons, err = PodFitsHost(pod, meta, nodeInfo)
	if err != nil {
		return false, predicateFails, err
	}
	if !fit {
		predicateFails = append(predicateFails, reasons...)
	}

	fit, reasons, err = PodFitsHostPorts(pod, meta, nodeInfo)
	if err != nil {
		return false, predicateFails, err
	}
	if !fit {
		predicateFails = append(predicateFails, reasons...)
	}

	fit, reasons, err = PodSelectorMatches(pod, meta, nodeInfo)
	if err != nil {
		return false, predicateFails, err
	}
	if !fit {
		predicateFails = append(predicateFails, reasons...)
	}

	return len(predicateFails) == 0, predicateFails, nil
}

type PodAffinityChecker struct {
	info           NodeInfo
	podLister      algorithm.PodLister
	failureDomains priorityutil.Topologies
}

func NewPodAffinityPredicate(info NodeInfo, podLister algorithm.PodLister, failureDomains []string) algorithm.FitPredicate {
	checker := &PodAffinityChecker{
		info:           info,
		podLister:      podLister,
		failureDomains: priorityutil.Topologies{DefaultKeys: failureDomains},
	}
	return checker.InterPodAffinityMatches
}

func (c *PodAffinityChecker) InterPodAffinityMatches(pod *api.Pod, meta interface{}, nodeInfo *schedulercache.NodeInfo) (bool, []algorithm.PredicateFailureReason, error) {
	node := nodeInfo.Node()
	if node == nil {
		return false, nil, fmt.Errorf("node not found")
	}
	if !c.satisfiesExistingPodsAntiAffinity(pod, meta, node) {
		return false, []algorithm.PredicateFailureReason{ErrPodAffinityNotMatch}, nil
	}

	// Now check if <pod> requirements will be satisfied on this node.
	affinity, err := api.GetAffinityFromPodAnnotations(pod.Annotations)
	if err != nil {
		return false, nil, err
	}
	if affinity == nil || (affinity.PodAffinity == nil && affinity.PodAntiAffinity == nil) {
		return true, nil, nil
	}
	if !c.satisfiesPodsAffinityAntiAffinity(pod, node, affinity) {
		return false, []algorithm.PredicateFailureReason{ErrPodAffinityNotMatch}, nil
	}

	if glog.V(10) {
		// We explicitly don't do glog.V(10).Infof() to avoid computing all the parameters if this is
		// not logged. There is visible performance gain from it.
		glog.Infof("Schedule Pod %+v on Node %+v is allowed, pod (anti)affinity constraints satisfied",
			podName(pod), node.Name)
	}
	return true, nil, nil
}

// AnyPodMatchesPodAffinityTerm checks if any of given pods can match the specific podAffinityTerm.
// First return value indicates whether a matching pod exists on a node that matches the topology key,
// while the second return value indicates whether a matching pod exists anywhere.
// TODO: Do we really need any pod matching, or all pods matching? I think the latter.
func (c *PodAffinityChecker) anyPodMatchesPodAffinityTerm(pod *api.Pod, allPods []*api.Pod, node *api.Node, term *api.PodAffinityTerm) (bool, bool, error) {
	matchingPodExists := false
	for _, existingPod := range allPods {
		match, err := priorityutil.PodMatchesTermsNamespaceAndSelector(existingPod, pod, term)
		if err != nil {
			return false, matchingPodExists, err
		}
		if match {
			matchingPodExists = true
			existingPodNode, err := c.info.GetNodeInfo(existingPod.Spec.NodeName)
			if err != nil {
				return false, matchingPodExists, err
			}
			if c.failureDomains.NodesHaveSameTopologyKey(node, existingPodNode, term.TopologyKey) {
				return true, matchingPodExists, nil
			}
		}
	}
	return false, matchingPodExists, nil
}

func getPodAffinityTerms(podAffinity *api.PodAffinity) (terms []api.PodAffinityTerm) {
	if podAffinity != nil {
		if len(podAffinity.RequiredDuringSchedulingIgnoredDuringExecution) != 0 {
			terms = podAffinity.RequiredDuringSchedulingIgnoredDuringExecution
		}
		// TODO: Uncomment this block when implement RequiredDuringSchedulingRequiredDuringExecution.
		//if len(podAffinity.RequiredDuringSchedulingRequiredDuringExecution) != 0 {
		//	terms = append(terms, podAffinity.RequiredDuringSchedulingRequiredDuringExecution...)
		//}
	}
	return terms
}

func getPodAntiAffinityTerms(podAntiAffinity *api.PodAntiAffinity) (terms []api.PodAffinityTerm) {
	if podAntiAffinity != nil {
		if len(podAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution) != 0 {
			terms = podAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution
		}
		// TODO: Uncomment this block when implement RequiredDuringSchedulingRequiredDuringExecution.
		//if len(podAntiAffinity.RequiredDuringSchedulingRequiredDuringExecution) != 0 {
		//	terms = append(terms, podAntiAffinity.RequiredDuringSchedulingRequiredDuringExecution...)
		//}
	}
	return terms
}

func getMatchingAntiAffinityTerms(pod *api.Pod, nodeInfoMap map[string]*schedulercache.NodeInfo) ([]matchingPodAntiAffinityTerm, error) {
	allNodeNames := make([]string, 0, len(nodeInfoMap))
	for name := range nodeInfoMap {
		allNodeNames = append(allNodeNames, name)
	}

	var lock sync.Mutex
	var result []matchingPodAntiAffinityTerm
	var firstError error
	appendResult := func(toAppend []matchingPodAntiAffinityTerm) {
		lock.Lock()
		defer lock.Unlock()
		result = append(result, toAppend...)
	}
	catchError := func(err error) {
		lock.Lock()
		defer lock.Unlock()
		if firstError == nil {
			firstError = err
		}
	}

	processNode := func(i int) {
		nodeInfo := nodeInfoMap[allNodeNames[i]]
		node := nodeInfo.Node()
		if node == nil {
			catchError(fmt.Errorf("node not found"))
			return
		}
		var nodeResult []matchingPodAntiAffinityTerm
		for _, existingPod := range nodeInfo.PodsWithAffinity() {
			affinity, err := api.GetAffinityFromPodAnnotations(existingPod.Annotations)
			if err != nil {
				catchError(err)
				return
			}
			if affinity == nil {
				continue
			}
			for _, term := range getPodAntiAffinityTerms(affinity.PodAntiAffinity) {
				match, err := priorityutil.PodMatchesTermsNamespaceAndSelector(pod, existingPod, &term)
				if err != nil {
					catchError(err)
					return
				}
				if match {
					nodeResult = append(nodeResult, matchingPodAntiAffinityTerm{term: &term, node: node})
				}
			}
		}
		if len(nodeResult) > 0 {
			appendResult(nodeResult)
		}
	}
	workqueue.Parallelize(16, len(allNodeNames), processNode)
	return result, firstError
}

func (c *PodAffinityChecker) getMatchingAntiAffinityTerms(pod *api.Pod, allPods []*api.Pod) ([]matchingPodAntiAffinityTerm, error) {
	var result []matchingPodAntiAffinityTerm
	for _, existingPod := range allPods {
		affinity, err := api.GetAffinityFromPodAnnotations(existingPod.Annotations)
		if err != nil {
			return nil, err
		}
		if affinity.PodAntiAffinity != nil {
			existingPodNode, err := c.info.GetNodeInfo(existingPod.Spec.NodeName)
			if err != nil {
				return nil, err
			}
			for _, term := range getPodAntiAffinityTerms(affinity.PodAntiAffinity) {
				match, err := priorityutil.PodMatchesTermsNamespaceAndSelector(pod, existingPod, &term)
				if err != nil {
					return nil, err
				}
				if match {
					result = append(result, matchingPodAntiAffinityTerm{term: &term, node: existingPodNode})
				}
			}
		}
	}
	return result, nil
}

// Checks if scheduling the pod onto this node would break any anti-affinity
// rules indicated by the existing pods.
func (c *PodAffinityChecker) satisfiesExistingPodsAntiAffinity(pod *api.Pod, meta interface{}, node *api.Node) bool {
	var matchingTerms []matchingPodAntiAffinityTerm
	if predicateMeta, ok := meta.(*predicateMetadata); ok {
		matchingTerms = predicateMeta.matchingAntiAffinityTerms
	} else {
		allPods, err := c.podLister.List(labels.Everything())
		if err != nil {
			glog.V(10).Infof("Failed to get all pods, %+v", err)
			return false
		}
		if matchingTerms, err = c.getMatchingAntiAffinityTerms(pod, allPods); err != nil {
			glog.V(10).Infof("Failed to get all terms that pod %+v matches, err: %+v", podName(pod), err)
			return false
		}
	}
	for _, term := range matchingTerms {
		if c.failureDomains.NodesHaveSameTopologyKey(node, term.node, term.term.TopologyKey) {
			glog.V(10).Infof("Cannot schedule pod %+v onto node %v,because of PodAntiAffinityTerm %v",
				podName(pod), node.Name, term.term)
			return false
		}
	}
	if glog.V(10) {
		// We explicitly don't do glog.V(10).Infof() to avoid computing all the parameters if this is
		// not logged. There is visible performance gain from it.
		glog.Infof("Schedule Pod %+v on Node %+v is allowed, existing pods anti-affinity rules satisfied.",
			podName(pod), node.Name)
	}
	return true
}

// Checks if scheduling the pod onto this node would break any rules of this pod.
func (c *PodAffinityChecker) satisfiesPodsAffinityAntiAffinity(pod *api.Pod, node *api.Node, affinity *api.Affinity) bool {
	allPods, err := c.podLister.List(labels.Everything())
	if err != nil {
		return false
	}

	// Check all affinity terms.
	for _, term := range getPodAffinityTerms(affinity.PodAffinity) {
		termMatches, matchingPodExists, err := c.anyPodMatchesPodAffinityTerm(pod, allPods, node, &term)
		if err != nil {
			glog.V(10).Infof("Cannot schedule pod %+v onto node %v,because of PodAffinityTerm %v, err: %v",
				podName(pod), node.Name, term, err)
			return false
		}
		if !termMatches {
			// If the requirement matches a pod's own labels ane namespace, and there are
			// no other such pods, then disregard the requirement. This is necessary to
			// not block forever because the first pod of the collection can't be scheduled.
			match, err := priorityutil.PodMatchesTermsNamespaceAndSelector(pod, pod, &term)
			if err != nil || !match || matchingPodExists {
				glog.V(10).Infof("Cannot schedule pod %+v onto node %v,because of PodAffinityTerm %v, err: %v",
					podName(pod), node.Name, term, err)
				return false
			}
		}
	}

	// Check all anti-affinity terms.
	for _, term := range getPodAntiAffinityTerms(affinity.PodAntiAffinity) {
		termMatches, _, err := c.anyPodMatchesPodAffinityTerm(pod, allPods, node, &term)
		if err != nil || termMatches {
			glog.V(10).Infof("Cannot schedule pod %+v onto node %v,because of PodAntiAffinityTerm %v, err: %v",
				podName(pod), node.Name, term, err)
			return false
		}
	}

	if glog.V(10) {
		// We explicitly don't do glog.V(10).Infof() to avoid computing all the parameters if this is
		// not logged. There is visible performance gain from it.
		glog.Infof("Schedule Pod %+v on Node %+v is allowed, pod afinnity/anti-affinity constraints satisfied.",
			podName(pod), node.Name)
	}
	return true
}

func PodToleratesNodeTaints(pod *api.Pod, meta interface{}, nodeInfo *schedulercache.NodeInfo) (bool, []algorithm.PredicateFailureReason, error) {
	node := nodeInfo.Node()
	if node == nil {
		return false, nil, fmt.Errorf("node not found")
	}

	taints, err := api.GetTaintsFromNodeAnnotations(node.Annotations)
	if err != nil {
		return false, nil, err
	}

	tolerations, err := api.GetTolerationsFromPodAnnotations(pod.Annotations)
	if err != nil {
		return false, nil, err
	}

	if tolerationsToleratesTaints(tolerations, taints) {
		return true, nil, nil
	}
	return false, []algorithm.PredicateFailureReason{ErrTaintsTolerationsNotMatch}, nil
}

func tolerationsToleratesTaints(tolerations []api.Toleration, taints []api.Taint) bool {
	// If the taint list is nil/empty, it is tolerated by all tolerations by default.
	if len(taints) == 0 {
		return true
	}

	// The taint list isn't nil/empty, a nil/empty toleration list can't tolerate them.
	if len(tolerations) == 0 {
		return false
	}

	for i := range taints {
		taint := &taints[i]
		// skip taints that have effect PreferNoSchedule, since it is for priorities
		if taint.Effect == api.TaintEffectPreferNoSchedule {
			continue
		}

		if !api.TaintToleratedByTolerations(taint, tolerations) {
			return false
		}
	}

	return true
}

// Determine if a pod is scheduled with best-effort QoS
func isPodBestEffort(pod *api.Pod) bool {
	return qos.GetPodQOS(pod) == qos.BestEffort
}

// CheckNodeMemoryPressurePredicate checks if a pod can be scheduled on a node
// reporting memory pressure condition.
func CheckNodeMemoryPressurePredicate(pod *api.Pod, meta interface{}, nodeInfo *schedulercache.NodeInfo) (bool, []algorithm.PredicateFailureReason, error) {
	node := nodeInfo.Node()
	if node == nil {
		return false, nil, fmt.Errorf("node not found")
	}

	var podBestEffort bool
	if predicateMeta, ok := meta.(*predicateMetadata); ok {
		podBestEffort = predicateMeta.podBestEffort
	} else {
		// We couldn't parse metadata - fallback to computing it.
		podBestEffort = isPodBestEffort(pod)
	}

	// pod is not BestEffort pod
	if !podBestEffort {
		return true, nil, nil
	}

	// is node under presure?
	for _, cond := range node.Status.Conditions {
		if cond.Type == api.NodeMemoryPressure && cond.Status == api.ConditionTrue {
			return false, []algorithm.PredicateFailureReason{ErrNodeUnderMemoryPressure}, nil
		}
	}

	return true, nil, nil
}

// CheckNodeDiskPressurePredicate checks if a pod can be scheduled on a node
// reporting disk pressure condition.
func CheckNodeDiskPressurePredicate(pod *api.Pod, meta interface{}, nodeInfo *schedulercache.NodeInfo) (bool, []algorithm.PredicateFailureReason, error) {
	node := nodeInfo.Node()
	if node == nil {
		return false, nil, fmt.Errorf("node not found")
	}

	// is node under presure?
	for _, cond := range node.Status.Conditions {
		if cond.Type == api.NodeDiskPressure && cond.Status == api.ConditionTrue {
			return false, []algorithm.PredicateFailureReason{ErrNodeUnderDiskPressure}, nil
		}
	}

	return true, nil, nil
}
