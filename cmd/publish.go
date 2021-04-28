// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation and Dapr Contributors.
// Licensed under the MIT License.
// ------------------------------------------------------------

package cmd

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/dapr/cli/pkg/print"
	"github.com/dapr/cli/pkg/standalone"
	"github.com/spf13/cobra"
)

var (
	publishAppID       string
	pubsubName         string
	publishTopic       string
	publishPayload     string
	publishPayloadFile string
)

var PublishCmd = &cobra.Command{
	Use:   "publish",
	Short: "Publish a pub-sub event. Supported platforms: Self-hosted",
	Example: `
# Publish to sample topic in target pubsub via a publishing app
dapr publish --publish-app-id myapp --pubsub target --topic sample --data '{"key":"value"}'
`,
	Run: func(cmd *cobra.Command, args []string) {
		bytePayload := []byte{}
		var err error
		if publishPayloadFile != "" && publishPayload != "" {
			print.FailureStatusEvent(os.Stdout, "Only one of --data and --data-file allowed in the same publish command")
			os.Exit(1)
		}

		if publishPayloadFile != "" {
			bytePayload, err = ioutil.ReadFile(publishPayloadFile)
			if err != nil {
				print.FailureStatusEvent(os.Stdout, "Error reading payload from '%s'. Error: %s", publishPayloadFile, err)
				os.Exit(1)
			}
		} else if publishPayload != "" {
			bytePayload = []byte(publishPayload)
		}

		client := standalone.NewClient()
		err = client.Publish(publishAppID, pubsubName, publishTopic, bytePayload)
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
	PublishCmd.Flags().StringVarP(&publishPayloadFile, "data-file", "f", "", "A file containing the JSON serialized data (optional)")
	PublishCmd.Flags().BoolP("help", "h", false, "Print this help message")
	PublishCmd.MarkFlagRequired("publish-app-id")
	PublishCmd.MarkFlagRequired("topic")
	PublishCmd.MarkFlagRequired("pubsub")
	RootCmd.AddCommand(PublishCmd)
}
