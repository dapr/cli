package cmd

import (
	"fmt"
	"os"

	"github.com/actionscore/cli/pkg/print"
	"github.com/actionscore/cli/pkg/publish"
	"github.com/spf13/cobra"
)

var publishAppID string
var publishTopic string
var publishPayload string

var PublishCmd = &cobra.Command{
	Use:   "publish",
	Short: "publish an event to multiple consumers",
	Run: func(cmd *cobra.Command, args []string) {
		err := publish.PublishTopic(publishAppID, publishTopic, publishPayload)
		if err != nil {
			print.FailureStatusEvent(os.Stdout, fmt.Sprintf("Error publishing topic %s: %s", publishTopic, err))
			return
		}

		print.SuccessStatusEvent(os.Stdout, "Event published successfully")
	},
}

func init() {
	PublishCmd.Flags().StringVarP(&publishTopic, "topic", "t", "", "the topic the app is listening on")
	PublishCmd.Flags().StringVarP(&publishAppID, "app-id", "a", "", "The app to publish the message to")
	PublishCmd.Flags().StringVarP(&publishPayload, "payload", "p", "", "(optional) a json payload")
	PublishCmd.MarkFlagRequired("app-id")
	PublishCmd.MarkFlagRequired("topic")
	RootCmd.AddCommand(PublishCmd)
}
