/*
Copyright 2021 The Dapr Authors
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

package utils

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/dapr/cli/pkg/print"

	"github.com/docker/docker/client"
	"github.com/gocarina/gocsv"
	"github.com/olekukonko/tablewriter"
	"gopkg.in/yaml.v2"
)

type ContainerRuntime string

const (
	DOCKER ContainerRuntime = "docker"
	PODMAN ContainerRuntime = "podman"

	marinerImageVariantName = "mariner"

	socketFormat = "%s/dapr-%s-%s.socket"

	windowsOsType = "windows"
	homeDirPrefix = "~/"
)

// IsValidContainerRuntime checks if the input is a valid container runtime.
// Valid container runtimes are docker and podman.
func IsValidContainerRuntime(containerRuntime string) bool {
	containerRuntime = strings.TrimSpace(containerRuntime)
	return containerRuntime == string(DOCKER) || containerRuntime == string(PODMAN)
}

// GetContainerRuntimeCmd returns a valid container runtime to be used by CLI operations.
// If the input is a valid container runtime, it is returned as is.
// Otherwise the default container runtime, docker, is returned.
func GetContainerRuntimeCmd(containerRuntime string) string {
	if IsValidContainerRuntime(containerRuntime) {
		return strings.TrimSpace(containerRuntime)
	}
	// Default to docker.
	return string(DOCKER)
}

// Contains returns true if vs contains x.
func Contains[T comparable](vs []T, x T) bool {
	for _, v := range vs {
		if v == x {
			return true
		}
	}
	return false
}

// PrintTable to print in the table format.
func PrintTable(csvContent string) {
	WriteTable(os.Stdout, csvContent)
}

// WriteTable writes the csv table to writer.
func WriteTable(writer io.Writer, csvContent string) {
	table := tablewriter.NewWriter(writer)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetBorder(false)
	table.SetHeaderLine(false)
	table.SetRowLine(false)
	table.SetCenterSeparator("")
	table.SetRowSeparator("")
	table.SetColumnSeparator("")
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	scanner := bufio.NewScanner(strings.NewReader(csvContent))
	header := true

	for scanner.Scan() {
		text := strings.Split(scanner.Text(), ",")

		if header {
			table.SetHeader(text)
			header = false
		} else {
			table.Append(text)
		}
	}

	table.Render()
}

func TruncateString(str string, maxLength int) string {
	if len(str) <= maxLength {
		return str
	}

	return str[0:maxLength-3] + "..."
}

func RunCmdAndWait(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", err
	}

	err = cmd.Start()
	if err != nil {
		return "", err
	}

	resp, err := io.ReadAll(stdout)
	if err != nil {
		return "", err
	}
	errB, err := io.ReadAll(stderr)
	if err != nil {
		//nolint
		return "", nil
	}

	err = cmd.Wait()
	if err != nil {
		// in case of error, capture the exact message.
		if len(errB) > 0 {
			return "", errors.New(string(errB))
		}
		return "", err
	}

	return string(resp), nil
}

func CreateContainerName(serviceContainerName string, dockerNetwork string) string {
	if dockerNetwork != "" {
		return fmt.Sprintf("%s_%s", serviceContainerName, dockerNetwork)
	}

	return serviceContainerName
}

func CreateDirectory(dir string) error {
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		return nil
	}
	return os.Mkdir(dir, 0o777)
}

// IsContainerRuntimeInstalled checks whether the given container runtime is installed.
// If the container runtime is unsupported, false is returned.
func IsContainerRuntimeInstalled(containerRuntime string) bool {
	if containerRuntime == string(PODMAN) {
		return isPodmanInstalled()
	} else if containerRuntime == string(DOCKER) {
		return isDockerInstalled()
	}
	// This should never happen.
	return false
}

// isDockerInstalled checks whether docker is installed.
func isDockerInstalled() bool {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return false
	}
	_, err = cli.Ping(context.Background())
	return err == nil
}

// isPodmanInstalled checks whether podman is installed.
func isPodmanInstalled() bool {
	cmd := exec.Command("podman", "version")
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}

// IsDaprListeningOnPort checks if Dapr is litening to a given port.
func IsDaprListeningOnPort(port int, timeout time.Duration) error {
	start := time.Now()
	for {
		host := fmt.Sprintf("127.0.0.1:%v", port)
		conn, err := net.DialTimeout("tcp", host, timeout)
		if err == nil {
			conn.Close()
			return nil
		}

		if time.Since(start).Seconds() >= timeout.Seconds() {
			// Give up.
			return err
		}

		time.Sleep(time.Second)
	}
}

func IsDaprListeningOnSocket(socket string, timeout time.Duration) error {
	start := time.Now()
	for {
		conn, err := net.DialTimeout("unix", socket, timeout)
		if err == nil {
			conn.Close()
			return nil
		}

		if time.Since(start).Seconds() >= timeout.Seconds() {
			// Give up.
			return err
		}

		time.Sleep(time.Second)
	}
}

func MarshalAndWriteTable(writer io.Writer, in interface{}) error {
	table, err := gocsv.MarshalString(in)
	if err != nil {
		return err
	}

	WriteTable(writer, table)
	return nil
}

func PrintDetail(writer io.Writer, outputFormat string, list interface{}) error {
	var err error
	output := []byte{}

	switch outputFormat {
	case "yaml":
		output, err = yaml.Marshal(list)
	case "json":
		output, err = json.MarshalIndent(list, "", "  ")
	default:
		err = fmt.Errorf("unsupported output format: %s", outputFormat)
	}
	if err != nil {
		return err
	}

	_, err = writer.Write(output)
	return err
}

func IsAddressLegal(address string) bool {
	var isLegal bool
	if address == "localhost" {
		isLegal = true
	} else if net.ParseIP(address) != nil {
		isLegal = true
	}
	return isLegal
}

// CheckIfPortAvailable returns an error if the port is not available else returns nil.
func CheckIfPortAvailable(port int) error {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return err
	}
	ln.Close()
	return nil
}

// GetEnv get value from environment variable.
func GetEnv(envName string, defaultValue string) string {
	if val, ok := os.LookupEnv(envName); ok {
		return val
	}

	return defaultValue
}

func GetSocket(path, appID, protocol string) string {
	return fmt.Sprintf(socketFormat, path, appID, protocol)
}

func GetDefaultRegistry(githubContainerRegistryName, dockerContainerRegistryName string) (string, error) {
	val := strings.ToLower(os.Getenv("DAPR_DEFAULT_IMAGE_REGISTRY"))
	switch val {
	case "":
		print.InfoStatusEvent(os.Stdout, "Container images will be pulled from Docker Hub")
		return dockerContainerRegistryName, nil
	case githubContainerRegistryName:
		print.InfoStatusEvent(os.Stdout, "Container images will be pulled from Dapr GitHub container registry")
		return githubContainerRegistryName, nil
	default:
		return "", fmt.Errorf("environment variable %q can only be set to %s", "DAPR_DEFAULT_IMAGE_REGISTRY", "GHCR")
	}
}

func ValidateImageVariant(imageVariant string) error {
	if imageVariant != "" && imageVariant != marinerImageVariantName {
		return fmt.Errorf("image variant %s is not supported", imageVariant)
	}
	return nil
}

func GetVariantVersion(version, imageVariant string) string {
	if imageVariant == "" {
		return version
	}
	return fmt.Sprintf("%s-%s", version, imageVariant)
}

// Returns image version and variant.
// Expected imageTag format: <version>-<variant>, i.e. 1.0.0-mariner or 1.0.0-rc.1-mariner.
func GetVersionAndImageVariant(imageTag string) (string, string) {
	imageVersionOffset := strings.LastIndex(imageTag, "-")
	imageVariant := imageTag[imageVersionOffset+1:]
	if imageVariant == marinerImageVariantName {
		return imageTag[:imageVersionOffset], imageVariant
	}
	return imageTag, ""
}

// Returns true if the given file path is valid.
func ValidateFilePath(filePath string) error {
	if filePath != "" {
		if _, err := os.Stat(filePath); err != nil {
			return fmt.Errorf("error in getting the file info for %s: %w", filePath, err)
		}
	}
	return nil
}

// GetAbsPath returns the absolute path of the given file path and base directory.
func GetAbsPath(baseDir, path string) string {
	if path == "" {
		return ""
	}
	if filepath.IsAbs(path) {
		return path
	}
	absPath := filepath.Join(baseDir, filepath.Clean(path))
	return absPath
}

// ResolveHomeDir resolves prefix of the given path, if present, to the user's home directory and returns it.
func ResolveHomeDir(filePath string) (string, error) {
	if filePath == "" {
		return "", nil
	}

	// Resolve the home directory prefix, if present. This is only supported on non-Windows platforms.
	if runtime.GOOS != windowsOsType && strings.HasPrefix(filePath, homeDirPrefix) {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("error in getting the home directory for %s: %w", filePath, err)
		}
		filePath = filepath.Join(homeDir, filePath[len(homeDirPrefix):])
	}

	return filePath, nil
}

func ReadFile(filePath string) ([]byte, error) {
	bytes, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error in reading the provided app config file: %w", err)
	}
	return bytes, nil
}

// FindFileInDir finds and returns the path of the given file name in the given directory.
func FindFileInDir(dirPath, fileName string) (string, error) {
	filePath := filepath.Join(dirPath, fileName)
	if err := ValidateFilePath(filePath); err != nil {
		return "", fmt.Errorf("error in validating the file path %q: %w", filePath, err)
	}
	return filePath, nil
}
