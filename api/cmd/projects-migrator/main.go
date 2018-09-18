package main

import (
	"errors"
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
	var removeDupUsers bool
	ctx := migrationContext{}

	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	flag.BoolVar(&ctx.dryRun, "dry-run", true, "If true, only print the object that would be created, without creating them")
	flag.BoolVar(&removeDupUsers, "rm-dup-users", false, "If true, only removes duplicated users and exits")
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

			glog.V(2).Infof("Adding %s as seed cluster", ctxName)
			kubeClient := kubernetes.NewForConfigOrDie(cfg)
			kubermaticClient := kubermaticclientset.NewForConfigOrDie(cfg)
			clusterProviders = append(clusterProviders, &clusterProvider{fmt.Sprintf("seed/%s", ctxName), kubeClient, kubermaticClient})
		}
	}

	ctx.masterKubeClient = kubernetes.NewForConfigOrDie(ctx.config)
	ctx.masterKubermaticClient = kubermaticclientset.NewForConfigOrDie(ctx.config)
	ctx.seedClusterProviders = clusterProviders

	// special case
	// if this flag was set then remove duplicates for users and exit the app
	if removeDupUsers {
		err := removeOrDetectDuplicatedUsers(ctx, "-1", false)
		if err != nil {
			glog.Fatalf("failed to remove duplicates for users, due to = %v", err)
		}
		glog.Info("successfully removed duplicates for users")
		return
	}

	// phase 0: stop when duplicated users
	//
	//           all duplicates were removed manually,
	//           that means our API server allowed to create a duplicate for a users
	//           please provide a fix, then remove duplicated users and then rerun the app
	//
	//           note:
	//           duplicated users have the same Spec.ID, Spec.Email, Spec.Name
	//           for example on dev environment: user-lknc7, user-n45j9, user-z9s20j
	err = removeOrDetectDuplicatedUsers(ctx, "0", true)
	if err != nil {
		glog.Fatalf(`PHASE 0 failed: due to = %v, please provide a fix to our API server, then remove duplicates for users (set rm-dup-users flag to true), and then rerun the app`, err)
	}

	// phase 1: migrate existing cluster resources along with ssh keys they use to projects
	err = migrateToProject(ctx)
	if err != nil {
		glog.Fatalf("PHASE 1 failed due to = %v", err)
	}

	// phase 2: migrate the remaining ssh keys to a project
	//
	//          note that:
	//          the remaining ssh keys are the ones that are owned by
	//          the project owner and were not used by running cluster (phase 1)
	err = migrateRemainingSSHKeys(ctx)
	if err != nil {
		glog.Fatalf("PHASE 2 failed due to %v", err)
	}

	// TODO:
	// phase 3 clean up
	//         - remove "user" label for cluster resources - that belong to a project
	//         - remove "owner" field for ssh keys resources - that belong to a project
	//
	//         note that:
	//         this step essentially breaks backward compatibility and prevents the old clients (dashboard) from finding the resources.

	// phase 4 remove ssh keys without an owner
	//
	//         the keys that don't have an owner have their sshKey.Spec.Owner field empty
	//         for example: key-8036218bef587f8230dad6426099da14-c7i4 and key-8036218bef587f8230dad6426099da14-wwk5 on dev env
	err = removeKeysWithoutOwner(ctx)
	if err != nil {
		glog.Errorf("PHASE 4 failed due to %v", err)
	}

	// phase 5: find clusters that are assigned to a project
	//          and replace OwnerReferences with a label
	err = remigrate(ctx)
	if err != nil {
		glog.Errorf("PHASE 5 failed due to %v", err)
	}

	// phase 6: move bindings to projects users belong to
	//          bindings are no longer stored under user.Projects field
	//          instead they are stored in a dedicated resource
	err = moveBindings(ctx)
	if err != nil {
		glog.Errorf("PHASE 6 failed due to %v", err)
	}
}

// moveBindings moves bindings to projects users belong to
// bindings are no longer stored under user.Projects field
// instead they are stored in a dedicated resource
func moveBindings(ctx migrationContext) error {
	glog.Info("\n")
	glog.Info("Running PHASE 6 ...")

	glog.Info("STEP 1: getting the list of users in the system")
	allUsers, err := ctx.masterKubermaticClient.KubermaticV1().Users().List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	glog.Info("STEP 2: getting the list of existing bindings")
	allBindings, err := ctx.masterKubermaticClient.KubermaticV1().UserProjectBindings().List(metav1.ListOptions{})
	if err != nil {
		return err
	}

	bindingForProjectExists := func(userEmail string, pg kubermaticv1.ProjectGroup) bool {
		for _, binding := range allBindings.Items {
			if binding.Spec.ProjectID == pg.Name && binding.Spec.Group == pg.Group && binding.Spec.UserEmail == userEmail {
				glog.V(3).Infof("the binding Name = %s, UserEmail = %s, ProjectID = %s, Group = %s already exists", binding.Name, binding.Spec.UserEmail, binding.Spec.ProjectID, binding.Spec.Group)
				return true
			}
		}
		return false
	}

	glog.Info("STEP 2: migrating bindings (if any)")
	for _, user := range allUsers.Items {
		for _, pg := range user.Spec.Projects {
			if !bindingForProjectExists(user.Spec.Email, pg) {
				// get the project
				project, err := ctx.masterKubermaticClient.KubermaticV1().Projects().Get(pg.Name, metav1.GetOptions{})
				if err != nil {
					return err
				}

				// create binding
				binding := &kubermaticv1.UserProjectBinding{
					ObjectMeta: metav1.ObjectMeta{
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ProjectKindName,
								UID:        project.GetUID(),
								Name:       project.Name,
							},
						},
						Name: rand.String(10),
					},
					Spec: kubermaticv1.UserProjectBindingSpec{
						UserEmail: user.Spec.Email,
						Group:     pg.Group,
						ProjectID: pg.Name,
					},
				}

				if !ctx.dryRun {
					_, err := ctx.masterKubermaticClient.KubermaticV1().UserProjectBindings().Create(binding)
					if err != nil {
						return err
					}
					glog.Infof("the binding Name = %s, UserEmail = %s, ProjectID = %s, Group = %s was created", binding.Name, binding.Spec.UserEmail, binding.Spec.ProjectID, binding.Spec.Group)
				} else {
					glog.Infof("the binding Name = %s, UserEmail = %s, ProjectID = %s, Group = %s was not created because dry-run option was requested", binding.Name, binding.Spec.UserEmail, binding.Spec.ProjectID, binding.Spec.Group)
				}
			}
		}

		// all bindings were created, clear Projects field
		if len(user.Spec.Projects) > 0 {
			user.Spec.Projects = []kubermaticv1.ProjectGroup{}
			if !ctx.dryRun {
				_, err := ctx.masterKubermaticClient.KubermaticV1().Users().Update(&user)
				if err != nil {
					return err
				}
				glog.Infof("the user ID = %s, Email = %s resource was updated", user.Name, user.Spec.Email)
			} else {
				glog.Infof("the user ID = %s, Email = %s, resource was not updated because dry-run option was requested", user.Name, user.Spec.Email)
			}
		}
	}
	return nil
}

// remigrate essentially removes existing OwnerReferences and replaces them with a label
// see also: https://github.com/kubermatic/kubermatic/pull/1839
func remigrate(ctx migrationContext) error {
	glog.Info("\n")
	glog.Info("Running PHASE 5 ...")
	//
	// step 1: get clusters that already belong to a project,
	//         clusters like that have OwnerReferences set for a project
	//
	//         note that this step will get clusters resources that belong to
	//         seed clusters (physical location)
	glog.Info("STEP 1: getting the list of clusters that were migrated")
	_, alreadyAdoptedClusters, err := getAllClusters(ctx)
	if err != nil {
		return err
	}

	glog.Info("STEP 2: remigrating the cluster resources (if any)")
	for _, providerClustersTuple := range alreadyAdoptedClusters {
		provider := providerClustersTuple.provider
		for _, cluster := range providerClustersTuple.clusters {
			projectName := isOwnedByProject(cluster.GetOwnerReferences(), nil)
			if len(projectName) > 0 {
				cluster.OwnerReferences = removeOwnerReferencesForProject(cluster.GetOwnerReferences(), projectName)
				cluster.Labels[kubermaticv1.ProjectIDLabelKey] = projectName
				if !ctx.dryRun {
					_, err := provider.kubermaticClient.KubermaticV1().Clusters().Update(&cluster)
					if err != nil {
						return err
					}
					glog.Infof("the cluster ID = %s, Name = %s, project = %s, physical location %s was remigrated", cluster.Name, cluster.Spec.HumanReadableName, projectName, provider.name)
				} else {
					glog.Infof("the cluster ID = %s, Name = %s, project = %s, physical location %s, was not remigrated because dry-run option was requested", cluster.Name, cluster.Spec.HumanReadableName, projectName, provider.name)
				}
			}
		}
	}
	return nil
}

func removeKeysWithoutOwner(ctx migrationContext) error {
	glog.Info("\n")
	glog.Infof("Running PHASE 4 ...")

	keysWithoutOwner := []kubermaticv1.UserSSHKey{}
	glog.Info("STEP 1: getting the list of keys that are owned by a project owner")
	{
		allKeys, err := ctx.masterKubermaticClient.KubermaticV1().UserSSHKeies().List(metav1.ListOptions{})
		if err != nil {
			return err
		}

		for _, key := range allKeys.Items {
			if sshAlreadyMigratedWithLog(key) {
				continue
			}
			if len(key.Spec.Owner) == 0 {
				keysWithoutOwner = append(keysWithoutOwner, key)
			}
		}
	}

	glog.Info("STEP 2: removing the keys without an owner (if any)")
	{
		for _, keyToRemove := range keysWithoutOwner {
			if !ctx.dryRun {
				err := ctx.masterKubermaticClient.KubermaticV1().UserSSHKeies().Delete(keyToRemove.Name, &metav1.DeleteOptions{})
				if err != nil {
					return err
				}
				glog.Infof("the ssh key = %s was removed", keyToRemove.Name)
			} else {
				glog.Infof("the ssh key = %s was NOT removed because dry-run option was requested", keyToRemove.Name)
			}
		}
	}

	return nil
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
					projectOwnersTuple[user.Spec.ID] = projectOwnerTuple{project: project, owner: *user}
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
			if sshAlreadyMigratedWithLog(key) {
				continue
			}
			if len(key.Spec.Owner) == 0 {
				glog.Warningf("the key ID = %s, Name = %s doesn't have an owner", key.Name, key.Spec.Name)
				continue
			}
			if projectOwner, ok := projectOwnersTuple[key.Spec.Owner]; ok {
				keysProjectTuple = append(keysProjectTuple, keyProjectTuple{key: key, project: projectOwner.project})
			} else {
				glog.V(2).Infof("the owner = %s of the key ID = %s, Name = %s doesn't have a project", key.Spec.Owner, key.Name, key.Spec.Name)
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

// removeOrDetectDuplicatedUsers finds users with the same spec.ID, spec.Email and spec.Name
// and removes duplication. This is safe to do because resources like clusters are bound to the user
// if there is a match on spec.ID field.
//
// note: if the stopOnDuplicated arg is set to true then this function exits when duplicates are detected
func removeOrDetectDuplicatedUsers(ctx migrationContext, phase string, stopOnDuplicates bool) error {
	glog.Infof("Running PHASE %s ...", phase)
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

	step2Msg := "STEP 2: removing duplicated users (if any)"
	if stopOnDuplicates {
		step2Msg = "STEP 2: detecting duplicated users (if any)"
	}

	glog.Info(step2Msg)
	{
		for _, userList := range seenUsers {
			if len(userList) > 1 {

				if stopOnDuplicates {
					return errors.New("duplicated users detected")
				}

				seenUserWithProjects := false
				for i := 0; i < len(userList)-1; i++ {
					user := userList[i]
					projectName, err := doesUserOwnProjectAs(user, ctx, "")
					if err != nil {
						return err
					}
					if len(projectName) > 0 {
						if seenUserWithProjects {
							return errors.New("there is more that one user that belongs to some projects fot the given key, please manually remove one of them and rerun the app")
						}
						glog.Warningf("cannot remove the user = %s because it already belongs to some projects = %v", user.Name, user.Spec.Projects)
						seenUserWithProjects = true
						continue
					}
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
	//         seed clusters (physical location)
	glog.Info("STEP 1: getting the list of clusters that needs to be migrated")
	clustersToAdoptByUserID, _, err := getAllClusters(ctx)
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
			projectName, err := doesUserOwnProjectAs(user, ctx, rbac.OwnerGroupNamePrefix)
			if err != nil {
				return err
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
				glog.Infof("a project (ID = %s, Name = %s) for user Spec.ID = %s, ID = %s was NOT created because dry-run option was requested", project.Name, project.Spec.Name, user.Spec.ID, user.Name)
				createdProject = project
			}

			user.Spec.Projects = append(user.Spec.Projects, kubermaticv1.ProjectGroup{Group: rbac.GenerateActualGroupNameFor(createdProject.Name, rbac.OwnerGroupNamePrefix), Name: createdProject.Name})
			if !ctx.dryRun {
				_, err := ctx.masterKubermaticClient.KubermaticV1().Users().Update(&user)
				if err != nil {
					return fmt.Errorf("failed to update user (ID = %s) object, err = %v", user.Spec.ID, err)
				}
			} else {
				glog.Infof("a project (ID = %s, Name = %s) for user Spec.ID = %s, ID = %s was NOT added to the \"Spec.Projects\" field because dry-run option was requested", project.Name, project.Spec.Name, user.Spec.ID, user.Name)
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
			if sshAlreadyMigratedWithLog(sshKey) {
				continue
			}
			if len(sshKey.Spec.Owner) == 0 {
				glog.Warningf("cannot migrate the following ssh key (ID = %s, Name = %s), because it doesn't have an owner", sshKey.Name, sshKey.Spec.Name)
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
				project, projectExists := ownersOfClusterWithProject[userID]
				for _, clusterResource := range clusterResourcesProviderTuple.clusters {
					if !projectExists {
						glog.Warningf("skipping cluster resource = %s (physical location %s), there is no project for a user with ID = %s (the user doesn't exist ?)", clusterResource.Name, clusterName, userID)
						continue
					}
					clusterResource.Labels[kubermaticv1.ProjectIDLabelKey] = project.Name
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
				project, ok := ownersOfClusterWithProject[userID]
				if !ok {
					glog.Warningf("skipping the ssh key = %s, there is no project for a user with ID = %s (the user doesn't exist ?)", sshKey.Name, userID)
					continue
				}
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
func getAllClusters(ctx migrationContext) (map[string]map[string]*clustersProviderTuple, []*clustersProviderTuple, error) {
	// clustersToAdoptByUserID structure that groups cluster resources by user and physical location
	clustersToAdoptByUserID := map[string]map[string]*clustersProviderTuple{}
	alreadyAdoptedClusters := []*clustersProviderTuple{}

	// helper is a helper method that adds the given cluster to the list of clusters
	// grouped by the user's ID and physical location
	helper := func(cluster kubermaticv1.Cluster, provider *clusterProvider) {
		if val, ok := cluster.Labels["user"]; !ok || len(val) == 0 {
			glog.Warningf("the cluster ID = %s, Name = %s doesn't have an owner (this might be okay, e2e tests ?)", cluster.Name, cluster.Spec.HumanReadableName)
			return
		}
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

	// get cluster resources that are located in seed clusters
	{
		for _, seedClusterProvider := range ctx.seedClusterProviders {

			seedClusters, err := seedClusterProvider.kubermaticClient.KubermaticV1().Clusters().List(metav1.ListOptions{})
			if err != nil {
				return nil, nil, err
			}

			alreadyAdoptedOnSeed := []kubermaticv1.Cluster{}

			for _, cluster := range seedClusters.Items {
				projectName := isOwnedByProject(cluster.GetOwnerReferences(), cluster.Labels)
				if len(projectName) == 0 {
					helper(cluster, seedClusterProvider)
				} else {
					alreadyAdoptedOnSeed = append(alreadyAdoptedOnSeed, cluster)
				}
			}
			alreadyAdoptedClusters = append(alreadyAdoptedClusters, &clustersProviderTuple{clusters: alreadyAdoptedOnSeed, provider: seedClusterProvider})
		}
	}

	return clustersToAdoptByUserID, alreadyAdoptedClusters, nil
}

// isOwnedByProject is a helper function that extract projectName from the given OwnerReferences
func isOwnedByProject(owners []metav1.OwnerReference, labels map[string]string) string {
	for _, owner := range owners {
		if owner.APIVersion == kubermaticv1.SchemeGroupVersion.String() && owner.Kind == kubermaticv1.ProjectKindName &&
			len(owner.Name) > 0 && len(owner.UID) > 0 {
			return owner.Name
		}
	}
	if projectName := labels[kubermaticv1.ProjectIDLabelKey]; len(projectName) > 0 {
		return projectName
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

func removeOwnerReferencesForProject(owners []metav1.OwnerReference, projectName string) []metav1.OwnerReference {
	newOwners := []metav1.OwnerReference{}
	for _, owner := range owners {
		if owner.APIVersion == kubermaticv1.SchemeGroupVersion.String() && owner.Kind == kubermaticv1.ProjectKindName &&
			owner.Name == projectName {
			continue
		}
		newOwners = append(newOwners, owner)
	}

	return newOwners
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

func sshAlreadyMigratedWithLog(sshKey kubermaticv1.UserSSHKey) bool {
	projectName := isOwnedByProject(sshKey.OwnerReferences, nil)
	if len(projectName) > 0 {
		glog.V(3).Infof("skipping the following ssh keys (ID = %s, Name = %s) as it already belongs to project = %s", sshKey.Name, sshKey.Spec.Name, projectName)
		return true
	}

	return false
}

// doesUserOwnProjectAs check if the given user owns a project and belongs to group specified by groupPrefix argument
// if the argument is an empty string they this method returns any project the user belongs to
func doesUserOwnProjectAs(user kubermaticv1.User, ctx migrationContext, groupPrefix string) (string, error) {
	// if the users owns more than one project take the first one
	for _, projectGroup := range user.Spec.Projects {
		if rbac.ExtractGroupPrefix(projectGroup.Group) == groupPrefix {
			return projectGroup.Name, nil
		} else if groupPrefix == "" {
			return projectGroup.Name, nil
		}
	}

	// if the project id was not found in Spec.Projects field then check the bindings
	allBindings, err := ctx.masterKubermaticClient.KubermaticV1().UserProjectBindings().List(metav1.ListOptions{})
	if err != nil {
		return "", err
	}
	for _, binding := range allBindings.Items {
		if binding.Spec.UserEmail == user.Spec.Email {
			if rbac.ExtractGroupPrefix(binding.Spec.Group) == rbac.OwnerGroupNamePrefix {
				return binding.Spec.ProjectID, nil
			} else if groupPrefix == "" {
				return binding.Spec.ProjectID, nil
			}
		}
	}

	// the user doesn't have a project
	return "", nil
}
