package resources

import (
	"fmt"

	"github.com/docker/distribution/reference"
)

// OpenshiftImageWithRegistry will return docker image name for Openshift images. The function is
// digest-aware and can be used with the overwriteRegistry option and with image-loader.
func OpenshiftImageWithRegistry(image, registry string) (string, error) {
	if registry == "" {
		return image, nil
	}
	imageRef, err := reference.ParseNamed(image)
	if err != nil {
		return "", fmt.Errorf("failed to parse image: %v", err)
	}
	if reference.Domain(imageRef) == registry {
		return image, nil
	}
	if taggedImageRef, ok := imageRef.(reference.NamedTagged); ok {
		return fmt.Sprintf("%s/%s:%s", registry, reference.Path(imageRef), taggedImageRef.Tag()), nil
	} else if digestedImageRef, ok := imageRef.(reference.Digested); ok {
		return fmt.Sprintf("%s/%s:%s", registry, reference.Path(imageRef), digestedImageRef.Digest().Hex()), nil
	}
	return "", nil
}
