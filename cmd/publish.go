// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
// ------------------------------------------------------------

package cmd

import (
	"fmt"
	"os"

	"github.com/dapr/cli/pkg/print"
	"github.com/dapr/cli/pkg/standalone"
	"github.com/spf13/cobra"
)

var publishTopic string
var publishPayload string
var pubsubName string

var PublishCmd = &cobra.Command{
	Use:   "publish",
	Short: "Publish an event to multiple consumers",
	Run: func(cmd *cobra.Command, args []string) {
		client := standalone.NewStandaloneClient()
		err := client.Publish(publishTopic, publishPayload, pubsubName)
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
	PublishCmd.Flags().StringVarP(&pubsubName, "pubsub", "", "", "name of the pub/sub component")
	PublishCmd.MarkFlagRequired("app-id")
	PublishCmd.MarkFlagRequired("topic")
	PublishCmd.MarkFlagRequired("pubsub")
	RootCmd.AddCommand(PublishCmd)
}
