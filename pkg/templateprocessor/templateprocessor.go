/*
Copyright 2025 The Dapr Authors
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

package templateprocessor

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Template pattern: {{ENV_VAR_NAME}} or {{ENV_VAR_NAME:default value}}
// Matches uppercase letters, digits, and underscores for variable names.
// Optionally matches a colon followed by a default value.
var templatePattern = regexp.MustCompile(`\{\{([A-Z_][A-Z0-9_]*)(?::([^}]*))?\}\}`)

// ProcessedResources contains information about the processed resources.
type ProcessedResources struct {
	TempDir        string   // Temporary directory containing processed files
	ProcessedPaths []string // Paths to processed resource directories
	HasTemplates   bool     // Whether any templates were found and processed
}

// ProcessResourcesWithEnvVars processes resource files by substituting environment variables
// in template placeholders like {{ENV_VAR_NAME}}. It creates a temporary directory with
// processed files and returns the paths to use for daprd.
func ProcessResourcesWithEnvVars(resourcesPaths []string) (*ProcessedResources, error) {
	if len(resourcesPaths) == 0 {
		return nil, fmt.Errorf("no resource paths provided")
	}

	// Create temporary directory for processed files.
	tempDir, err := os.MkdirTemp("", "dapr-resources-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary directory: %w", err)
	}

	result := &ProcessedResources{
		TempDir:        tempDir,
		ProcessedPaths: make([]string, 0, len(resourcesPaths)),
		HasTemplates:   false,
	}

	// Process each resource path.
	for i, resourcePath := range resourcesPaths {
		// Create a subdirectory in temp dir for each resource path.
		destDir := filepath.Join(tempDir, fmt.Sprintf("resources-%d", i))
		err := os.MkdirAll(destDir, 0755)
		if err != nil {
			Cleanup(result.TempDir)
			return nil, fmt.Errorf("failed to create destination directory: %w", err)
		}

		// Check if resource path exists and is a directory.
		info, err := os.Stat(resourcePath)
		if err != nil {
			Cleanup(result.TempDir)
			return nil, fmt.Errorf("resource path %q does not exist: %w", resourcePath, err)
		}

		if !info.IsDir() {
			Cleanup(result.TempDir)
			return nil, fmt.Errorf("resource path %q is not a directory", resourcePath)
		}

		// Process all files in the resource directory.
		hasTemplates, err := processDirectory(resourcePath, destDir)
		if err != nil {
			Cleanup(result.TempDir)
			return nil, fmt.Errorf("failed to process resource path %q: %w", resourcePath, err)
		}

		if hasTemplates {
			result.HasTemplates = true
		}

		result.ProcessedPaths = append(result.ProcessedPaths, destDir)
	}

	return result, nil
}

// processDirectory recursively processes all files in a directory.
func processDirectory(srcDir, destDir string) (bool, error) {
	hasTemplates := false

	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return false, fmt.Errorf("failed to read directory %q: %w", srcDir, err)
	}

	for _, entry := range entries {
		srcPath := filepath.Join(srcDir, entry.Name())
		destPath := filepath.Join(destDir, entry.Name())

		if entry.IsDir() {
			// Create subdirectory and process recursively.
			err := os.MkdirAll(destPath, 0755)
			if err != nil {
				return false, fmt.Errorf("failed to create directory %q: %w", destPath, err)
			}

			subHasTemplates, err := processDirectory(srcPath, destPath)
			if err != nil {
				return false, err
			}
			if subHasTemplates {
				hasTemplates = true
			}
		} else {
			// Process file.
			fileHasTemplates, err := processFile(srcPath, destPath)
			if err != nil {
				return false, fmt.Errorf("failed to process file %q: %w", srcPath, err)
			}
			if fileHasTemplates {
				hasTemplates = true
			}
		}
	}

	return hasTemplates, nil
}

// processFile processes a single file, substituting environment variables if it's a
// text file (YAML, JSON), otherwise copying it as-is.
func processFile(srcPath, destPath string) (bool, error) {
	// Check if file should be processed for templates based on extension.
	shouldProcess := shouldProcessFile(srcPath)

	if !shouldProcess {
		// Copy file as-is.
		return false, copyFile(srcPath, destPath)
	}

	// Read source file.
	content, err := os.ReadFile(srcPath)
	if err != nil {
		return false, fmt.Errorf("failed to read file: %w", err)
	}

	// Substitute environment variables.
	processedContent, hasTemplates := substituteEnvVars(content)

	// Get original file permissions.
	info, err := os.Stat(srcPath)
	if err != nil {
		return false, fmt.Errorf("failed to get file info: %w", err)
	}

	// Write processed content to destination.
	err = os.WriteFile(destPath, processedContent, info.Mode())
	if err != nil {
		return false, fmt.Errorf("failed to write file: %w", err)
	}

	return hasTemplates, nil
}

// shouldProcessFile determines if a file should be processed for templates
// based on its extension.
func shouldProcessFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	processableExtensions := []string{".yaml", ".yml", ".json"}

	for _, processable := range processableExtensions {
		if ext == processable {
			return true
		}
	}
	return false
}

// substituteEnvVars replaces template placeholders {{ENV_VAR_NAME}} or
// {{ENV_VAR_NAME:default value}} with environment variable values.
// If the environment variable doesn't exist:
//   - If a default value is provided, use it
//   - Otherwise, leave the template as-is
//
// Returns the processed content and whether any substitutions were made.
func substituteEnvVars(content []byte) ([]byte, bool) {
	hasTemplates := false
	strContent := string(content)

	// Find all template matches.
	result := templatePattern.ReplaceAllStringFunc(strContent, func(match string) string {
		// Extract variable name and optional default value.
		// The regex captures: {{VAR_NAME}} or {{VAR_NAME:default}}
		matches := templatePattern.FindStringSubmatch(match)
		if len(matches) < 2 {
			return match
		}

		varName := matches[1]
		// Check if the match contains a colon to determine if default value syntax is used
		hasDefaultSyntax := strings.Contains(match, ":")
		defaultValue := ""
		if len(matches) > 2 {
			defaultValue = matches[2]
		}

		// Get environment variable value.
		if value, exists := os.LookupEnv(varName); exists {
			hasTemplates = true
			return value
		}

		// If env var doesn't exist but default value syntax is present (even if empty),
		// use the default value.
		if hasDefaultSyntax {
			hasTemplates = true
			return defaultValue
		}

		// If no env var and no default syntax, leave the template as-is.
		return match
	})

	return []byte(result), hasTemplates
}

// copyFile copies a file from src to dest, preserving permissions.
func copyFile(src, dest string) error {
	// Open source file.
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	// Get source file info for permissions.
	info, err := srcFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	// Create destination file.
	destFile, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, info.Mode())
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	// Copy content.
	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		return fmt.Errorf("failed to copy file content: %w", err)
	}

	return nil
}

// Cleanup removes the temporary directory and all its contents.
func Cleanup(tempDir string) error {
	if tempDir == "" {
		return nil
	}

	err := os.RemoveAll(tempDir)
	if err != nil {
		return fmt.Errorf("failed to cleanup temporary directory %q: %w", tempDir, err)
	}

	return nil
}
