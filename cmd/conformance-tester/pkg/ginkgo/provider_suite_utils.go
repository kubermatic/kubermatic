package ginkgo

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/clients"
	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/scenarios"
	legacytypes "k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/types"
	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/util"
	controllerutil "k8c.io/kubermatic/v2/pkg/controller/util"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"
	clusterv1alpha1 "k8c.io/machine-controller/sdk/apis/cluster/v1alpha1"
	"k8c.io/machine-controller/sdk/providerconfig"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/util/retry"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func scenarioEntriesByProvider(s []scenarios.Scenario, p providerconfig.CloudProvider) []TableEntry {
	var entries []TableEntry
	for _, scenario := range s {
		if scenario.CloudProvider() == kubermaticv1.ProviderType(p) {
			entries = append(entries, Entry(scenario.Name(), scenario))
		}
	}
	return entries
}

func scenarioEntries(s []scenarios.Scenario) []TableEntry {
	var entries []TableEntry
	for _, scenario := range s {
		entries = append(entries, Entry(string(scenario.CloudProvider()), scenario))

	}
	return entries
}

func KKP(msg string) string {
	return fmt.Sprintf("[KKP] %s", msg)
}

func CloudProvider(msg string) string {
	return fmt.Sprintf("[CloudProvider] %s", msg)
}

func commonSetup(rootCtx context.Context, log *zap.SugaredLogger, scenario scenarios.Scenario, legacyOpts *legacytypes.Options) (*kubermaticv1.Cluster, ctrlruntimeclient.Client) {
	var userClusterClient ctrlruntimeclient.Client
	var cluster *kubermaticv1.Cluster
	var err error
	// By("Creating a new cluster")
	// legacyOpts := toLegacyOptions(opts, runtimeOpts)
	By(KKP("Create Cluster"), func() {
		cluster, err = clients.NewKubeClient(legacyOpts).CreateCluster(rootCtx, log, scenario)
		Expect(err).NotTo(HaveOccurred())
	})

	By(KKP("Wait for successful reconciliation"), func() {
		// NB: It's important for this health check loop to refresh the cluster object, as
		// during reconciliation some cloud providers will fill in missing fields in the CloudSpec,
		// and later when we create MachineDeployments we potentially rely on these fields
		// being set in the cluster variable.
		versions := kubermatic.GetVersions()
		log.Info("Waiting for cluster to be successfully reconciled...")

		Eventually(func() bool {
			if err := legacyOpts.SeedClusterClient.Get(rootCtx, types.NamespacedName{Name: cluster.Name}, cluster); err != nil {
				return false
			}

			// ignore Kubermatic version in this check, to allow running against a 3rd party setup
			missingConditions, _ := controllerutil.ClusterReconciliationSuccessful(cluster, versions, true)
			if len(missingConditions) > 0 {
				return false
			}

			return true
		}, 5*time.Second, 10*time.Minute).Should(BeTrue(), "cluster was not reconciled successfully within the timeout")

		Eventually(func() bool {
			newCluster := &kubermaticv1.Cluster{}
			namespacedClusterName := types.NamespacedName{Name: cluster.Name}
			if err := legacyOpts.SeedClusterClient.Get(rootCtx, namespacedClusterName, newCluster); err != nil {
				if apierrors.IsNotFound(err) {
					return false
				}
			}

			// Check for this first, because otherwise we instantly return as the cluster-controller did not
			// create any pods yet
			if !newCluster.Status.ExtendedHealth.AllHealthy() {
				return false
			}

			controlPlanePods := &corev1.PodList{}
			if err := legacyOpts.SeedClusterClient.List(
				rootCtx,
				controlPlanePods,
				&ctrlruntimeclient.ListOptions{Namespace: newCluster.Status.NamespaceName},
			); err != nil {
				return false
			}

			unready := sets.New[string]()
			for _, pod := range controlPlanePods.Items {
				if !util.PodIsReady(&pod) {
					unready.Insert(pod.Name)
				}
			}

			if unready.Len() == 0 {
				return true
			}

			return false
		}, legacyOpts.ControlPlaneReadyWaitTimeout, 5*time.Second).Should(BeTrue(), "cluster did not become healthy within the timeout")
	})

	By(KKP("Wait for control plane"), func() {
		Eventually(func() bool {
			if err := legacyOpts.SeedClusterClient.Get(rootCtx, types.NamespacedName{Name: cluster.Name}, cluster); err != nil {
				return false
			}
			versions := kubermatic.GetVersions()
			// ignore Kubermatic version in this check, to allow running against a 3rd party setup
			missingConditions, _ := controllerutil.ClusterReconciliationSuccessful(cluster, versions, true)
			return len(missingConditions) == 0
		}, 10*time.Minute, 5*time.Second).Should(BeTrue())

		Eventually(func() bool {
			var err error
			userClusterClient, err = legacyOpts.ClusterClientProvider.GetClient(rootCtx, cluster)
			return err == nil
		}, 10*time.Minute, 5*time.Second).Should(BeTrue())
	})

	By(KKP("Add LB and PV Finalizers"), func() {
		Expect(retry.RetryOnConflict(retry.DefaultBackoff, func() error {
			if err := legacyOpts.SeedClusterClient.Get(rootCtx, types.NamespacedName{Name: cluster.Name}, cluster); err != nil {
				return err
			}
			cluster.Finalizers = append(cluster.Finalizers,
				kubermaticv1.InClusterPVCleanupFinalizer,
				kubermaticv1.InClusterLBCleanupFinalizer,
			)
			return legacyOpts.SeedClusterClient.Update(rootCtx, cluster)
		})).NotTo(HaveOccurred(), "failed to add finalizers to the cluster")
	})

	return cluster, userClusterClient
}

func commonCleanup(rootCtx context.Context, log *zap.SugaredLogger, client clients.Client, scenario scenarios.Scenario, userClusterClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) {
	// By(KKP("Removing machine deployment"))
	err := client.DeleteMachineDeployments(rootCtx, log, scenario, userClusterClient, cluster)
	Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("failed to delete machine deployments with error %v", err))
	By(KKP("Delete cluster"), func() {
		deleteTimeout := 15 * time.Minute
		err = client.DeleteCluster(rootCtx, log, cluster, deleteTimeout)
		Expect(err).NotTo(HaveOccurred())
	})
	log.Info("Ending scenario test")
}

func machineSetup(rootCtx context.Context, log *zap.SugaredLogger, client clients.Client, scenario scenarios.Scenario, userClusterClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster, legacyOpts *legacytypes.Options) {
	By(KKP("Create MachineDeployments"), func() {
		err := client.CreateMachineDeployments(rootCtx, log, scenario, userClusterClient, cluster)
		Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("failed to create machine deployments with error %v", err))
	})
	By(KKP("Wait for machines to get a node"), func() {
		Eventually(func() bool {
			machineList := &clusterv1alpha1.MachineList{}
			if err := userClusterClient.List(rootCtx, machineList); err != nil {
				return false
			}
			if len(machineList.Items) < legacyOpts.NodeCount {
				return false
			}

			for _, machine := range machineList.Items {
				if machine.Status.NodeRef == nil || machine.Status.NodeRef.Name == "" {
					return false
				}
			}

			return true
		}, 10*time.Minute, 5*time.Second).Should(BeTrue(), "not all machines got a node within the timeout")
	})
	By(KKP("Wait for nodes to be ready"), func() {
		Eventually(func() bool {
			nodeList := &corev1.NodeList{}
			if err := userClusterClient.List(rootCtx, nodeList); err != nil {
				return false
			}

			unready := sets.New[string]()
			for _, node := range nodeList.Items {
				if !util.NodeIsReady(node) {
					unready.Insert(node.Name)
				}
			}

			if unready.Len() == 0 {
				return true
			}

			return false
		}, legacyOpts.NodeReadyTimeout, 5*time.Second).Should(BeTrue(), "not all nodes became ready within the timeout")
	})
	By(KKP("Wait for Pods inside usercluster to be ready"), func() {
		Eventually(func() bool {
			podList := &corev1.PodList{}
			if err := userClusterClient.List(rootCtx, podList); err != nil {
				return false
			}

			unready := sets.New[string]()
			for _, pod := range podList.Items {
				// Ignore pods failing kubelet admission (KKP #6185)
				if !util.PodIsReady(&pod) && !podFailedKubeletAdmissionDueToNodeAffinityPredicate(&pod, log) && !util.PodIsCompleted(&pod) {
					unready.Insert(pod.Name)
				}
			}

			if unready.Len() == 0 {
				return true
			}

			return false
		}, legacyOpts.NodeReadyTimeout, legacyOpts.UserClusterPollInterval).Should(BeTrue(), "not all pods became ready within the timeout")
	})
	By(KKP("Wait for addons"), func() {
		Eventually(func() bool {
			addons := kubermaticv1.AddonList{}
			if err := legacyOpts.SeedClusterClient.List(rootCtx, &addons, ctrlruntimeclient.InNamespace(cluster.Status.NamespaceName)); err != nil {
				return false
			}

			unhealthyAddons := sets.New[string]()
			for _, addon := range addons.Items {
				if addon.Status.Conditions[kubermaticv1.AddonReconciledSuccessfully].Status != corev1.ConditionTrue {
					unhealthyAddons.Insert(addon.Name)
				}
			}

			if unhealthyAddons.Len() > 0 {
				return false
			}

			return true
		}, 2*time.Minute, 2*time.Second).Should(BeTrue(), "not all addons became healthy within the timeout")
	})
}

// podFailedKubeletAdmissionDueToNodeAffinityPredicate detects a condition in
// which a pod is scheduled but fails kubelet admission due to a race condition
// between scheduler and kubelet.
// see: https://github.com/kubernetes/kubernetes/issues/93338
func podFailedKubeletAdmissionDueToNodeAffinityPredicate(p *corev1.Pod, log *zap.SugaredLogger) bool {
	failedAdmission := p.Status.Phase == "Failed" && p.Status.Reason == "NodeAffinity"
	if failedAdmission {
		log.Infow("pod failed kubelet admission due to NodeAffinity predicate", "pod", *p)
	}

	return failedAdmission
}
