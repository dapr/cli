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
	publishAppID   string
	pubsubName     string
	publishTopic   string
	publishPayload string
)

var PublishCmd = &cobra.Command{
	Use:   "publish",
	Short: "Publish a pub-sub event. Supported platforms: Self-hosted",
	Example: `
# Publish to sample topic in target pubsub via a publishing app
dapr publish --publish-app-id myapp --pubsub target --topic sample --data '{"key":"value"}'
`,
	Run: func(cmd *cobra.Command, args []string) {
		client := standalone.NewClient()
		err := client.Publish(publishAppID, pubsubName, publishTopic, publishPayload)
		if err != nil {
			print.FailureStatusEvent(os.Stdout, fmt.Sprintf("Error publishing topic %s: %s", publishTopic, err))
			os.Exit(1)
		}

		print.SuccessStatusEvent(os.Stdout, "Event published successfully")
	},
}

func init() {
	PublishCmd.Flags().StringVarP(&publishAppID, "publish-app-id", "i", "", "The ID of the publishing app")
	PublishCmd.Flags().StringVarP(&pubsubName, "pubsub", "p", "", "The name of the pub/sub component")
	PublishCmd.Flags().StringVarP(&publishTopic, "topic", "t", "", "The topic to be published to")
	PublishCmd.Flags().StringVarP(&publishPayload, "data", "d", "", "The JSON serialized data string (optional)")
	PublishCmd.Flags().BoolP("help", "h", false, "Print this help message")
	PublishCmd.MarkFlagRequired("publish-app-id")
	PublishCmd.MarkFlagRequired("topic")
	PublishCmd.MarkFlagRequired("pubsub")
	RootCmd.AddCommand(PublishCmd)
}
