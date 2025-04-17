/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package images

import (
	"archive/tar"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	semverlib "github.com/Masterminds/semver/v3"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/sirupsen/logrus"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	ksemver "k8c.io/kubermatic/sdk/v2/semver"
	"k8c.io/kubermatic/v2/pkg/addon"
	"k8c.io/kubermatic/v2/pkg/cni"
	"k8c.io/kubermatic/v2/pkg/controller/operator/common/vpa"
	masteroperator "k8c.io/kubermatic/v2/pkg/controller/operator/master/resources/kubermatic"
	seedoperatorkubermatic "k8c.io/kubermatic/v2/pkg/controller/operator/seed/resources/kubermatic"
	"k8c.io/kubermatic/v2/pkg/controller/operator/seed/resources/metering"
	seedoperatornodeportproxy "k8c.io/kubermatic/v2/pkg/controller/operator/seed/resources/nodeportproxy"
	kubernetescontroller "k8c.io/kubermatic/v2/pkg/controller/seed-controller-manager/kubernetes"
	"k8c.io/kubermatic/v2/pkg/controller/seed-controller-manager/mla"
	"k8c.io/kubermatic/v2/pkg/controller/seed-controller-manager/monitoring"
	envoyagent "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/envoy-agent"
	"k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/gatekeeper"
	"k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/konnectivity"
	k8sdashboard "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/kubernetes-dashboard"
	nodelocaldns "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/node-local-dns"
	"k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/usersshkeys"
	"k8c.io/kubermatic/v2/pkg/defaulting"
	"k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/cloudcontroller"
	"k8c.io/kubermatic/v2/pkg/resources/csi/vmwareclouddirector"
	"k8c.io/kubermatic/v2/pkg/resources/operatingsystemmanager"
	"k8c.io/kubermatic/v2/pkg/resources/registry"
	"k8c.io/kubermatic/v2/pkg/test/fake"
	"k8c.io/kubermatic/v2/pkg/version"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"
	"k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
	"k8s.io/utils/ptr"
)

const mockNamespaceName = "mock-namespace"

type ImageSourceDest struct {
	Source      string
	Destination string
}

func GetImageSourceDestList(ctx context.Context, log logrus.FieldLogger, images []string, registry string) ([]ImageSourceDest, error) {
	var retaggedImages []ImageSourceDest
	for _, image := range images {
		retaggedImage, err := RewriteImage(log, image, registry)
		if err != nil {
			return nil, fmt.Errorf("failed to rewrite %q: %w", image, err)
		}

		retaggedImages = append(retaggedImages, retaggedImage)
	}

	return retaggedImages, nil
}

func RewriteImage(log logrus.FieldLogger, sourceImage, registry string) (ImageSourceDest, error) {
	imageRef, err := name.ParseReference(sourceImage)
	if err != nil {
		return ImageSourceDest{}, fmt.Errorf("failed to parse image: %w", err)
	}

	targetImage := fmt.Sprintf("%s/%s:%s", registry, imageRef.Context().RepositoryStr(), imageRef.Identifier())

	// if the image reference includes a digest, we strip the digest from the image name.
	// since crane.Copy preserves digests, it's enough to keep it in the source image.
	if _, ok := imageRef.(name.Digest); ok {
		if index := strings.Index(sourceImage, "@"); index > 0 {
			digestLessImage := sourceImage[:index]
			imageRef, err = name.ParseReference(digestLessImage)
			if err != nil {
				return ImageSourceDest{}, fmt.Errorf("failed to parse image without digest part: %w", err)
			}
		}

		targetImage = fmt.Sprintf("%s/%s:%s", registry, imageRef.Context().RepositoryStr(), imageRef.Identifier())
	}

	fields := logrus.Fields{
		"source-image": sourceImage,
		"target-image": targetImage,
	}

	log.WithFields(fields).Info("Image found")

	return ImageSourceDest{
		Source:      sourceImage,
		Destination: targetImage,
	}, nil
}

func ExtractAddons(ctx context.Context, log logrus.FieldLogger, addonImageName string) (string, error) {
	tempDir, err := os.MkdirTemp("", "kkp-mirror-images-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary directory: %w", err)
	}

	image, err := crane.Pull(addonImageName, crane.WithContext(ctx))
	if err != nil {
		return "", fmt.Errorf("failed to fetch remote image: %w", err)
	}

	exportFilePath := filepath.Join(tempDir, "image.tar")
	exportFile, err := os.Create(exportFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to create %s: %w", exportFilePath, err)
	}
	defer exportFile.Close()

	log.WithFields(logrus.Fields{
		"image":          addonImageName,
		"temp-directory": tempDir,
		"export-file":    exportFilePath,
	}).Debug("Exporting addon image to archive file…")

	if err := crane.Export(image, exportFile); err != nil {
		return "", fmt.Errorf("failed to export addon image to file: %w", err)
	}

	reader, err := os.Open(exportFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to open %s: %w", exportFilePath, err)
	}
	defer reader.Close()

	tarReader := tar.NewReader(reader)
	if err := extractAddonsFromArchive(tempDir, tarReader); err != nil {
		return "", fmt.Errorf("failed to extract addons from archive: %w", err)
	}

	return filepath.Join(tempDir, "addons"), nil
}

func extractAddonsFromArchive(dir string, reader *tar.Reader) error {
	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		if header == nil {
			continue
		}

		path := filepath.Join(dir, header.Name)

		// prevent path traversal (https://cwe.mitre.org/data/definitions/22.html)
		if strings.Contains(path, "..") {
			continue
		}

		switch header.Typeflag {
		case tar.TypeDir:
			// we only want to extract the addons folder and thus we skip
			// everything else in the image archive.
			if strings.HasPrefix(header.Name, "addons") {
				if err = os.MkdirAll(path, 0o755); err != nil {
					return err
				}
			}
		case tar.TypeReg:
			// we only care about files in the addons folder.
			if strings.HasPrefix(header.Name, "addons/") {
				f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
				if err != nil {
					return err
				}

				if _, err := io.Copy(f, reader); err != nil {
					return err
				}

				f.Close()
			}
		}
	}

	return nil
}

func ArchiveImages(ctx context.Context, log logrus.FieldLogger, archivePath string, dryRun bool, images []string) (int, int, error) {
	srcToImage := make(map[string]v1.Image)
	for _, src := range images {
		log = log.WithFields(logrus.Fields{
			"image": src,
		})
		log.Info("Fetching image…")
		img, err := crane.Pull(src, crane.WithAuthFromKeychain(authn.DefaultKeychain), crane.WithContext(ctx))
		if err != nil {
			log.WithError(err).Error("Failed to fetch remote image. Skipping...")
			continue
		} else {
			log.Info("Image fetched.")

			// double check by loading the image fully (sometimes they timeout during saving)
			if _, err := img.RawManifest(); err != nil {
				log.WithError(err).Error("Failed to fetch manifest. Skipping...")
				continue
			}

			// all good with the image, let it be archived
			srcToImage[src] = img
		}
	}

	if dryRun {
		return len(images), len(images), nil
	}

	if len(srcToImage) == 0 {
		return 0, len(images), nil
	}
	log = log.WithFields(logrus.Fields{
		"archived-count": len(srcToImage),
	})
	log.Info("Saving images to archive…")
	if err := crane.MultiSave(srcToImage, archivePath); err != nil {
		return 0, 0, fmt.Errorf("failed to save images to archive: %w", err)
	}

	return len(srcToImage), len(images), nil
}

func pathOpener(path string) tarball.Opener {
	return func() (io.ReadCloser, error) {
		return os.Open(path)
	}
}

func LoadImages(ctx context.Context, log logrus.FieldLogger, archivePath string, dryRun bool, registry string, userAgent string) error {
	indexManifest, err := tarball.LoadManifest(pathOpener(archivePath))
	if err != nil {
		return fmt.Errorf("failed to load manifest: %w", err)
	}

	repositoryToRefToImage := make(map[string]map[name.Reference]remote.Taggable)
	for _, descriptor := range indexManifest {
		for _, tagStr := range descriptor.RepoTags {
			repoTag, err := name.NewTag(tagStr)
			if err != nil {
				return fmt.Errorf("failed to parse tag %s: %w", tagStr, err)
			}

			img, err := tarball.Image(pathOpener(archivePath), &repoTag)
			if err != nil {
				return fmt.Errorf("failed to load image %s from tarball: %w", tagStr, err)
			}

			imageSourceDest, err := RewriteImage(log, tagStr, registry)
			if err != nil {
				return fmt.Errorf("failed to rewrite %q: %w", tagStr, err)
			}

			ref, err := name.ParseReference(imageSourceDest.Destination, name.Insecure)
			if err != nil {
				return fmt.Errorf("failed to parse reference %s: %w", imageSourceDest.Destination, err)
			}

			if repositoryToRefToImage[ref.Context().RepositoryStr()] == nil {
				repositoryToRefToImage[ref.Context().RepositoryStr()] = make(map[name.Reference]remote.Taggable)
			}
			repositoryToRefToImage[ref.Context().RepositoryStr()][ref] = img
		}
	}
	if dryRun {
		return nil
	}

	// remote.MultiWrite only supports one repository at a time, so we need to iterate over all repositories
	for _, refToImage := range repositoryToRefToImage {
		err = remote.MultiWrite(refToImage, remote.WithAuthFromKeychain(authn.DefaultKeychain), remote.WithContext(ctx), remote.WithUserAgent(userAgent))
		if err != nil {
			return fmt.Errorf("failed to write images: %w", err)
		}
	}
	return nil
}

func CopyImages(ctx context.Context, log logrus.FieldLogger, dryRun bool, images []string, registry string, userAgent string) (int, int, error) {
	imageList, err := GetImageSourceDestList(ctx, log, images, registry)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to generate list of images: %w", err)
	}

	if dryRun {
		return 0, len(imageList), nil
	}

	var failedImages []string

	for index, image := range imageList {
		if err := copyImage(ctx, log.WithField("image", fmt.Sprintf("%d/%d", index+1, len(imageList))), image, userAgent); err != nil {
			log.Errorf("Failed to copy image: %v", err)
			failedImages = append(failedImages, fmt.Sprintf("  - %s", image.Source))
		}
	}

	successCount := len(imageList) - len(failedImages)
	if len(failedImages) > 0 {
		return successCount, len(imageList), fmt.Errorf("failed images:\n%s", strings.Join(failedImages, "\n"))
	}

	return successCount, len(imageList), nil
}

func copyImage(ctx context.Context, log logrus.FieldLogger, image ImageSourceDest, userAgent string) error {
	log = log.WithFields(logrus.Fields{
		"source-image": image.Source,
		"target-image": image.Destination,
	})

	options := []crane.Option{
		crane.WithContext(ctx),
		crane.WithUserAgent(userAgent),
	}

	log.Info("Copying image…")

	numTries := 0
	backoff := wait.Backoff{
		Steps:    3,
		Duration: 500 * time.Millisecond,
		Factor:   1.0,
		Jitter:   0.1,
	}

	retriable := func(err error) bool {
		numTries++
		log.Info("Retrying…")

		return numTries <= backoff.Steps
	}

	return retry.OnError(backoff, retriable, func() error {
		err := crane.Copy(image.Source, image.Destination, options...)
		if err != nil {
			log.Error("Copying image:", err)
		}

		return err
	})
}

func GetImagesForVersion(log logrus.FieldLogger, clusterVersion *version.Version, cloudSpec kubermaticv1.CloudSpec, cniPlugin *kubermaticv1.CNIPluginSettings, konnectivityEnabled bool, config *kubermaticv1.KubermaticConfiguration, addons map[string]*addon.Addon, kubermaticVersions kubermatic.Versions, caBundle resources.CABundle, registryPrefix string) (images []string, err error) {
	seed, err := defaulting.DefaultSeed(&kubermaticv1.Seed{}, config, zap.NewNop().Sugar())
	if err != nil {
		return nil, fmt.Errorf("failed to default Seed: %w", err)
	}

	templateData, err := getTemplateData(config, clusterVersion, cloudSpec, cniPlugin, konnectivityEnabled, kubermaticVersions, caBundle, seed)
	if err != nil {
		return nil, err
	}

	creatorImages, err := getImagesFromReconcilers(log, templateData, config, kubermaticVersions, seed)
	if err != nil {
		return nil, fmt.Errorf("failed to get images from internal creator functions: %w", err)
	}

	images = append(images, creatorImages...)

	addonImages, err := getImagesFromAddons(log, addons, templateData.Cluster())
	if err != nil {
		return nil, fmt.Errorf("failed to get images from addons: %w", err)
	}

	images = append(images, addonImages...)
	if backupImages, err := etcdBackupImages(config.Spec.SeedController); err != nil {
		return nil, fmt.Errorf("failed to get images from etcd backups: %w", err)
	} else {
		images = append(images, backupImages...)
	}

	if registryPrefix != "" {
		var filteredImages []string
		for _, image := range images {
			if strings.HasPrefix(image, registryPrefix) {
				filteredImages = append(filteredImages, image)
			}
		}

		images = filteredImages
	}

	return images, nil
}

func etcdBackupImages(configuration kubermaticv1.KubermaticSeedControllerConfiguration) ([]string, error) {
	var images []string

	if configuration.BackupStoreContainer == "" {
		configuration.BackupStoreContainer = defaulting.DefaultBackupStoreContainer
	}

	if image, err := kubernetes.ContainerFromString(configuration.BackupStoreContainer); err != nil {
		return nil, fmt.Errorf("failed to get backup store image: %w", err)
	} else {
		images = append(images, image.Image)
	}

	if configuration.BackupDeleteContainer == "" {
		configuration.BackupDeleteContainer = defaulting.DefaultBackupDeleteContainer
	}

	if image, err := kubernetes.ContainerFromString(configuration.BackupDeleteContainer); err != nil {
		return nil, fmt.Errorf("failed to get backup delete image: %w", err)
	} else {
		images = append(images, image.Image)
	}

	return images, nil
}

func getImagesFromReconcilers(_ logrus.FieldLogger, templateData *resources.TemplateData, config *kubermaticv1.KubermaticConfiguration, kubermaticVersions kubermatic.Versions, seed *kubermaticv1.Seed) (images []string, err error) {
	statefulsetReconcilers := kubernetescontroller.GetStatefulSetReconcilers(templateData, false, false, 0)
	statefulsetReconcilers = append(statefulsetReconcilers, monitoring.GetStatefulSetReconcilers(templateData)...)

	deploymentReconcilers := kubernetescontroller.GetDeploymentReconcilers(templateData, false, kubermaticVersions)
	deploymentReconcilers = append(deploymentReconcilers, monitoring.GetDeploymentReconcilers(templateData)...)
	deploymentReconcilers = append(deploymentReconcilers, masteroperator.APIDeploymentReconciler(config, "", kubermaticVersions))
	deploymentReconcilers = append(deploymentReconcilers, masteroperator.MasterControllerManagerDeploymentReconciler(config, "", kubermaticVersions))
	deploymentReconcilers = append(deploymentReconcilers, masteroperator.UIDeploymentReconciler(config, kubermaticVersions))
	deploymentReconcilers = append(deploymentReconcilers, seedoperatorkubermatic.SeedControllerManagerDeploymentReconciler("", kubermaticVersions, config, seed))
	deploymentReconcilers = append(deploymentReconcilers, seedoperatornodeportproxy.EnvoyDeploymentReconciler(config, seed, false, kubermaticVersions))
	deploymentReconcilers = append(deploymentReconcilers, seedoperatornodeportproxy.UpdaterDeploymentReconciler(config, seed, kubermaticVersions))
	deploymentReconcilers = append(deploymentReconcilers, vpa.AdmissionControllerDeploymentReconciler(config, kubermaticVersions))
	deploymentReconcilers = append(deploymentReconcilers, vpa.RecommenderDeploymentReconciler(config, kubermaticVersions))
	deploymentReconcilers = append(deploymentReconcilers, vpa.UpdaterDeploymentReconciler(config, kubermaticVersions))
	deploymentReconcilers = append(deploymentReconcilers, mla.GatewayDeploymentReconciler(templateData, nil))
	deploymentReconcilers = append(deploymentReconcilers, operatingsystemmanager.DeploymentReconciler(templateData))
	deploymentReconcilers = append(deploymentReconcilers, k8sdashboard.DeploymentReconciler(templateData.RewriteImage))
	deploymentReconcilers = append(deploymentReconcilers, gatekeeper.ControllerDeploymentReconciler(false, templateData.RewriteImage, nil))
	deploymentReconcilers = append(deploymentReconcilers, vmwareclouddirector.ControllerDeploymentReconciler(templateData))

	if templateData.Cluster().Spec.Features[kubermaticv1.ClusterFeatureExternalCloudProvider] {
		deploymentReconcilers = append(deploymentReconcilers, cloudcontroller.DeploymentReconciler(templateData))
	}

	if templateData.IsKonnectivityEnabled() {
		deploymentReconcilers = append(deploymentReconcilers, konnectivity.DeploymentReconciler(templateData.Cluster().Spec.Version, "dummy", 0, kubermaticv1.DefaultKonnectivityKeepaliveTime, registry.GetImageRewriterFunc(templateData.OverwriteRegistry), nil))
	}

	cronjobReconcilers := kubernetescontroller.GetCronJobReconcilers(templateData)
	if mcjr := metering.CronJobReconciler("reportName", kubermaticv1.MeteringReportConfiguration{}, "caBundleName", templateData.RewriteImage, seed); mcjr != nil {
		cronjobReconcilers = append(cronjobReconcilers, mcjr)
	}

	var daemonsetReconcilers []reconciling.NamedDaemonSetReconcilerFactory
	daemonsetReconcilers = append(daemonsetReconcilers, usersshkeys.DaemonSetReconciler(
		kubermaticVersions,
		templateData.RewriteImage,
	))
	daemonsetReconcilers = append(daemonsetReconcilers, nodelocaldns.DaemonSetReconciler(templateData.RewriteImage))
	daemonsetReconcilers = append(daemonsetReconcilers, envoyagent.DaemonSetReconciler(net.IPv4(0, 0, 0, 0), kubermaticVersions, "", templateData.RewriteImage))

	for _, creatorGetter := range statefulsetReconcilers {
		_, creator := creatorGetter()
		statefulset, err := creator(&appsv1.StatefulSet{})
		if err != nil {
			return nil, err
		}
		images = append(images, getImagesFromPodSpec(statefulset.Spec.Template.Spec)...)
	}

	for _, createFunc := range deploymentReconcilers {
		_, creator := createFunc()
		deployment, err := creator(&appsv1.Deployment{})
		if err != nil {
			return nil, err
		}
		images = append(images, getImagesFromPodSpec(deployment.Spec.Template.Spec)...)
	}

	for _, createFunc := range cronjobReconcilers {
		_, creator := createFunc()
		cronJob, err := creator(&batchv1.CronJob{})
		if err != nil {
			return nil, err
		}
		images = append(images, getImagesFromPodSpec(cronJob.Spec.JobTemplate.Spec.Template.Spec)...)
	}

	for _, createFunc := range daemonsetReconcilers {
		_, creator := createFunc()
		daemonset, err := creator(&appsv1.DaemonSet{})
		if err != nil {
			return nil, err
		}
		images = append(images, getImagesFromPodSpec(daemonset.Spec.Template.Spec)...)
	}

	// Add images for Enterprise Edition addons/components.
	additionalImages, err := getAdditionalImagesFromReconcilers(templateData)
	if err != nil {
		return nil, err
	}

	images = append(images, additionalImages...)
	return images, nil
}

func getImagesFromPodSpec(spec corev1.PodSpec) (images []string) {
	for _, initContainer := range spec.InitContainers {
		images = append(images, initContainer.Image)
	}

	for _, container := range spec.Containers {
		images = append(images, container.Image)
	}

	return images
}

func getTemplateData(config *kubermaticv1.KubermaticConfiguration, clusterVersion *version.Version, cloudSpec kubermaticv1.CloudSpec, cniPlugin *kubermaticv1.CNIPluginSettings, konnectivityEnabled bool, kubermaticVersions kubermatic.Versions, caBundle resources.CABundle, seed *kubermaticv1.Seed) (*resources.TemplateData, error) {
	// We need listers and a set of objects to not have our deployment/statefulset creators fail
	mockObjects := []runtime.Object{
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      resources.ApiserverServiceName,
				Namespace: mockNamespaceName,
			},
			Spec: corev1.ServiceSpec{
				Ports:     []corev1.ServicePort{{NodePort: 99}},
				ClusterIP: "192.0.2.10",
			},
		},
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      resources.OpenVPNServerServiceName,
				Namespace: mockNamespaceName,
			},
			Spec: corev1.ServiceSpec{
				Ports:     []corev1.ServicePort{{NodePort: 96}},
				ClusterIP: "192.0.2.2",
			},
		},
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      resources.DNSResolverServiceName,
				Namespace: mockNamespaceName,
			},
			Spec: corev1.ServiceSpec{
				Ports:     []corev1.ServicePort{{NodePort: 98}},
				ClusterIP: "192.0.2.11",
			},
		},
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      resources.KonnectivityProxyServiceName,
				Namespace: mockNamespaceName,
			},
			Spec: corev1.ServiceSpec{
				Ports:     []corev1.ServicePort{{Name: "secure", Port: 443, Protocol: corev1.ProtocolTCP, TargetPort: intstr.FromInt(8132)}},
				ClusterIP: "192.0.2.20",
			},
		},
	}

	datacenter := &kubermaticv1.Datacenter{
		Spec: kubermaticv1.DatacenterSpec{
			Anexia:              &kubermaticv1.DatacenterSpecAnexia{},
			Azure:               &kubermaticv1.DatacenterSpecAzure{},
			Hetzner:             &kubermaticv1.DatacenterSpecHetzner{},
			Kubevirt:            &kubermaticv1.DatacenterSpecKubevirt{},
			Nutanix:             &kubermaticv1.DatacenterSpecNutanix{},
			Openstack:           &kubermaticv1.DatacenterSpecOpenstack{},
			VMwareCloudDirector: &kubermaticv1.DatacenterSpecVMwareCloudDirector{},
			VSphere:             &kubermaticv1.DatacenterSpecVSphere{},
		},
	}

	clusterSemver, err := ksemver.NewSemver(clusterVersion.Version.String())
	if err != nil {
		return nil, err
	}

	fakeCluster := &kubermaticv1.Cluster{}
	fakeCluster.Labels = map[string]string{kubermaticv1.ProjectIDLabelKey: "project"}
	fakeCluster.Spec.Cloud = cloudSpec
	fakeCluster.Spec.Version = *clusterSemver
	fakeCluster.Spec.ClusterNetwork.Pods.CIDRBlocks = []string{"172.25.0.0/16"}
	fakeCluster.Spec.ClusterNetwork.Services.CIDRBlocks = []string{"10.10.10.0/24"}
	fakeCluster.Spec.ClusterNetwork.DNSDomain = "cluster.local"
	fakeCluster.Spec.ClusterNetwork.KonnectivityEnabled = ptr.To(konnectivityEnabled) //nolint:staticcheck
	fakeCluster.Spec.CNIPlugin = cniPlugin
	fakeCluster.Spec.Features = map[string]bool{kubermaticv1.ClusterFeatureEtcdLauncher: true}
	if enabled, exists := config.Spec.FeatureGates[kubermaticv1.ClusterFeatureEtcdLauncher]; exists && !enabled {
		fakeCluster.Spec.Features[kubermaticv1.ClusterFeatureEtcdLauncher] = false
	}
	fakeCluster.Spec.AuditLogging = &kubermaticv1.AuditLoggingSettings{
		Enabled: true,
	}

	if cloudSpec.Openstack != nil || cloudSpec.Hetzner != nil || cloudSpec.Azure != nil || cloudSpec.VSphere != nil || cloudSpec.Anexia != nil || cloudSpec.Kubevirt != nil {
		if fakeCluster.Spec.Features == nil {
			fakeCluster.Spec.Features = make(map[string]bool)
		}
		fakeCluster.Spec.Features[kubermaticv1.ClusterFeatureExternalCloudProvider] = true
	}

	fakeCluster.Spec.EnableUserSSHKeyAgent = ptr.To(true)
	fakeCluster.Spec.KubernetesDashboard = &kubermaticv1.KubernetesDashboard{
		Enabled: true,
	}

	fakeCluster.Status.NamespaceName = mockNamespaceName
	fakeCluster.Status.Versions.ControlPlane = *clusterSemver
	fakeCluster.Status.Versions.Apiserver = *clusterSemver
	fakeCluster.Status.Versions.ControllerManager = *clusterSemver
	fakeCluster.Status.Versions.Scheduler = *clusterSemver

	fakeDynamicClient := fake.NewClientBuilder().WithRuntimeObjects(mockObjects...).Build()

	meteringConfig := &kubermaticv1.MeteringConfiguration{
		RetentionDays:    defaulting.DefaultMeteringRetentionDays,
		StorageSize:      defaulting.DefaultMeteringStorageSize,
		StorageClassName: "mock-storage-class",
	}
	seed.Spec.Metering = meteringConfig

	return resources.NewTemplateDataBuilder().
		WithKubermaticConfiguration(config).
		WithContext(context.Background()).
		WithClient(fakeDynamicClient).
		WithCluster(fakeCluster).
		WithDatacenter(datacenter).
		WithSeed(seed).
		WithNodeAccessNetwork("192.0.2.0/24").
		WithEtcdDiskSize(resource.Quantity{}).
		WithKubermaticImage(defaulting.DefaultKubermaticImage).
		WithEtcdLauncherImage(defaulting.DefaultEtcdLauncherImage).
		WithDnatControllerImage(defaulting.DefaultDNATControllerImage).
		WithNetworkIntfMgrImage(defaulting.DefaultNetworkInterfaceManagerImage).
		WithBackupPeriod(20 * time.Minute).
		WithFailureDomainZoneAntiaffinity(false).
		WithVersions(kubermaticVersions).
		WithCABundle(caBundle).
		WithKonnectivityEnabled(konnectivityEnabled).
		Build(), nil
}

func GetVersions(log logrus.FieldLogger, config *kubermaticv1.KubermaticConfiguration, versionFilter string) ([]*version.Version, error) {
	var versions []*version.Version

	log = log.WithField("versions-filter", versionFilter)

	if config != nil {
		log.Debug("Loading versions")
		versions = getVersionsFromKubermaticConfiguration(config)
	}

	if versionFilter == "" {
		return versions, nil
	}

	log.Debug("Filtering versions")
	constraint, err := semverlib.NewConstraint(versionFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to parse version filter %q: %w", versionFilter, err)
	}

	var filteredVersions []*version.Version
	for _, ver := range versions {
		if constraint.Check(ver.Version) {
			filteredVersions = append(filteredVersions, ver)
		}
	}

	return filteredVersions, nil
}

func GetCloudSpecs() []kubermaticv1.CloudSpec {
	return []kubermaticv1.CloudSpec{
		{
			ProviderName: string(kubermaticv1.AlibabaCloudProvider),
			Alibaba: &kubermaticv1.AlibabaCloudSpec{
				AccessKeyID:     "fakeAccessKeyID",
				AccessKeySecret: "fakeAccessKeySecret",
			},
		},
		{
			ProviderName: string(kubermaticv1.AnexiaCloudProvider),
			Anexia: &kubermaticv1.AnexiaCloudSpec{
				Token: "fakeToken",
			},
		},
		{
			ProviderName: string(kubermaticv1.AWSCloudProvider),
			AWS: &kubermaticv1.AWSCloudSpec{
				AccessKeyID:     "fakeAccessKeyID",
				SecretAccessKey: "fakeSecretAccessKey",
			},
		},
		{
			ProviderName: string(kubermaticv1.AzureCloudProvider),
			Azure: &kubermaticv1.AzureCloudSpec{
				TenantID:       "fakeTenantID",
				SubscriptionID: "fakeSubscriptionID",
				ClientID:       "fakeClientID",
				ClientSecret:   "fakeClientSecret",
			},
		},
		{
			ProviderName: string(kubermaticv1.BringYourOwnCloudProvider),
			BringYourOwn: &kubermaticv1.BringYourOwnCloudSpec{},
		},
		{
			ProviderName: string(kubermaticv1.BaremetalCloudProvider),
			Baremetal:    &kubermaticv1.BaremetalCloudSpec{},
		},
		{
			ProviderName: string(kubermaticv1.EdgeCloudProvider),
			Edge:         &kubermaticv1.EdgeCloudSpec{},
		},
		{
			ProviderName: string(kubermaticv1.DigitaloceanCloudProvider),
			Digitalocean: &kubermaticv1.DigitaloceanCloudSpec{
				Token: "fakeToken",
			},
		},
		{
			ProviderName: string(kubermaticv1.GCPCloudProvider),
			GCP:          &kubermaticv1.GCPCloudSpec{},
		},
		{
			ProviderName: string(kubermaticv1.HetznerCloudProvider),
			Hetzner: &kubermaticv1.HetznerCloudSpec{
				Token:   "fakeToken",
				Network: "fakeNetwork",
			},
		},
		{
			ProviderName: string(kubermaticv1.KubevirtCloudProvider),
			Kubevirt: &kubermaticv1.KubevirtCloudSpec{
				Kubeconfig:    "fakeKubeconfig",
				CSIKubeconfig: "fakeKubeconfig",
			},
		},
		{
			ProviderName: string(kubermaticv1.NutanixCloudProvider),
			Nutanix:      &kubermaticv1.NutanixCloudSpec{},
		},
		{
			ProviderName: string(kubermaticv1.OpenstackCloudProvider),
			Openstack: &kubermaticv1.OpenstackCloudSpec{
				Domain:   "fakeDomain",
				Username: "fakeUsername",
				Password: "fakePassword",
			},
		},
		{
			ProviderName: string(kubermaticv1.PacketCloudProvider),
			Packet:       &kubermaticv1.PacketCloudSpec{},
		},
		{
			ProviderName: string(kubermaticv1.VMwareCloudDirectorCloudProvider),
			VMwareCloudDirector: &kubermaticv1.VMwareCloudDirectorCloudSpec{
				Username:     "fakeUsername",
				Password:     "fakePassword",
				Organization: "fakeOrganization",
				VDC:          "fakeVDC",
			},
		},
		{
			ProviderName: string(kubermaticv1.VSphereCloudProvider),
			VSphere:      &kubermaticv1.VSphereCloudSpec{},
		},
	}
}

// list all the supported CNI plugins along with their supported versions.
func GetCNIPlugins() []*kubermaticv1.CNIPluginSettings {
	cniPluginSettings := []*kubermaticv1.CNIPluginSettings{}
	supportedCNIPlugins := cni.GetSupportedCNIPlugins()

	for _, cniPlugin := range sets.List(supportedCNIPlugins) {
		// error cannot ever occur since we just listed the supported CNIPluginTypes
		versions, _ := cni.GetAllowedCNIPluginVersions(kubermaticv1.CNIPluginType(cniPlugin))

		for _, version := range sets.List(versions) {
			cniPluginSettings = append(cniPluginSettings, &kubermaticv1.CNIPluginSettings{
				Type:    kubermaticv1.CNIPluginType(cniPlugin),
				Version: version,
			})
		}
	}

	return cniPluginSettings
}

func getImagesFromManifest(log logrus.FieldLogger, decoder runtime.Decoder, b []byte) ([]string, error) {
	obj, err := runtime.Decode(decoder, b)
	if err != nil {
		if runtime.IsNotRegisteredError(err) {
			// We must skip custom objects. We try to look up the object info though to give a useful warning
			metaFactory := &json.SimpleMetaFactory{}
			if gvk, err := metaFactory.Interpret(b); err == nil {
				log = log.WithField("gvk", gvk.String())
			}

			log.Debug("Skipping object because it is not of a known GVK")
			return nil, nil
		}

		return nil, fmt.Errorf("unable to decode object: %w", err)
	}

	images := getImagesFromObject(obj)
	if images == nil {
		return nil, nil
	}

	return images, nil
}

func getImagesFromObject(obj runtime.Object) []string {
	// We don't have the conversion funcs available thus we must check all available Kubernetes types which can contain images
	switch obj := obj.(type) {
	case *appsv1.Deployment:
		return getImagesFromPodSpec(obj.Spec.Template.Spec)
	case *appsv1.ReplicaSet:
		return getImagesFromPodSpec(obj.Spec.Template.Spec)
	case *appsv1.StatefulSet:
		return getImagesFromPodSpec(obj.Spec.Template.Spec)
	case *appsv1.DaemonSet:
		return getImagesFromPodSpec(obj.Spec.Template.Spec)
	case *corev1.Pod:
		return getImagesFromPodSpec(obj.Spec)

	// CronJob
	case *batchv1.CronJob:
		return getImagesFromPodSpec(obj.Spec.JobTemplate.Spec.Template.Spec)
	case *batchv1beta1.CronJob:
		return getImagesFromPodSpec(obj.Spec.JobTemplate.Spec.Template.Spec)

	// Job
	case *batchv1.Job:
		return getImagesFromPodSpec(obj.Spec.Template.Spec)
	}

	return nil
}

func getVersionsFromKubermaticConfiguration(config *kubermaticv1.KubermaticConfiguration) []*version.Version {
	versions := []*version.Version{}

	for _, v := range config.Spec.Versions.Versions {
		versions = append(versions, &version.Version{
			Version: v.Semver(),
		})
	}

	return versions
}
