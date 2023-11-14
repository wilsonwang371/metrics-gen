package utils

import (
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
)

func NewFilenameForTracing(oldName string, suffix string) string {
	if strings.HasSuffix(oldName, ".go") {
		base := strings.TrimSuffix(oldName, ".go")
		return base + fmt.Sprintf("_%s.go", suffix)
	}
	log.Errorf("filename %s does not have .go suffix", oldName)
	return oldName
}

func DeduplicateStrings(input []string) []string {
	// Create a map to store unique strings
	uniqueStrings := make(map[string]struct{})

	// Create a slice to store deduplicated strings
	deduplicated := []string{}

	// Iterate over the input slice
	for _, str := range input {
		// Check if the string is not in the map (not seen before)
		if _, ok := uniqueStrings[str]; !ok {
			// Add the string to the map and the deduplicated slice
			uniqueStrings[str] = struct{}{}
			deduplicated = append(deduplicated, str)
		}
	}

	return deduplicated
}
