package utils

import (
	"bufio"
	"os"
	"os/exec"
	"strings"

	"github.com/olekukonko/tablewriter"

	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

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
