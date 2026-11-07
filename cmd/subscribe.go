/*
Copyright 2026 The Dapr Authors
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

package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	"github.com/dapr/cli/pkg/print"
	"github.com/dapr/cli/pkg/standalone"
)

var (
	subscribeAppID    string
	subscribePubsub   string
	subscribeTopic    string
	subscribeMetadata string
	subscribeRoutes   string
	subscribeSocket   string
	subscribeOutput   string
	subscribeResponse string
)

var SubscribeCmd = &cobra.Command{
	Use:   "subscribe",
	Short: "Subscribe to an HTTP pub-sub topic and print received events. Supported platforms: Self-hosted",
	Example: `
# Terminal 1: start a sidecar with an unused HTTP app port
dapr run --app-id myapp --app-port 5001

# Terminal 2: let the CLI act as the HTTP subscriber on that app port
dapr subscribe --subscribe-app-id myapp --pubsub target --topic sample

# Subscribe to raw payloads
dapr subscribe --subscribe-app-id myapp --pubsub target --topic sample --metadata '{"rawPayload":"true"}'

# Subscribe using custom routes
dapr subscribe --subscribe-app-id myapp --pubsub target --topic sample --routes '{"rules":[{"match":"event.type == '\''widget'\''","path":"/widgets"}],"default":"/products"}'

# Print the timestamp and metadata with each event
dapr subscribe --subscribe-app-id myapp --pubsub target --topic sample --output all

# Ask Dapr to retry each delivered event
dapr subscribe --subscribe-app-id myapp --pubsub target --topic sample --response retry
`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		metadata := make(map[string]string)
		if strings.TrimSpace(subscribeMetadata) != "" {
			if err := decodeSubscribeJSON(subscribeMetadata, &metadata); err != nil {
				return fmt.Errorf("error parsing metadata as JSON: %w", err)
			}
			if metadata == nil {
				metadata = make(map[string]string)
			}
		}

		var routes *standalone.SubscriptionRoutes
		if strings.TrimSpace(subscribeRoutes) != "" {
			routes = &standalone.SubscriptionRoutes{}
			if err := decodeSubscribeJSON(subscribeRoutes, routes); err != nil {
				return fmt.Errorf("error parsing routes as JSON: %w", err)
			}
		}

		if subscribeSocket != "" {
			if runtime.GOOS == "windows" {
				return errors.New("the unix-domain-socket option is not supported on Windows")
			}
			print.WarningStatusEvent(os.Stdout, "The --unix-domain-socket option does not affect subscription delivery; Dapr sends events to the configured app port")
		}

		ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt)
		defer stop()

		client := standalone.NewClient()
		print.InfoStatusEvent(os.Stdout, "Starting subscription for app %s on topic %s in pubsub %s", subscribeAppID, subscribeTopic, subscribePubsub)
		err := client.Subscribe(ctx, standalone.SubscribeOptions{
			SubscribeAppID: subscribeAppID,
			PubsubName:     subscribePubsub,
			Topic:          subscribeTopic,
			Metadata:       metadata,
			Routes:         routes,
			Output:         standalone.SubscribeOutput(strings.ToLower(subscribeOutput)),
			Response:       standalone.SubscribeResponse(strings.ToUpper(subscribeResponse)),
			Writer:         os.Stdout,
		})
		if err != nil {
			return fmt.Errorf("error subscribing app %s: %w", subscribeAppID, err)
		}

		print.SuccessStatusEvent(os.Stdout, "Subscription stopped")
		return nil
	},
}

func decodeSubscribeJSON(raw string, destination any) error {
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	if err := decoder.Decode(new(any)); err == nil {
		return errors.New("unexpected trailing JSON value")
	} else if !errors.Is(err, io.EOF) {
		return err
	}
	return nil
}

func init() {
	SubscribeCmd.Flags().StringVarP(&subscribeAppID, "subscribe-app-id", "a", "", "The ID of the subscriber app")
	SubscribeCmd.Flags().StringVarP(&subscribePubsub, "pubsub", "p", "", "The name of the pub/sub component")
	SubscribeCmd.Flags().StringVarP(&subscribeTopic, "topic", "t", "", "The topic to subscribe to")
	SubscribeCmd.Flags().StringVarP(&subscribeMetadata, "metadata", "m", "", "The JSON serialized subscription metadata (optional)")
	SubscribeCmd.Flags().StringVarP(&subscribeRoutes, "routes", "r", "", "The JSON serialized routing configuration (optional)")
	SubscribeCmd.Flags().StringVarP(&subscribeSocket, "unix-domain-socket", "u", "", "Unix domain socket directory (subscription callbacks still use the app port)")
	SubscribeCmd.Flags().StringVarP(&subscribeOutput, "output", "o", string(standalone.SubscribeOutputData), "Output fields: data, time, metadata, or all")
	SubscribeCmd.Flags().StringVar(&subscribeResponse, "response", "success", "Response to each event: success, retry, or drop")
	SubscribeCmd.Flags().BoolP("help", "h", false, "Print this help message")

	_ = SubscribeCmd.MarkFlagRequired("subscribe-app-id")
	_ = SubscribeCmd.MarkFlagRequired("topic")
	_ = SubscribeCmd.MarkFlagRequired("pubsub")

	RootCmd.AddCommand(SubscribeCmd)
}
