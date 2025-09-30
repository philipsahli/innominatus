package main

import (
	"regexp"
)

func BuildGraph(scoreSpec *ScoreSpec) map[string][]string {
	graph := make(map[string][]string)
	
	// Parse container variables to find resource references
	for containerName, container := range scoreSpec.Containers {
		containerKey := "container:" + containerName
		var dependencies []string
		
		// Look for ${resources.resourceName.*} patterns in variables
		resourcePattern := regexp.MustCompile(`\$\{resources\.([^.}]+)`)
		
		for _, value := range container.Variables {
			matches := resourcePattern.FindAllStringSubmatch(value, -1)
			for _, match := range matches {
				if len(match) > 1 {
					resourceName := match[1]
					// Check if this resource actually exists in the spec
					if _, exists := scoreSpec.Resources[resourceName]; exists {
						// Avoid duplicates
						if !contains(dependencies, resourceName) {
							dependencies = append(dependencies, resourceName)
						}
					}
				}
			}
		}
		
		if len(dependencies) > 0 {
			graph[containerKey] = dependencies
		}
	}
	
	// Add environment node and connect it to all resources
	if scoreSpec.Environment != nil {
		var resourceList []string
		for resourceName := range scoreSpec.Resources {
			resourceList = append(resourceList, resourceName)
		}
		if len(resourceList) > 0 {
			graph["environment"] = resourceList
		}
	}
	
	return graph
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}