// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation and Dapr Contributors.
// Licensed under the MIT License.
// ------------------------------------------------------------

package utils

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"time"

	"github.com/docker/docker/client"
	"github.com/gocarina/gocsv"
	"github.com/olekukonko/tablewriter"
	"gopkg.in/yaml.v2"
)

const (
	socketFormat = "%s/dapr-%s-%s.socket"
)

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
	strLength := len(str)
	if strLength <= maxLength {
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
		// in case of error, capture the exact message
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
	return os.Mkdir(dir, 0777)
}

// IsDockerInstalled checks whether docker is installed/running.
func IsDockerInstalled() bool {
	// nolint:staticcheck
	cli, err := client.NewEnvClient()
	if err != nil {
		return false
	}
	_, err = cli.Ping(context.Background())
	return err == nil
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
	obj := list
	s := reflect.ValueOf(list)
	if s.Kind() == reflect.Slice && s.Len() == 1 {
		obj = s.Index(0).Interface()
	}

	var err error
	output := []byte{}

	switch outputFormat {
	case "yaml":
		output, err = yaml.Marshal(obj)
	case "json":
		output, err = json.MarshalIndent(obj, "", "  ")
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

func GetSocket(path, appID, protocol string) string {
	return fmt.Sprintf(socketFormat, path, appID, protocol)
}
