/*
Copyright 2016 The Kubernetes Authors.

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

// Package reconciler implements interfaces that attempt to reconcile the
// desired state of the with the actual state of the world by triggering
// relevant actions (attach, detach, mount, unmount).
package reconciler

import (
	"fmt"
	"io/ioutil"
	"path"
	"time"

	"github.com/golang/glog"
	"k8s.io/kubernetes/cmd/kubelet/app/options"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	"k8s.io/kubernetes/pkg/kubelet/config"
	"k8s.io/kubernetes/pkg/kubelet/volumemanager/cache"
	"k8s.io/kubernetes/pkg/types"
	"k8s.io/kubernetes/pkg/util/goroutinemap/exponentialbackoff"
	"k8s.io/kubernetes/pkg/util/mount"
	"k8s.io/kubernetes/pkg/util/strings"
	"k8s.io/kubernetes/pkg/util/wait"
	"k8s.io/kubernetes/pkg/volume"
	"k8s.io/kubernetes/pkg/volume/util/nestedpendingoperations"
	"k8s.io/kubernetes/pkg/volume/util/operationexecutor"
	volumetypes "k8s.io/kubernetes/pkg/volume/util/types"
	"k8s.io/kubernetes/pkg/volume/util/volumehelper"
)

// Reconciler runs a periodic loop to reconcile the desired state of the world
// with the actual state of the world by triggering attach, detach, mount, and
// unmount operations.
// Note: This is distinct from the Reconciler implemented by the attach/detach
// controller. This reconciles state for the kubelet volume manager. That
// reconciles state for the attach/detach controller.
type Reconciler interface {
	// Starts running the reconciliation loop which executes periodically, checks
	// if volumes that should be mounted are mounted and volumes that should
	// be unmounted are unmounted. If not, it will trigger mount/unmount
	// operations to rectify.
	// If attach/detach management is enabled, the manager will also check if
	// volumes that should be attached are attached and volumes that should
	// be detached are detached and trigger attach/detach operations as needed.
	Run(sourcesReady config.SourcesReady, stopCh <-chan struct{})
}

// NewReconciler returns a new instance of Reconciler.
//
// controllerAttachDetachEnabled - if true, indicates that the attach/detach
//   controller is responsible for managing the attach/detach operations for
//   this node, and therefore the volume manager should not
// loopSleepDuration - the amount of time the reconciler loop sleeps between
//   successive executions
//   reconstructDuration - the amount of time the reconstruct sleeps between
//   successive executions
// waitForAttachTimeout - the amount of time the Mount function will wait for
//   the volume to be attached
// hostName - the hostname for this node, used by Attach and Detach methods
// desiredStateOfWorld - cache containing the desired state of the world
// actualStateOfWorld - cache containing the actual state of the world
// operationExecutor - used to trigger attach/detach/mount/unmount operations
//   safely (prevents more than one operation from being triggered on the same
//   volume)
// mounter - mounter passed in from kubelet, passed down unmount path
// volumePluginMrg - volume plugin manager passed from kubelet
func NewReconciler(
	kubeClient internalclientset.Interface,
	controllerAttachDetachEnabled bool,
	loopSleepDuration time.Duration,
	reconstructDuration time.Duration,
	waitForAttachTimeout time.Duration,
	hostName string,
	desiredStateOfWorld cache.DesiredStateOfWorld,
	actualStateOfWorld cache.ActualStateOfWorld,
	operationExecutor operationexecutor.OperationExecutor,
	mounter mount.Interface,
	volumePluginMgr *volume.VolumePluginMgr,
	kubeletPodsDir string) Reconciler {
	return &reconciler{
		kubeClient:                    kubeClient,
		controllerAttachDetachEnabled: controllerAttachDetachEnabled,
		loopSleepDuration:             loopSleepDuration,
		reconstructDuration:           reconstructDuration,
		waitForAttachTimeout:          waitForAttachTimeout,
		hostName:                      hostName,
		desiredStateOfWorld:           desiredStateOfWorld,
		actualStateOfWorld:            actualStateOfWorld,
		operationExecutor:             operationExecutor,
		mounter:                       mounter,
		volumePluginMgr:               volumePluginMgr,
		kubeletPodsDir:                kubeletPodsDir,
		timeOfLastReconstruct:         time.Now(),
	}
}

type reconciler struct {
	kubeClient                    internalclientset.Interface
	controllerAttachDetachEnabled bool
	loopSleepDuration             time.Duration
	reconstructDuration           time.Duration
	waitForAttachTimeout          time.Duration
	hostName                      string
	desiredStateOfWorld           cache.DesiredStateOfWorld
	actualStateOfWorld            cache.ActualStateOfWorld
	operationExecutor             operationexecutor.OperationExecutor
	mounter                       mount.Interface
	volumePluginMgr               *volume.VolumePluginMgr
	kubeletPodsDir                string
	timeOfLastReconstruct         time.Time
}

func (rc *reconciler) Run(sourcesReady config.SourcesReady, stopCh <-chan struct{}) {
	wait.Until(rc.reconciliationLoopFunc(sourcesReady), rc.loopSleepDuration, stopCh)
}

func (rc *reconciler) reconciliationLoopFunc(sourcesReady config.SourcesReady) func() {
	return func() {
		rc.reconcile()

		// Add all sources ready check so that reconciler's reconstruct process will start after
		// desired state of world is populated with pod volume information from different sources. Otherwise,
		// reconciler's reconstruct process may add incomplete volume information and cause confusion.
		// In addition, if some sources are not ready, the reconstruct process may clean up pods' volumes
		// that are still in use because desired states could not get a complete list of pods.
		if sourcesReady.AllReady() && time.Since(rc.timeOfLastReconstruct) > rc.reconstructDuration {
			glog.V(5).Infof("Sources are all ready, starting reconstruct state function")
			rc.reconstruct()
		}
	}
}

func (rc *reconciler) reconcile() {
	// Unmounts are triggered before mounts so that a volume that was
	// referenced by a pod that was deleted and is now referenced by another
	// pod is unmounted from the first pod before being mounted to the new
	// pod.

	// Ensure volumes that should be unmounted are unmounted.
	for _, mountedVolume := range rc.actualStateOfWorld.GetMountedVolumes() {
		if !rc.desiredStateOfWorld.PodExistsInVolume(mountedVolume.PodName, mountedVolume.VolumeName) {
			// Volume is mounted, unmount it
			glog.V(12).Infof("Attempting to start UnmountVolume for volume %q (spec.Name: %q) from pod %q (UID: %q).",
				mountedVolume.VolumeName,
				mountedVolume.OuterVolumeSpecName,
				mountedVolume.PodName,
				mountedVolume.PodUID)
			err := rc.operationExecutor.UnmountVolume(
				mountedVolume.MountedVolume, rc.actualStateOfWorld)
			if err != nil &&
				!nestedpendingoperations.IsAlreadyExists(err) &&
				!exponentialbackoff.IsExponentialBackoff(err) {
				// Ignore nestedpendingoperations.IsAlreadyExists and exponentialbackoff.IsExponentialBackoff errors, they are expected.
				// Log all other errors.
				glog.Errorf(
					"operationExecutor.UnmountVolume failed for volume %q (spec.Name: %q) pod %q (UID: %q) controllerAttachDetachEnabled: %v with err: %v",
					mountedVolume.VolumeName,
					mountedVolume.OuterVolumeSpecName,
					mountedVolume.PodName,
					mountedVolume.PodUID,
					rc.controllerAttachDetachEnabled,
					err)
			}
			if err == nil {
				glog.Infof("UnmountVolume operation started for volume %q (spec.Name: %q) from pod %q (UID: %q).",
					mountedVolume.VolumeName,
					mountedVolume.OuterVolumeSpecName,
					mountedVolume.PodName,
					mountedVolume.PodUID)
			}
		}
	}

	// Ensure volumes that should be attached/mounted are attached/mounted.
	for _, volumeToMount := range rc.desiredStateOfWorld.GetVolumesToMount() {
		volMounted, devicePath, err := rc.actualStateOfWorld.PodExistsInVolume(volumeToMount.PodName, volumeToMount.VolumeName)
		volumeToMount.DevicePath = devicePath
		if cache.IsVolumeNotAttachedError(err) {
			if rc.controllerAttachDetachEnabled || !volumeToMount.PluginIsAttachable {
				// Volume is not attached (or doesn't implement attacher), kubelet attach is disabled, wait
				// for controller to finish attaching volume.
				glog.V(12).Infof("Attempting to start VerifyControllerAttachedVolume for volume %q (spec.Name: %q) pod %q (UID: %q)",
					volumeToMount.VolumeName,
					volumeToMount.VolumeSpec.Name(),
					volumeToMount.PodName,
					volumeToMount.Pod.UID)
				err := rc.operationExecutor.VerifyControllerAttachedVolume(
					volumeToMount.VolumeToMount,
					rc.hostName,
					rc.actualStateOfWorld)
				if err != nil &&
					!nestedpendingoperations.IsAlreadyExists(err) &&
					!exponentialbackoff.IsExponentialBackoff(err) {
					// Ignore nestedpendingoperations.IsAlreadyExists and exponentialbackoff.IsExponentialBackoff errors, they are expected.
					// Log all other errors.
					glog.Errorf(
						"operationExecutor.VerifyControllerAttachedVolume failed for volume %q (spec.Name: %q) pod %q (UID: %q) controllerAttachDetachEnabled: %v with err: %v",
						volumeToMount.VolumeName,
						volumeToMount.VolumeSpec.Name(),
						volumeToMount.PodName,
						volumeToMount.Pod.UID,
						rc.controllerAttachDetachEnabled,
						err)
				}
				if err == nil {
					glog.Infof("VerifyControllerAttachedVolume operation started for volume %q (spec.Name: %q) pod %q (UID: %q)",
						volumeToMount.VolumeName,
						volumeToMount.VolumeSpec.Name(),
						volumeToMount.PodName,
						volumeToMount.Pod.UID)
				}
			} else {
				// Volume is not attached to node, kubelet attach is enabled, volume implements an attacher,
				// so attach it
				volumeToAttach := operationexecutor.VolumeToAttach{
					VolumeName: volumeToMount.VolumeName,
					VolumeSpec: volumeToMount.VolumeSpec,
					NodeName:   rc.hostName,
				}
				glog.V(12).Infof("Attempting to start AttachVolume for volume %q (spec.Name: %q)  pod %q (UID: %q)",
					volumeToMount.VolumeName,
					volumeToMount.VolumeSpec.Name(),
					volumeToMount.PodName,
					volumeToMount.Pod.UID)
				err := rc.operationExecutor.AttachVolume(volumeToAttach, rc.actualStateOfWorld)
				if err != nil &&
					!nestedpendingoperations.IsAlreadyExists(err) &&
					!exponentialbackoff.IsExponentialBackoff(err) {
					// Ignore nestedpendingoperations.IsAlreadyExists and exponentialbackoff.IsExponentialBackoff errors, they are expected.
					// Log all other errors.
					glog.Errorf(
						"operationExecutor.AttachVolume failed for volume %q (spec.Name: %q) pod %q (UID: %q) controllerAttachDetachEnabled: %v with err: %v",
						volumeToMount.VolumeName,
						volumeToMount.VolumeSpec.Name(),
						volumeToMount.PodName,
						volumeToMount.Pod.UID,
						rc.controllerAttachDetachEnabled,
						err)
				}
				if err == nil {
					glog.Infof("AttachVolume operation started for volume %q (spec.Name: %q) pod %q (UID: %q)",
						volumeToMount.VolumeName,
						volumeToMount.VolumeSpec.Name(),
						volumeToMount.PodName,
						volumeToMount.Pod.UID)
				}
			}
		} else if !volMounted || cache.IsRemountRequiredError(err) {
			// Volume is not mounted, or is already mounted, but requires remounting
			remountingLogStr := ""
			if cache.IsRemountRequiredError(err) {
				remountingLogStr = "Volume is already mounted to pod, but remount was requested."
			}
			glog.V(12).Infof("Attempting to start MountVolume for volume %q (spec.Name: %q) to pod %q (UID: %q). %s",
				volumeToMount.VolumeName,
				volumeToMount.VolumeSpec.Name(),
				volumeToMount.PodName,
				volumeToMount.Pod.UID,
				remountingLogStr)
			err := rc.operationExecutor.MountVolume(
				rc.waitForAttachTimeout,
				volumeToMount.VolumeToMount,
				rc.actualStateOfWorld)
			if err != nil &&
				!nestedpendingoperations.IsAlreadyExists(err) &&
				!exponentialbackoff.IsExponentialBackoff(err) {
				// Ignore nestedpendingoperations.IsAlreadyExists and exponentialbackoff.IsExponentialBackoff errors, they are expected.
				// Log all other errors.
				glog.Errorf(
					"operationExecutor.MountVolume failed for volume %q (spec.Name: %q) pod %q (UID: %q) controllerAttachDetachEnabled: %v with err: %v",
					volumeToMount.VolumeName,
					volumeToMount.VolumeSpec.Name(),
					volumeToMount.PodName,
					volumeToMount.Pod.UID,
					rc.controllerAttachDetachEnabled,
					err)
			}
			if err == nil {
				glog.Infof("MountVolume operation started for volume %q (spec.Name: %q) to pod %q (UID: %q). %s",
					volumeToMount.VolumeName,
					volumeToMount.VolumeSpec.Name(),
					volumeToMount.PodName,
					volumeToMount.Pod.UID,
					remountingLogStr)
			}
		}
	}

	// Ensure devices that should be detached/unmounted are detached/unmounted.
	for _, attachedVolume := range rc.actualStateOfWorld.GetUnmountedVolumes() {
		if !rc.desiredStateOfWorld.VolumeExists(attachedVolume.VolumeName) {
			if attachedVolume.GloballyMounted {
				// Volume is globally mounted to device, unmount it
				glog.V(12).Infof("Attempting to start UnmountDevice for volume %q (spec.Name: %q)",
					attachedVolume.VolumeName,
					attachedVolume.VolumeSpec.Name())
				err := rc.operationExecutor.UnmountDevice(
					attachedVolume.AttachedVolume, rc.actualStateOfWorld, rc.mounter)
				if err != nil &&
					!nestedpendingoperations.IsAlreadyExists(err) &&
					!exponentialbackoff.IsExponentialBackoff(err) {
					// Ignore nestedpendingoperations.IsAlreadyExists and exponentialbackoff.IsExponentialBackoff errors, they are expected.
					// Log all other errors.
					glog.Errorf(
						"operationExecutor.UnmountDevice failed for volume %q (spec.Name: %q) controllerAttachDetachEnabled: %v with err: %v",
						attachedVolume.VolumeName,
						attachedVolume.VolumeSpec.Name(),
						rc.controllerAttachDetachEnabled,
						err)
				}
				if err == nil {
					glog.Infof("UnmountDevice operation started for volume %q (spec.Name: %q)",
						attachedVolume.VolumeName,
						attachedVolume.VolumeSpec.Name())
				}
			} else {
				// Volume is attached to node, detach it
				if rc.controllerAttachDetachEnabled || !attachedVolume.PluginIsAttachable {
					// Kubelet not responsible for detaching or this volume has a non-attachable volume plugin,
					// so just remove it to actualStateOfWorld without attach.
					rc.actualStateOfWorld.MarkVolumeAsDetached(
						attachedVolume.VolumeName, rc.hostName)
				} else {
					// Only detach if kubelet detach is enabled
					glog.V(12).Infof("Attempting to start DetachVolume for volume %q (spec.Name: %q)",
						attachedVolume.VolumeName,
						attachedVolume.VolumeSpec.Name())
					err := rc.operationExecutor.DetachVolume(
						attachedVolume.AttachedVolume, false /* verifySafeToDetach */, rc.actualStateOfWorld)
					if err != nil &&
						!nestedpendingoperations.IsAlreadyExists(err) &&
						!exponentialbackoff.IsExponentialBackoff(err) {
						// Ignore nestedpendingoperations.IsAlreadyExists && exponentialbackoff.IsExponentialBackoff errors, they are expected.
						// Log all other errors.
						glog.Errorf(
							"operationExecutor.DetachVolume failed for volume %q (spec.Name: %q) controllerAttachDetachEnabled: %v with err: %v",
							attachedVolume.VolumeName,
							attachedVolume.VolumeSpec.Name(),
							rc.controllerAttachDetachEnabled,
							err)
					}
					if err == nil {
						glog.Infof("DetachVolume operation started for volume %q (spec.Name: %q)",
							attachedVolume.VolumeName,
							attachedVolume.VolumeSpec.Name())
					}
				}
			}
		}
	}
}

// reconstruct process tries to observe the real world by scanning all pods' volume directories from the disk.
// If the actual and desired state of worlds are not consistent with the observed world, it means that some
// mounted volumes are left out probably during kubelet restart. This process will reconstruct
// the volumes and udpate the actual and desired states. In the following reconciler loop, those volumes will
// be cleaned up.
func (rc *reconciler) reconstruct() {
	defer rc.updateReconstructTime()
	rc.reconstructStates(rc.kubeletPodsDir)
}

func (rc *reconciler) updateReconstructTime() {
	rc.timeOfLastReconstruct = time.Now()
}

type podVolume struct {
	podName        volumetypes.UniquePodName
	volumeSpecName string
	mountPath      string
	pluginName     string
}

// reconstructFromDisk scans the volume directories under the given pod directory. If the volume is not
// in either actual or desired state of world, or pending operation, this function will reconstruct
// the volume spec and put it in both the actual and desired state of worlds. If no running
// container is mounting the volume, the volume will be removed by desired state of world's populator and
// cleaned up by the reconciler.
func (rc *reconciler) reconstructStates(podsDir string) {
	// Get volumes information by reading the pod's directory
	podVolumes, err := getVolumesFromPodDir(podsDir)
	if err != nil {
		glog.Errorf("Cannot get volumes from disk %v", err)
		return
	}
	for _, volume := range podVolumes {
		volumeToMount, err := rc.reconstructVolume(volume)
		if err != nil {
			glog.Errorf("Could not construct volume information: %v", err)
			continue
		}

		// Check if there is an pending operation for the given pod and volume.
		// Need to check pending operation before checking the actual and desired
		// states to avoid race condition during checking. For example, the following
		// might happen if pending operation is checked after checking actual and desired states.
		// 1. Checking the pod and it does not exist in either actual or desired state.
		// 2. An operation for the given pod finishes and the actual state is updated.
		// 3. Checking and there is no pending operation for the given pod.
		if rc.operationExecutor.IsOperationPending(volumeToMount.VolumeName, volumeToMount.PodName) {
			continue
		}
		desiredPods := rc.desiredStateOfWorld.GetPods()
		actualPods := rc.actualStateOfWorld.GetPods()
		if desiredPods[volume.podName] || actualPods[volume.podName] {
			continue
		}

		glog.V(3).Infof(
			"Could not find pod information in desired or actual states or pending operation, update it in both states: %+v",
			volumeToMount)
		if err = rc.updateStates(volumeToMount); err != nil {
			glog.Errorf("Error occurred during reconstruct volume from disk: %v", err)
		}
	}
}

// Reconstruct Volume object and volumeToMount data structure by reading the pod's volume directories
func (rc *reconciler) reconstructVolume(volume podVolume) (*operationexecutor.VolumeToMount, error) {
	plugin, err := rc.volumePluginMgr.FindPluginByName(volume.pluginName)
	if err != nil {
		return nil, err
	}
	volumeSpec, err := plugin.ConstructVolumeSpec(volume.volumeSpecName, volume.mountPath)
	if err != nil {
		return nil, err
	}
	volumeName, err := plugin.GetVolumeName(volumeSpec)
	if err != nil {
		return nil, err
	}
	pod := &api.Pod{
		ObjectMeta: api.ObjectMeta{
			UID: types.UID(volume.podName),
		},
	}
	attachablePlugin, err := rc.volumePluginMgr.FindAttachablePluginByName(volume.pluginName)
	if err != nil {
		return nil, err
	}
	var uniqueVolumeName api.UniqueVolumeName
	if attachablePlugin != nil {
		uniqueVolumeName = volumehelper.GetUniqueVolumeName(volume.pluginName, volumeName)
	} else {
		uniqueVolumeName = volumehelper.GetUniqueVolumeNameForNonAttachableVolume(volume.podName, plugin, volumeSpec)
	}

	volumeToMount := &operationexecutor.VolumeToMount{
		VolumeName:          uniqueVolumeName,
		PodName:             volume.podName,
		VolumeSpec:          volumeSpec,
		OuterVolumeSpecName: volumeName, /*volumeName is InnerVolumeSpecName. But this information will not be used for cleanup*/
		Pod:                 pod,
		PluginIsAttachable:  attachablePlugin != nil,
		VolumeGidValue:      "",
		DevicePath:          "",
	}
	return volumeToMount, nil
}

func (rc *reconciler) updateStates(volumeToMount *operationexecutor.VolumeToMount) error {
	err := rc.actualStateOfWorld.MarkVolumeAsAttached(
		volumeToMount.VolumeName, volumeToMount.VolumeSpec, "", volumeToMount.DevicePath)
	if err != nil {
		return fmt.Errorf("Could not add volume information to actual state of world: %v", err)
	}
	err = rc.actualStateOfWorld.AddPodToVolume(
		volumeToMount.PodName,
		types.UID(volumeToMount.PodName),
		volumeToMount.VolumeName,
		nil,
		volumeToMount.OuterVolumeSpecName,
		volumeToMount.DevicePath)
	if err != nil {
		return fmt.Errorf("Could not add pod to volume information to actual state of world: %v", err)
	}
	if volumeToMount.PluginIsAttachable {
		err = rc.actualStateOfWorld.MarkDeviceAsMounted(volumeToMount.VolumeName)
		if err != nil {
			return fmt.Errorf("Could not mark device is mounted to actual state of world: %v", err)
		}
	}
	_, err = rc.desiredStateOfWorld.AddPodToVolume(volumeToMount.PodName,
		volumeToMount.Pod,
		volumeToMount.VolumeSpec,
		volumeToMount.OuterVolumeSpecName,
		volumeToMount.VolumeGidValue)
	if err != nil {
		return fmt.Errorf("Could not add pod to volume information to desired state of world: %v", err)
	}
	return nil
}

// getVolumesFromPodDir scans through the volumes directories under the given pod directory.
// It returns a list of pod volume information including pod's uid, volume's plugin name, mount path,
// and volume spec name.
func getVolumesFromPodDir(podDir string) ([]podVolume, error) {
	podsDirInfo, err := ioutil.ReadDir(podDir)
	if err != nil {
		return nil, err
	}
	volumes := []podVolume{}
	for i := range podsDirInfo {
		if !podsDirInfo[i].IsDir() {
			continue
		}
		podName := podsDirInfo[i].Name()
		podDir := path.Join(podDir, podName)
		volumesDir := path.Join(podDir, options.DefaultKubeletVolumesDirName)
		volumesDirInfo, err := ioutil.ReadDir(volumesDir)
		if err != nil {
			glog.Errorf("Could not read volume directory %q: %v", volumesDir, err)
			continue
		}
		for _, volumeDir := range volumesDirInfo {
			pluginName := volumeDir.Name()
			volumePluginPath := path.Join(volumesDir, pluginName)

			volumePluginDirs, err := ioutil.ReadDir(volumePluginPath)
			if err != nil {
				glog.Errorf("Could not read volume plugin directory %q: %v", volumePluginPath, err)
				continue
			}

			unescapePluginName := strings.UnescapeQualifiedNameForDisk(pluginName)
			for _, volumeNameDir := range volumePluginDirs {
				if volumeNameDir != nil {
					volumeName := volumeNameDir.Name()
					mountPath := path.Join(volumePluginPath, volumeName)
					volumes = append(volumes, podVolume{
						podName:        volumetypes.UniquePodName(podName),
						volumeSpecName: volumeName,
						mountPath:      mountPath,
						pluginName:     unescapePluginName,
					})
				}
			}

		}
	}
	glog.V(10).Infof("Get volumes from pod directory %q %+v", podDir, volumes)
	return volumes, nil
}
