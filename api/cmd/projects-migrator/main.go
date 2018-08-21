package main

import (
	"flag"
	"fmt"

	"github.com/golang/glog"

	"github.com/kubermatic/kubermatic/api/pkg/controller/rbac"
	kubermaticclientset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type migrationContext struct {
	masterKubeClient       kubernetes.Interface
	masterKubermaticClient kubermaticclientset.Interface
	config                 *rest.Config
	seedClusterProviders   []*clusterProvider
	dryRun                 bool
}

// clusterProvider holds set of clients that allow for communication with the cluster and
type clusterProvider struct {
	name             string
	kubeClient       kubernetes.Interface
	kubermaticClient kubermaticclientset.Interface
}

func main() {
	var kubeconfig, masterURL string
	ctx := migrationContext{}

	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	flag.BoolVar(&ctx.dryRun, "dry-run", true, "If true, only print the object that would be created, without creating them")
	flag.Parse()

	var err error
	ctx.config, err = clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		glog.Fatal(err)
	}

	clusterProviders := []*clusterProvider{}
	{
		clientcmdConfig, err := clientcmd.LoadFromFile(kubeconfig)
		if err != nil {
			glog.Fatal(err)
		}

		for ctxName := range clientcmdConfig.Contexts {
			clientConfig := clientcmd.NewNonInteractiveClientConfig(
				*clientcmdConfig,
				ctxName,
				&clientcmd.ConfigOverrides{CurrentContext: ctxName},
				nil,
			)
			cfg, err := clientConfig.ClientConfig()
			if err != nil {
				glog.Fatal(err)
			}
			if cfg.Host == ctx.config.Host && cfg.Username == ctx.config.Username && cfg.Password == ctx.config.Password {
				glog.V(2).Infof("Skipping adding %s as a seed cluster. It is exactly the same as existing kubernetes master client", ctxName)
				continue
			}

			glog.V(2).Infof("Adding %s as seed cluster", ctxName)
			kubeClient := kubernetes.NewForConfigOrDie(cfg)
			kubermaticClient := kubermaticclientset.NewForConfigOrDie(ctx.config)
			clusterProviders = append(clusterProviders, &clusterProvider{fmt.Sprintf("seed/%s", ctxName), kubeClient, kubermaticClient})
		}
	}

	ctx.masterKubeClient = kubernetes.NewForConfigOrDie(ctx.config)
	ctx.masterKubermaticClient = kubermaticclientset.NewForConfigOrDie(ctx.config)
	ctx.seedClusterProviders = clusterProviders

	// phase 0: remove duplicated users
	//
	//          duplicated users have the same Spec.ID, Spec.Email, Spec.Name
	//          for example on dev environment: user-lknc7, user-n45j9, user-z9s20j
	err = removeDuplicatedUsers(ctx)
	if err != nil {
		glog.Errorf("phase 0 failed due to = %v", err)
	}

	// phase 1: migrate existing cluster resources along with ssh keys they use to projects
	err = migrateToProject(ctx)
	if err != nil {
		glog.Errorf("phase 1 failed due to = %v", err)
	}

	// phase 2: migrate the remaining ssh keys to a project
	//
	//          note that:
	//          the remaining ssh keys are the ones that are owned by
	//          the project owner and were not used by running cluster (phase 1)
	err = migrateRemainingSSHKeys(ctx)
	if err != nil {
		glog.Errorf("phase 2 failed due to %v", err)
	}

	// TODO:
	// phase 3 clean up
	//         - remove "user" label for cluster resources - that belong to a project
	//         - remove "owner" field for ssh keys resources - that belong to a project
	//
	//         note that:
	//         this step essentially breaks backward compatibility and prevents the old clients (dashboard) from finding the resources.

	// TODO:
	// phase 4 remove ssh keys without an owner
	//
	//         the keys that don't have an owner have their sshKey.Spec.Owner field empty
	//         for example: key-8036218bef587f8230dad6426099da14-c7i4 and key-8036218bef587f8230dad6426099da14-wwk5 on dev env
}

//  migrateRemainingSSHKeys assigns the keys that are owned by the project owner
func migrateRemainingSSHKeys(ctx migrationContext) error {
	glog.Info("\n")
	glog.Infof("Running PHASE 2 ...")

	type projectOwnerTuple struct {
		project kubermaticv1.Project
		owner   kubermaticv1.User
	}

	projectOwnersTuple := map[string]projectOwnerTuple{}
	glog.Info("STEP 1: getting the list of projects owners")
	{
		allProjects, err := ctx.masterKubermaticClient.KubermaticV1().Projects().List(metav1.ListOptions{})
		if err != nil {
			return err
		}
		for _, project := range allProjects.Items {
			userName := isOwnedByUser(project.OwnerReferences)
			if len(userName) > 0 {
				if user, err := ctx.masterKubermaticClient.KubermaticV1().Users().Get(userName, metav1.GetOptions{}); err == nil {
					projectOwnersTuple[userName] = projectOwnerTuple{project: project, owner: *user}
				}
			} else {
				return fmt.Errorf("project ID = %s, Name = %s doesn't have an owner", project.Name, project.Spec.Name)
			}
		}
	}

	type keyProjectTuple struct {
		key     kubermaticv1.UserSSHKey
		project kubermaticv1.Project
	}

	keysProjectTuple := []keyProjectTuple{}
	glog.Info("STEP 2: getting the list of keys that are owned by a project owner")
	{
		allKeys, err := ctx.masterKubermaticClient.KubermaticV1().UserSSHKeies().List(metav1.ListOptions{})
		if err != nil {
			return err
		}

		for _, key := range allKeys.Items {
			if len(key.Spec.Owner) == 0 {
				glog.Warningf("the key ID = %s, Name = %s doesn't have an owner", key.Name, key.Spec.Name)
				continue
			}
			if projectOwner, ok := projectOwnersTuple[key.Spec.Owner]; ok {
				keysProjectTuple = append(keysProjectTuple, keyProjectTuple{key: key, project: projectOwner.project})
			} else {
				glog.V(3).Infof("the owner = %s of the key ID = %s, Name = %s doesn't have a project", key.Spec.Owner, key.Name, key.Spec.Name)
			}
		}

		sshKeysToAdoptByProjectID := map[string][]kubermaticv1.UserSSHKey{}
		for _, keyProject := range keysProjectTuple {
			keys := sshKeysToAdoptByProjectID[keyProject.project.Name]
			keys = append(keys, keyProject.key)
			sshKeysToAdoptByProjectID[keyProject.project.Name] = keys
		}
		printSSHKeysToAdopt("project", sshKeysToAdoptByProjectID)
	}

	glog.Info("STEP 3: migrating the remaining keys")
	{
		for _, keyProject := range keysProjectTuple {
			oRef := createOwnerReferenceForProject(keyProject.project)
			key := keyProject.key
			key.OwnerReferences = append(key.OwnerReferences, oRef)
			if !ctx.dryRun {
				_, err := ctx.masterKubermaticClient.KubermaticV1().UserSSHKeies().Update(&key)
				if err != nil {
					return err
				}
				glog.Infof("the ssh key = %s was migrated to the project = %s from the system", key.Name, keyProject.project.Name)
			} else {
				glog.Infof("the ssh key = %s was NOT migrated to the project = %s because dry-run option was requested", key.Name, keyProject.project.Name)
			}
		}
	}

	return nil
}

// removeDuplicatedUsers finds users with the same spec.ID, spec.Email and spec.Name
// and removes duplication. This is safe to do because resources like clusters are bound to the user
// if there is a match on spec.ID field.
func removeDuplicatedUsers(ctx migrationContext) error {
	glog.Info("Running PHASE 0 ...")
	allUsers, err := ctx.masterKubermaticClient.KubermaticV1().Users().List(metav1.ListOptions{})
	if err != nil {
		return err
	}

	seenUsersKey := func(user kubermaticv1.User) string {
		return fmt.Sprintf("%s:%s:%s", user.Spec.ID, user.Spec.Email, user.Spec.Name)
	}

	seenUsers := map[string][]kubermaticv1.User{}
	glog.Info("STEP 1: building the list of duplicated users in the system")
	{
		for _, user := range allUsers.Items {
			key := seenUsersKey(user)
			seenUserKeyListToUpdate := []kubermaticv1.User{}
			if seenUserList, ok := seenUsers[seenUsersKey(user)]; ok {
				seenUserKeyListToUpdate = append(seenUserList, user)
			} else {
				seenUserKeyListToUpdate = append(seenUserKeyListToUpdate, user)
			}
			seenUsers[key] = seenUserKeyListToUpdate
		}
	}
	printDuplicatedUsers(seenUsers)

	glog.Info("STEP 2: removing duplicated users if any")
	{
		for _, userList := range seenUsers {
			if len(userList) > 1 {
				for i := 0; i < len(userList)-1; i++ {
					userName := userList[i].Name
					if !ctx.dryRun {
						err := ctx.masterKubermaticClient.KubermaticV1().Users().Delete(userName, &metav1.DeleteOptions{})
						if err != nil {
							return err
						}
						glog.Infof("the user = %s was removed from the system", userName)
					} else {
						glog.Infof("the user = %s was NOT removed from the system because dry-run option was requested", userName)
					}
				}
			}
		}
	}
	return nil
}

// migrateToProject starts the migration of existing resources.
// The purpose of this step is to find all running clusters and group them by the user they belong to.
// Similarly with ssh keys that are being used by the clusters.
// Next we create a default project for the users and start the migration process by updating the existing resources.
func migrateToProject(ctx migrationContext) error {
	glog.Info("\n")
	glog.Info("Running PHASE 1 ...")
	//
	// step 1: get clusters that doesn't belong to any project,
	//         clusters like that don't have OwnerReferences set for a project
	//
	//         note that this step will get clusters resources that belong to
	//         master and seed clusters (physical location)
	glog.Info("STEP 1: getting the list of clusters that needs to be migrated")
	clustersToAdoptByUserID, err := getAllClusters(ctx)
	if err != nil {
		return err
	}
	printClusterToAdopt(clustersToAdoptByUserID)

	//
	// step 2: get the list of users that own "clustersToAdopt"
	glog.Info("STEP 2: getting the list of users that own the clusters")
	ownersOfClustersToAdopt := []kubermaticv1.User{}
	{
		allUsers, err := ctx.masterKubermaticClient.KubermaticV1().Users().List(metav1.ListOptions{})
		if err != nil {
			return err
		}
		for _, user := range allUsers.Items {
			if clustersMap, ok := clustersToAdoptByUserID[user.Spec.ID]; ok {
				doubledUserID := false
				for _, clusterResourcesTuple := range clustersMap {
					for _, clusterResource := range clusterResourcesTuple.clusters {
						if clusterResource.Status.UserEmail != user.Spec.Email {
							doubledUserID = true
							glog.V(3).Infof(
								"the running cluster belongs to the user (%s) with Spec.ID = %s, Name = %s and Email = %s, but the user with the same ID but different Email = %s exists in the system - skipping this user",
								user.Name, user.Spec.ID, clusterResource.Status.UserName, clusterResource.Status.UserEmail,
								user.Spec.Email,
							)
							break
						}
					}
					break
				}
				if doubledUserID {
					continue
				}
				for _, ownerOfCluster := range ownersOfClustersToAdopt {
					if ownerOfCluster.Spec.Name == user.Spec.Name && ownerOfCluster.Spec.Email == user.Spec.Email && ownerOfCluster.Spec.ID == user.Spec.ID {
						return fmt.Errorf("user (%s) with Name = %s, Email = %s and ID = %s was already added to the list", user.Name, user.Spec.Name, user.Spec.Email, user.Spec.ID)
					}
				}
				glog.V(3).Infof("adding the user with ID = %s (%s) and name = %s to the list ", user.Name, user.Spec.ID, user.Spec.Name)
				ownersOfClustersToAdopt = append(ownersOfClustersToAdopt, user)
			}
		}
	}

	//
	// step 3: create a default project for each user
	//         if was not already created
	glog.Info("STEP 3: creating a default project for each user if not already exists")
	ownersOfClusterWithProject := map[string]kubermaticv1.Project{}
	{
		defaultProjectName := "default"
		for _, user := range ownersOfClustersToAdopt {
			projectName := ""
			// if the users owns more than one project take the first one
			for _, projectGroup := range user.Spec.Projects {
				if rbac.ExtractGroupPrefix(projectGroup.Group) == rbac.OwnerGroupNamePrefix {
					projectName = projectGroup.Name
					break
				}
			}
			if len(projectName) > 0 {
				project, err := ctx.masterKubermaticClient.KubermaticV1().Projects().Get(projectName, metav1.GetOptions{})
				if err != nil {
					return err
				}
				ownersOfClusterWithProject[user.Spec.ID] = *project
				glog.Infof("the userID = %s already has a project (ID = %s, Name = %s)", user.Spec.ID, project.Name, project.Spec.Name)
				continue
			}

			// create a project for the given user
			project := &kubermaticv1.Project{
				ObjectMeta: metav1.ObjectMeta{
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: kubermaticv1.SchemeGroupVersion.String(),
							Kind:       kubermaticv1.UserKindName,
							UID:        user.GetUID(),
							Name:       user.Name,
						},
					},
					Name: rand.String(10),
				},
				Spec: kubermaticv1.ProjectSpec{
					Name: defaultProjectName,
				},
				Status: kubermaticv1.ProjectStatus{
					Phase: kubermaticv1.ProjectInactive,
				},
			}

			createdProject := &kubermaticv1.Project{}
			if !ctx.dryRun {
				var err error
				createdProject, err = ctx.masterKubermaticClient.KubermaticV1().Projects().Create(project)
				if err != nil {
					return err
				}
			} else {
				glog.Infof("a project (ID = %s, Name = %s) for userID = %s was NOT created because dry-run option was requested", project.Name, project.Spec.Name, user.Spec.ID)
				createdProject = project
			}

			user.Spec.Projects = append(user.Spec.Projects, kubermaticv1.ProjectGroup{Group: rbac.GenerateActualGroupNameFor(createdProject.Name, rbac.OwnerGroupNamePrefix), Name: createdProject.Name})
			if !ctx.dryRun {
				_, err := ctx.masterKubermaticClient.KubermaticV1().Users().Update(&user)
				if err != nil {
					glog.Errorf("failed to update user (ID = %s) object, however we can continue because the user object will be updated by the \"rbac-controller\" anyway, err = %v", user.Spec.ID, err)
				}
			} else {
				glog.Infof("a project (ID = %s, Name = %s) for userID = %s was NOT added to the \"Spec.Projects\" field because dry-run option was requested", project.Name, project.Spec.Name, user.Spec.ID)
			}
			ownersOfClusterWithProject[user.Spec.ID] = *project
		}
	}

	//
	// step 4: get ssh keys that are being used by clusters
	//         this steps uses clustersToAdoptByUserID to determine
	//         whether the given key is used by the cluster.
	//
	//         note that:
	//         the returned collection of keys is also grouped by userID
	sshKeysToAdoptByUserID := map[string][]kubermaticv1.UserSSHKey{}
	glog.Info("STEP 4: getting the list of ssh keys that are being used by a cluster")
	{
		sshKeys, err := ctx.masterKubermaticClient.KubermaticV1().UserSSHKeies().List(metav1.ListOptions{})
		if err != nil {
			return err
		}

		for _, sshKey := range sshKeys.Items {
			if len(sshKey.Spec.Owner) == 0 {
				glog.Warningf("cannot migrate the following ssh key (ID = %s, Name = %s), because it doesn't have an owner", sshKey.Name, sshKey.Spec.Name)
				continue
			}

			projectName := isOwnedByProject(sshKey.OwnerReferences)
			if len(projectName) > 0 {
				glog.V(3).Infof("skipping the following ssh keys (ID = %s, Name = %s) as it already belongs to project = %s", sshKey.Name, sshKey.Spec.Name, projectName)
				continue
			}

			// take only the keys that are being used by a cluster
			if _, ok := clustersToAdoptByUserID[sshKey.Spec.Owner]; ok {
				userSSHKeys := sshKeysToAdoptByUserID[sshKey.Spec.Owner]
				userSSHKeys = append(userSSHKeys, sshKey)
				sshKeysToAdoptByUserID[sshKey.Spec.Owner] = userSSHKeys
				glog.V(2).Infof("adding the ssh keys (ID = %s, Name = %s) to the list", sshKey.Name, sshKey.Spec.Name)
			} else {
				glog.V(3).Infof("skipping the following ssh keys (ID = %s, Name = %s) as it is not being used by any cluster", sshKey.Name, sshKey.Spec.Name)
			}
		}
	}
	printSSHKeysToAdopt("user", sshKeysToAdoptByUserID)

	// step 5: migrate cluster resources
	//         in order to add (migrate) a cluster to a project
	//         we need to add appropriate OwnerReference to a cluster resource
	glog.Info("STEP 5: migrating cluster resources")
	{
		for userID, clustersMap := range clustersToAdoptByUserID {
			for clusterName, clusterResourcesProviderTuple := range clustersMap {
				project := ownersOfClusterWithProject[userID]
				ownerRef := metav1.OwnerReference{
					APIVersion: kubermaticv1.SchemeGroupVersion.String(),
					Kind:       kubermaticv1.ProjectKindName,
					UID:        project.GetUID(),
					Name:       project.Name,
				}
				for _, clusterResource := range clusterResourcesProviderTuple.clusters {
					clusterResource.OwnerReferences = append(clusterResource.OwnerReferences, ownerRef)
					if !ctx.dryRun {
						kubermaticClient := clusterResourcesProviderTuple.provider.kubermaticClient
						_, err := kubermaticClient.KubermaticV1().Clusters().Update(&clusterResource)
						if err != nil {
							return err
						}
						glog.Infof("the cluster = %s resource for userID = %s was migrated to the project = %s, physical location = %s", clusterResource.Name, userID, project.Name, clusterName)
					} else {
						glog.Infof("the cluster resource (Name = %s) for userID = %s was NOT migrated because dry-run option was requested, physical location = %s", clusterResource.Name, userID, clusterName)
					}
				}
			}
		}
	}

	// step 6: migrate ssh keys
	//         in order to add (migrate) an ssh key to a project
	//         we need to add appropriate OwnerReference to an ssk key resource
	glog.Info("STEP 6: migrating ssh keys")
	{
		for userID, sshKeys := range sshKeysToAdoptByUserID {
			for _, sshKey := range sshKeys {
				project := ownersOfClusterWithProject[userID]
				ownerRef := createOwnerReferenceForProject(project)
				sshKey.OwnerReferences = append(sshKey.OwnerReferences, ownerRef)
				if !ctx.dryRun {
					_, err := ctx.masterKubermaticClient.KubermaticV1().UserSSHKeies().Update(&sshKey)
					if err != nil {
						return err
					}
					glog.Infof("the ssh key = %s for userID = %s was migrated to the project = %s", sshKey.Name, userID, project.Name)
				} else {
					glog.Infof("the ssh key (ID = %s Name = %s) for userID = %s was NOT migrated because dry-run option was requested", sshKey.Name, sshKey.Spec.Name, userID)
				}
			}
		}
	}

	return nil
}

type clustersProviderTuple struct {
	clusters []kubermaticv1.Cluster
	provider *clusterProvider
}

// getAllClusters gets all clusters resources in the system and groups them by user and cluster (physical location)
func getAllClusters(ctx migrationContext) (map[string]map[string]*clustersProviderTuple, error) {
	// clustersToAdoptByUserID structure that groups cluster resources by user and physical location
	clustersToAdoptByUserID := map[string]map[string]*clustersProviderTuple{}

	// helper is a helper method that adds the given cluster to the list of clusters
	// grouped by the user's ID and physical location
	helper := func(cluster kubermaticv1.Cluster, provider *clusterProvider) {
		userClustersMap := clustersToAdoptByUserID[cluster.Labels["user"]]

		if userClustersMap == nil {
			userClustersMap = make(map[string]*clustersProviderTuple)
		}

		userClusterResourcesTuple := userClustersMap[provider.name]
		if userClusterResourcesTuple == nil {
			userClusterResourcesTuple = &clustersProviderTuple{}
			userClusterResourcesTuple.provider = provider
		}

		userClusterResourcesTuple.clusters = append(userClusterResourcesTuple.clusters, cluster)
		userClustersMap[provider.name] = userClusterResourcesTuple
		clustersToAdoptByUserID[cluster.Labels["user"]] = userClustersMap
	}

	// get cluster resources that are located in master cluster
	{
		masterClusters, err := ctx.masterKubermaticClient.KubermaticV1().Clusters().List(metav1.ListOptions{})
		if err != nil {
			return nil, err
		}

		for _, cluster := range masterClusters.Items {
			projectName := isOwnedByProject(cluster.GetOwnerReferences())
			if len(projectName) == 0 {
				helper(cluster, &clusterProvider{"master", ctx.masterKubeClient, ctx.masterKubermaticClient})
			}
		}
	}

	// get cluster resources that are located in seed clusters
	{
		for _, seedClusterProvider := range ctx.seedClusterProviders {

			seedClusters, err := seedClusterProvider.kubermaticClient.KubermaticV1().Clusters().List(metav1.ListOptions{})
			if err != nil {
				return nil, err
			}

			for _, cluster := range seedClusters.Items {
				projectName := isOwnedByProject(cluster.GetOwnerReferences())
				if len(projectName) == 0 {
					helper(cluster, seedClusterProvider)
				}
			}
		}
	}

	return clustersToAdoptByUserID, nil
}

// isOwnedByProject is a helper function that extract projectName from the given OwnerReferences
func isOwnedByProject(owners []metav1.OwnerReference) string {
	for _, owner := range owners {
		if owner.APIVersion == kubermaticv1.SchemeGroupVersion.String() && owner.Kind == kubermaticv1.ProjectKindName &&
			len(owner.Name) > 0 && len(owner.UID) > 0 {
			return owner.Name
		}
	}
	return ""
}

func isOwnedByUser(owners []metav1.OwnerReference) string {
	for _, owner := range owners {
		if owner.APIVersion == kubermaticv1.SchemeGroupVersion.String() && owner.Kind == kubermaticv1.UserKindName &&
			len(owner.Name) > 0 && len(owner.UID) > 0 {
			return owner.Name
		}
	}
	return ""
}

// printClusterToAdopt prints cluster resources to stdout
func printClusterToAdopt(clustersToAdoptByUserID map[string]map[string]*clustersProviderTuple) {
	for userID, clustersMap := range clustersToAdoptByUserID {
		glog.V(2).Info("==================================================================================================")
		glog.V(2).Infof("clusters that will be migrated for user with ID = %s:", userID)
		glog.V(2).Info("==================================================================================================")
		for clusterName, clusterResourcesTuple := range clustersMap {
			glog.V(2).Infof("%1s there are %d clusters resources in \"%s\" cluster (physical location)", "", len(clusterResourcesTuple.clusters), clusterName)
			for index, clusterResource := range clusterResourcesTuple.clusters {
				glog.V(3).Infof("%2s %d. name = %s", "", index+1, clusterResource.Name)
			}
		}
	}
}

// printSSHKeysToAdopt prints ssh keys resources to stdout
func printSSHKeysToAdopt(keyName string, sshKeysToAdoptByID map[string][]kubermaticv1.UserSSHKey) {
	if len(sshKeysToAdoptByID) == 0 {
		glog.V(2).Infof("there are not ssh keys to migrate for %s(s)", keyName)
		return
	}
	for key, sshKeysResources := range sshKeysToAdoptByID {
		glog.V(2).Info("==================================================================================================")
		glog.V(2).Infof("ssh keys that will be migrated for %s with ID = %s:", keyName, key)
		glog.V(2).Info("==================================================================================================")
		glog.V(2).Infof("%1s there are %d ssh key resources", "", len(sshKeysResources))
		for index, sshKey := range sshKeysResources {
			glog.V(3).Infof("%2s %d. name = %s", "", index+1, sshKey.Name)
		}
	}
}

func printDuplicatedUsers(users map[string][]kubermaticv1.User) {
	glog.V(2).Info("==================================================================================================")
	glog.V(2).Info("duplicated users in the system")
	glog.V(2).Info("==================================================================================================")
	for userKey, userList := range users {
		if len(userList) > 1 {
			glog.V(2).Infof("%1s there are %d users with the same key %s", "", len(userList), userKey)
			for index, duplicatedUser := range userList {
				glog.V(3).Infof("%2s %d. user (%s) has Name = %s, Email = %s and ID = %s, the account was created on %s", "", index+1, duplicatedUser.Name, duplicatedUser.Spec.Name, duplicatedUser.Spec.Email, duplicatedUser.Spec.ID, duplicatedUser.CreationTimestamp.String())
			}
		}
	}
}

func createOwnerReferenceForProject(project kubermaticv1.Project) metav1.OwnerReference {
	return metav1.OwnerReference{
		APIVersion: kubermaticv1.SchemeGroupVersion.String(),
		Kind:       kubermaticv1.ProjectKindName,
		UID:        project.GetUID(),
		Name:       project.Name,
	}
}
