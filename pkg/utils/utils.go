package utils

import (
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
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

// GoGetPackage runs `go get` on the given import path
func GoGetPackage(importPath string) error {
	log.Infof("go get %s", importPath)
	success := false
	cmds := [][]string{
		{"go", "get", "-d", importPath},
		{"go", "install", importPath},
	}
	for _, cmd := range cmds {
		c := exec.Command(cmd[0], cmd[1:]...)
		// c.Stdout = os.Stdout
		// c.Stderr = os.Stderr
		if err := c.Run(); err == nil {
			success = true
			break
		} else {
			log.Warnf("failed to run %s: %v", strings.Join(cmd, " "), err)
		}
	}
	if !success {
		return fmt.Errorf("failed to go get %s", importPath)
	}
	return nil
} // ignore_security_alert RCE

// GetPackages updates the packages in go.mod
func FetchPackages(goModPath string, pkgs []string) error {
	// get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	// make sure go.mod exists
	if _, err := os.Stat(goModPath); os.IsNotExist(err) {
		return fmt.Errorf("go.mod does not exist")
	}

	// get direcotry of go.mod
	absPath, err := filepath.Abs(goModPath)
	if err != nil {
		return err
	}
	projPath := filepath.Dir(absPath)

	// change the working directory to the project path
	if err := os.Chdir(projPath); err != nil {
		return err
	}

	for _, pkg := range pkgs {
		if err := GoGetPackage(pkg); err != nil {
			return err
		}
	}

	// change the working directory back to the original
	if err := os.Chdir(cwd); err != nil {
		return err
	}
	return nil
}

func ParseArguments(input string) map[string]string {
	args := make(map[string]string)

	// Split the input string by whitespace
	parts := strings.Fields(input)

	// Parse each part in the format "key=value"
	for _, part := range parts {
		keyValue := strings.SplitN(part, "=", 2)
		if len(keyValue) == 2 {
			key := keyValue[0]
			value := keyValue[1]
			args[key] = value
		}
	}

	return args
}

// Function to generate a random number string of a specified length
func GenerateRandNumString(length int) string {
	const charset = "0123456789" // You can add more characters if needed
	result := make([]byte, length)

	for i := 0; i < length; i++ {
		result[i] = charset[rand.Intn(len(charset))]
	}

	return string(result)
}
