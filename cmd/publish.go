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

var (
	publishTopic   string
	publishPayload string
	pubsubName     string
)

var PublishCmd = &cobra.Command{
	Use:   "publish",
	Short: "Publish a pub-sub event",
	Run: func(cmd *cobra.Command, args []string) {
		client := standalone.NewClient()
		err := client.Publish(publishTopic, publishPayload, pubsubName)
		if err != nil {
			print.FailureStatusEvent(os.Stdout, fmt.Sprintf("Error publishing topic %s: %s", publishTopic, err))
			os.Exit(1)
		}

		print.SuccessStatusEvent(os.Stdout, "Event published successfully")
	},
}

func init() {
	PublishCmd.Flags().StringVarP(&publishTopic, "topic", "t", "", "The topic to be published to")
	PublishCmd.Flags().StringVarP(&publishPayload, "data", "d", "", "The JSON serialized string (optional)")
	PublishCmd.Flags().StringVarP(&pubsubName, "pubsub", "", "", "The name of the pub/sub component")
	PublishCmd.Flags().BoolP("help", "h", false, "Print this help message")
	PublishCmd.MarkFlagRequired("app-id")
	PublishCmd.MarkFlagRequired("topic")
	PublishCmd.MarkFlagRequired("pubsub")
	RootCmd.AddCommand(PublishCmd)
}
