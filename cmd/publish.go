// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
// ------------------------------------------------------------

package cmd

import (
	"fmt"
	"os"

	"github.com/dapr/cli/pkg/print"
	"github.com/dapr/cli/pkg/publish"
	"github.com/spf13/cobra"
)

var publishTopic string
var publishPayload string

var PublishCmd = &cobra.Command{
	Use:   "publish",
	Short: "Publish an event to multiple consumers",
	Run: func(cmd *cobra.Command, args []string) {
		err := publish.SendPayloadToTopic(publishTopic, publishPayload)
		if err != nil {
			print.FailureStatusEvent(os.Stdout, fmt.Sprintf("Error publishing topic %s: %s", publishTopic, err))
			return
		}

		print.SuccessStatusEvent(os.Stdout, "Event published successfully")
	},
}

func init() {
	PublishCmd.Flags().StringVarP(&publishTopic, "topic", "t", "", "the topic the app is listening on")
	PublishCmd.Flags().StringVarP(&publishPayload, "data", "d", "", "(optional) a json serialized string")
	PublishCmd.MarkFlagRequired("app-id")
	PublishCmd.MarkFlagRequired("topic")
	RootCmd.AddCommand(PublishCmd)
}
