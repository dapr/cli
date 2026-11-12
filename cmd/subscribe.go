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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/dapr/cli/pkg/print"
)

var (
	subscribeAppID    string
	subscribeAppPort  int
	subscribeMetadata map[string]string
	subscribeTo       []string
)

// subscribeTarget is a parsed --to value: <pubsub-name>:<topic>[:/route].
type subscribeTarget struct {
	PubsubName string
	Topic      string
	Route      string
}

// subscription is the programmatic subscription returned from GET /dapr/subscribe.
type subscription struct {
	PubsubName string            `json:"pubsubname"`
	Topic      string            `json:"topic"`
	Route      string            `json:"route"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

var SubscribeCmd = &cobra.Command{
	Use:   "subscribe",
	Short: "Subscribe to topic(s) of pub/sub component(s) and print received events. Supported platforms: Self-hosted",
	Example: `
# Subscribe to a topic, route defaults to /_target/sample
dapr subscribe --to target:sample

# Subscribe with an explicit route
dapr subscribe --to target:sample:/sample

# Subscribe to several topics without cloud event
dapr subscribe --app-id myapp --app-port 3000 --to target:sample:/sample --to other:events --metadata rawPayload=true
`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		targets, err := parseSubscribeTargets(subscribeTo)
		if err != nil {
			print.FailureStatusEvent(os.Stderr, err.Error())
			os.Exit(1)
		}

		if err = runSubscribe(targets, subscribeMetadata); err != nil {
			print.FailureStatusEvent(os.Stderr, fmt.Sprintf("Error subscribing: %s", err))
			os.Exit(1)
		}
	},
}

// parseSubscribeTargets parses every --to value and rejects duplicate
// pubsub/topic pairs and duplicate routes.
func parseSubscribeTargets(values []string) ([]subscribeTarget, error) {
	targets := make([]subscribeTarget, 0, len(values))
	topics := map[string]struct{}{}
	routes := map[string]struct{}{}

	for _, v := range values {
		// The pubsub name and topic must not contain ':'; the route may.
		parts := strings.SplitN(v, ":", 3)
		if len(parts) < 2 {
			return nil, fmt.Errorf("invalid --to %q: expected <pubsub-name>:<topic>[:/route]", v)
		}

		t := subscribeTarget{
			PubsubName: strings.TrimSpace(parts[0]),
			Topic:      strings.TrimSpace(parts[1]),
		}
		if t.PubsubName == "" || t.Topic == "" {
			return nil, fmt.Errorf("invalid --to %q: pubsub name and topic must not be empty", v)
		}

		if len(parts) == 3 {
			t.Route = strings.TrimSpace(parts[2])
			if !strings.HasPrefix(t.Route, "/") {
				return nil, fmt.Errorf("invalid --to %q: route must start with '/'", v)
			}
		} else {
			t.Route = "/_" + t.PubsubName + "/" + t.Topic
		}

		key := t.PubsubName + ":" + t.Topic
		if _, ok := topics[key]; ok {
			return nil, fmt.Errorf("duplicate --to for pubsub %q and topic %q", t.PubsubName, t.Topic)
		}
		if _, ok := routes[t.Route]; ok {
			return nil, fmt.Errorf("route %q is used by more than one --to", t.Route)
		}
		topics[key] = struct{}{}
		routes[t.Route] = struct{}{}
		targets = append(targets, t)
	}
	return targets, nil
}

func runSubscribe(targets []subscribeTarget, metadata map[string]string) error {
	subs := make([]subscription, 0, len(targets))
	mux := http.NewServeMux()

	for _, t := range targets {
		route := t.Route
		subs = append(subs, subscription{
			PubsubName: t.PubsubName,
			Topic:      t.Topic,
			Route:      route,
			Metadata:   metadata,
		})

		mux.HandleFunc(route, func(w http.ResponseWriter, r *http.Request) {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			fmt.Printf("%s %s %s\n", time.Now().Format(time.RFC3339), route, body)

			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"status":"SUCCESS"}`)
		})
	}

	mux.HandleFunc("/dapr/subscribe", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(subs)
	})

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", subscribeAppPort))
	if err != nil {
		return err
	}
	port := listener.Addr().(*net.TCPAddr).Port

	srv := &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	errCh := make(chan error, 1)
	go func() { errCh <- srv.Serve(listener) }()

	appID := subscribeAppID
	if appID == "" {
		appID = "<app-id>"
	}
	print.InfoStatusEvent(os.Stdout, "Listening on port %d. Start the sidecar with: dapr run --app-id %s --app-port %d", port, appID, port)
	for _, s := range subs {
		print.InfoStatusEvent(os.Stdout, "Subscribed to %s/%s on route %s", s.PubsubName, s.Topic, s.Route)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case err = <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}

func init() {
	SubscribeCmd.Flags().StringVarP(&subscribeAppID, "app-id", "a", "", "The ID of the subscriber app (used in the sidecar start hint)")
	SubscribeCmd.Flags().IntVarP(&subscribeAppPort, "app-port", "p", 3000, "The port to listen on for Dapr, 0 picks a free port")
	SubscribeCmd.Flags().StringToStringVarP(&subscribeMetadata, "metadata", "m", nil, "Subscription metadata as key=value, repeatable; a repeated key keeps the last value")
	SubscribeCmd.Flags().StringArrayVarP(&subscribeTo, "to", "t", nil, "Target as <pubsub-name>:<topic>[:/route], repeatable; route defaults to /_<pubsub-name>/<topic>")
	SubscribeCmd.Flags().BoolP("help", "h", false, "Print this help message")
	SubscribeCmd.MarkFlagRequired("to")
	RootCmd.AddCommand(SubscribeCmd)
}
