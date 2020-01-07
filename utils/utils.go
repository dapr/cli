// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
// ------------------------------------------------------------

package utils

import (
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"github.com/olekukonko/tablewriter"

	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

// PrintTable to print in the table format
func PrintTable(csvContent string) {
	table := tablewriter.NewWriter(os.Stdout)
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
	strLength := len(str)
	if strLength <= maxLength {
		return str
	}

	return str[0:maxLength-3] + "..."
}
func RunCmd(name string, args ...string) (string, error) {
	cmd := exec.Command("docker", args...)

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

	resp, err := ioutil.ReadAll(stdout)
	if err != nil {
		return "", err
	}
	errB, err := ioutil.ReadAll(stderr)
	if err != nil {
		return "", nil
	}

	err = cmd.Wait()
	if err != nil {
		if len(errB) > 0 {
			return "", errors.New(string(errB))
		}
		return "", err
	}
	return strings.TrimSuffix(string(resp), "\n"), nil
}
func RunCmdAndWait(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	err := cmd.Start()
	if err != nil {
		return err
	}

	err = cmd.Wait()
	if err != nil {
		return err
	}

	return nil
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
	return os.Mkdir(dir, 0777)
}
