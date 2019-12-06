package seedsync

import (
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"
)

func seedCreator(seed *kubermaticv1.Seed) reconciling.NamedSeedCreatorGetter {
	return func() (string, reconciling.SeedCreator) {
		return seed.Name, func(s *kubermaticv1.Seed) (*kubermaticv1.Seed, error) {
			s.Labels = seed.Labels
			if s.Labels == nil {
				s.Labels = make(map[string]string)
			}
			s.Labels[ManagedByLabel] = ControllerName

			s.Annotations = seed.Annotations
			s.Spec = seed.Spec

			return s, nil
		}
	}
}
