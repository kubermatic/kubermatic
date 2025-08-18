package scenarios

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/config"
)

// FlavorMutator allows provider-specific adjustments to a generated flavor combo,
// e.g. injecting image locations or reshaping provider-specific fields.
type FlavorMutator func(combo map[string]interface{}, image string) (map[string]interface{}, error)

// providerFlavorMutators registers per-provider mutators used by the generator.
var providerFlavorMutators = map[string]FlavorMutator{
	"kubevirt": KubevirtFlavorMutator,
}

// deepCopyMap creates a deep copy of a map.
func deepCopyMap(original map[string]interface{}) map[string]interface{} {
	newMap := make(map[string]interface{})
	for key, value := range original {
		switch v := value.(type) {
		case map[string]interface{}:
			newMap[key] = deepCopyMap(v)
		case []interface{}:
			newSlice := make([]interface{}, len(v))
			for i, item := range v {
				if iv, ok := item.(map[string]interface{}); ok {
					newSlice[i] = deepCopyMap(iv)
				} else {
					newSlice[i] = item
				}
			}
			newMap[key] = newSlice
		default:
			newMap[key] = v
		}
	}
	return newMap
}

// convertKeysToStrings recursively converts map keys from interface{} to string.
// This is crucial because yaml.Unmarshal can create map[interface{}]interface{}.
func convertKeysToStrings(i interface{}) (interface{}, error) {
	switch x := i.(type) {
	case map[string]interface{}:
		m2 := make(map[string]interface{})
		for k, v := range x {
			v2, err := convertKeysToStrings(v)
			if err != nil {
				return nil, err
			}
			m2[k] = v2
		}
		return m2, nil
	case map[interface{}]interface{}:
		m2 := map[string]interface{}{}
		for k, v := range x {
			kStr, ok := k.(string)
			if !ok {
				return nil, fmt.Errorf("found non-string key in map: %v", k)
			}
			v2, err := convertKeysToStrings(v)
			if err != nil {
				return nil, err
			}
			m2[kStr] = v2
		}
		return m2, nil
	case []interface{}:
		s2 := make([]interface{}, len(x))
		for i, v := range x {
			v2, err := convertKeysToStrings(v)
			if err != nil {
				return nil, err
			}
			s2[i] = v2
		}
		return s2, nil
	default:
		return i, nil
	}
}

// generateCombinations creates the Cartesian product of a configuration map.
func generateCombinations(node map[string]interface{}) []map[string]interface{} {
	var combinations = []map[string]interface{}{{}}

	var keys []string
	for k := range node {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := node[k]
		var newCombinations []map[string]interface{}

		var valueOptions []interface{}
		if subMap, ok := v.(map[string]interface{}); ok {
			subCombinations := generateCombinations(subMap)
			for _, sc := range subCombinations {
				valueOptions = append(valueOptions, sc)
			}
		} else if slice, ok := v.([]interface{}); ok {
			valueOptions = slice
		} else {
			valueOptions = []interface{}{v}
		}

		if len(valueOptions) == 0 {
			valueOptions = []interface{}{nil}
		}

		for _, combo := range combinations {
			for _, option := range valueOptions {
				newCombo := deepCopyMap(combo)
				if option != nil {
					newCombo[k] = option
				}
				newCombinations = append(newCombinations, newCombo)
			}
		}
		combinations = newCombinations
	}

	return combinations
}

func flattenForName(prefix string, node map[string]interface{}, flatMap map[string]interface{}) {
	keys := make([]string, 0, len(node))
	for k := range node {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := node[k]
		newPrefix := k
		if prefix != "" {
			newPrefix = prefix + "." + k
		}

		switch value := v.(type) {
		case map[string]interface{}:
			flattenForName(newPrefix, value, flatMap)
		default:
			flatMap[newPrefix] = v
		}
	}
}

func mapToFlavor(combo map[string]interface{}) config.Flavor {
	flatCombo := make(map[string]interface{})
	flattenForName("", combo, flatCombo)

	var nameParts []string
	var keys []string
	for k := range flatCombo {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		nameParts = append(nameParts, fmt.Sprintf("%s-%v", strings.ReplaceAll(key, ".", "-"), flatCombo[key]))
	}

	name := strings.Join(nameParts, "_")
	if name == "" {
		name = "default"
	}

	return config.Flavor{
		Name:  name,
		Value: combo,
	}
}

func GenerateScenarios(templateFile, outputFile string) error {
	if templateFile == "" {
		return fmt.Errorf("--from flag must be set")
	}

	data, err := os.ReadFile(templateFile)
	if err != nil {
		return fmt.Errorf("failed to read template file: %w", err)
	}

	var rawTemplate interface{}
	if err := yaml.Unmarshal(data, &rawTemplate); err != nil {
		return fmt.Errorf("failed to unmarshal template file: %w", err)
	}

	saneTemplate, err := convertKeysToStrings(rawTemplate)
	if err != nil {
		return fmt.Errorf("failed to sanitize template: %w", err)
	}

	template, ok := saneTemplate.(map[string]interface{})
	if !ok {
		return fmt.Errorf("template root is not a map")
	}

	providersNode, ok := template["providers"]
	if !ok {
		return fmt.Errorf("template has no 'providers' list")
	}
	providersSlice, ok := providersNode.([]interface{})
	if !ok {
		return fmt.Errorf("'providers' is not a list")
	}

	distributionsNode, ok := template["distributions"]
	if !ok {
		return fmt.Errorf("template has no 'distributions' map")
	}
	distributionsMap, ok := distributionsNode.(map[string]interface{})
	if !ok {
		return fmt.Errorf("'distributions' is not a map")
	}

	versionsNode, hasVersions := template["versions"]
	var versions []string
	if hasVersions {
		if vSlice, ok := versionsNode.([]interface{}); ok {
			for _, v := range vSlice {
				if s, ok := v.(string); ok {
					versions = append(versions, s)
				}
			}
		}
	}

	var scenarios []config.Scenario
	for _, providerNode := range providersSlice {
		provider, ok := providerNode.(string)
		if !ok {
			return fmt.Errorf("provider name in 'providers' list is not a string")
		}

		var providerConfig map[string]interface{}
		if providerCfgNode, ok := template[provider]; ok {
			providerCfgMap, ok := providerCfgNode.(map[string]interface{})
			if !ok {
				return fmt.Errorf("provider config for %q is not a map", provider)
			}
			providerConfig = providerCfgMap
		}

		baseCombinations := []map[string]interface{}{{}}
		if providerConfig != nil {
			baseCombinations = generateCombinations(providerConfig)
		}

		for distName, imagesNode := range distributionsMap {
			images, ok := imagesNode.([]interface{})
			if !ok {
				return fmt.Errorf("images for distribution %q is not a list", distName)
			}

			if len(images) == 0 {
				images = []interface{}{""} // Create one flavor even if no images are specified.
			}

			var finalFlavors []config.Flavor
			for _, baseCombo := range baseCombinations {
				for _, imageNode := range images {
					image, ok := imageNode.(string)
					if !ok {
						return fmt.Errorf("image for distribution %q is not a string", distName)
					}

					newCombo := deepCopyMap(baseCombo)

					// Apply provider-specific flavor mutation if any (e.g., inject image)
					if mut, ok := providerFlavorMutators[provider]; ok {
						var err error
						newCombo, err = mut(newCombo, image)
						if err != nil {
							return fmt.Errorf("failed to mutate flavor for provider %q: %w", provider, err)
						}
					}

					finalFlavors = append(finalFlavors, mapToFlavor(newCombo))
				}
			}

			scenarios = append(scenarios, config.Scenario{
				Provider:        provider,
				OperatingSystem: distName,
				Flavors:         finalFlavors,
			})
		}
	}

	outputConfig := config.Config{
		Versions:  versions,
		Scenarios: scenarios,
	}

	outputData, err := yaml.Marshal(outputConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal output config: %w", err)
	}

	if err := os.WriteFile(outputFile, outputData, 0644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	fmt.Printf("Successfully generated %d scenarios to %s\n", len(scenarios), outputFile)
	return nil
}
