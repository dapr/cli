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
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/dapr/cli/pkg/print"

	"github.com/docker/docker/client"
	"github.com/gocarina/gocsv"
	"github.com/olekukonko/tablewriter"
	"gopkg.in/yaml.v2"
)

const (
	socketFormat = "%s/dapr-%s-%s.socket"
	ProcTCP      = "/proc/net/tcp"
	ProcUDP      = "/proc/net/udp"
	ProcTCP6     = "/proc/net/tcp6"
	ProcUDP6     = "/proc/net/udp6"
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
	if len(str) <= maxLength || maxLength <= 3 {
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

type SocketInfo struct {
	IP       string
	Port     int64
	Protocol string
}

// Remove empty string from array.
func removeEmptyStr(array []string) []string {
	var result []string
	for _, s := range array {
		if s != "" {
			result = append(result, s)
		}
	}
	return result
}

// Convert hexadecimal to decimal.
func hexToDec(h string) int64 {
	d, _ := strconv.ParseInt(h, 16, 32)
	return d
}

// Convert the ipv4 to decimal.
func convertIP(ip string) string {
	var out string

	// Check ip size if greater than 8 is a ipv6 type.
	if len(ip) > 8 {
		i := []string{
			ip[30:32],
			ip[28:30],
			ip[26:28],
			ip[24:26],
			ip[22:24],
			ip[20:22],
			ip[18:20],
			ip[16:18],
			ip[14:16],
			ip[12:14],
			ip[10:12],
			ip[8:10],
			ip[6:8],
			ip[4:6],
			ip[2:4],
			ip[0:2],
		}
		out = fmt.Sprintf("%v%v:%v%v:%v%v:%v%v:%v%v:%v%v:%v%v:%v%v",
			i[14], i[15], i[13], i[12],
			i[10], i[11], i[8], i[9],
			i[6], i[7], i[4], i[5],
			i[2], i[3], i[0], i[1])
	} else {
		i := []int64{
			hexToDec(ip[6:8]),
			hexToDec(ip[4:6]),
			hexToDec(ip[2:4]),
			hexToDec(ip[0:2]),
		}
		out = fmt.Sprintf("%v.%v.%v.%v", i[0], i[1], i[2], i[3])
	}
	return out
}

// Format local ip and port.
func formatLocalSocket(line string) (string, int64) {
	lineArray := removeEmptyStr(strings.Split(strings.TrimSpace(line), " "))
	ipPort := strings.Split(lineArray[1], ":")
	ip := convertIP(ipPort[0])
	port := hexToDec(ipPort[1])
	return ip, port
}

// Get sockets from tcp or udp file.
func readSockets(protocol string) ([]string, error) {
	var procPath string
	switch protocol {
	case "tcp":
		procPath = ProcTCP
	case "udp":
		procPath = ProcUDP
	case "tcp6":
		procPath = ProcTCP6
	case "udp6":
		procPath = ProcUDP6
	default:
		err := errors.New(protocol + " is a invalid protocol, tcp and udp only")
		return nil, err
	}

	data, err := ioutil.ReadFile(procPath)
	if err != nil {
		err = errors.New("read proc file error:" + err.Error())
		return nil, err
	}
	lines := strings.Split(string(data), "\n")

	// Return lines without Header line and blank line on the end.
	return lines[1 : len(lines)-1], nil
}

// Get define protocol local socket info.
func GetDefineProtocolSockets(protocol string) ([]SocketInfo, error) {
	sockets := []SocketInfo{}
	tcpData, err := readSockets(protocol)
	if err != nil {
		return nil, err
	}

	for _, line := range tcpData {
		ip, port := formatLocalSocket(line)
		sockets = append(sockets, SocketInfo{ip, port, protocol})
	}
	return sockets, nil
}
