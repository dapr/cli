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
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	cmdruntime "github.com/dapr/cli/cmd/runtime"
	"github.com/dapr/cli/pkg/print"
	"github.com/dapr/cli/pkg/standalone"
	daprsyscall "github.com/dapr/cli/pkg/syscall"
	"github.com/dapr/cli/utils"
)

var (
	subscribeAppID    string
	subscribeAppPort  int
	subscribeSocket   string
	subscribeMetadata map[string]string
	subscribeTo       []string

	// Flags of the hidden "subscribe serve" command.
	subscribeServePort int
	subscribeServeSubs string
)

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
# Subscribe to a topic, the route defaults to /_target/sample
dapr subscribe --app-id myapp --to target:sample

# Subscribe with an explicit route
dapr subscribe --app-id myapp --to target:sample:/sample

# Subscribe with metadata for every target without cloud event
dapr subscribe --app-id myapp --to target:sample --to other:events --metadata rawPayload=true

# Per-target metadata overrides --metadata for that target
dapr subscribe --app-id myapp --to "target:sample:/sample?rawPayload=true&ttlInSeconds=10" --to other:events

# Subscribe using Unix domain socket
dapr subscribe --unix-domain-socket /tmp --app-id myapp --to target:sample
`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		subs, err := parseSubscriptions(subscribeTo, subscribeMetadata)
		if err != nil {
			print.FailureStatusEvent(os.Stderr, err.Error())
			os.Exit(1)
		}

		// TODO(@ep0ll): add Windows support.
		if subscribeSocket != "" {
			if runtime.GOOS == string(windowsOsType) {
				print.FailureStatusEvent(os.Stderr, "The unix-domain-socket option is not supported on Windows")
				os.Exit(1)
			}
			print.WarningStatusEvent(os.Stdout, "Unix domain sockets are currently a preview feature")
		}

		if err = runSubscribe(subs); err != nil {
			print.FailureStatusEvent(os.Stderr, "Error subscribing: %s", err)
			os.Exit(1)
		}
	},
}

// subscribeServeCmd is the hidden command the CLI re-executes itself with to
// run the HTTP server that acts as the subscriber app next to the sidecar.
var subscribeServeCmd = &cobra.Command{
	Use:    "serve",
	Hidden: true,
	Args:   cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		var subs []subscription
		if err := json.Unmarshal([]byte(subscribeServeSubs), &subs); err != nil {
			print.FailureStatusEvent(os.Stderr, "Error parsing subscriptions: %s", err)
			os.Exit(1)
		}
		if err := serveSubscriptions(subscribeServePort, subs); err != nil {
			print.FailureStatusEvent(os.Stderr, "Error serving subscriptions: %s", err)
			os.Exit(1)
		}
	},
}

// parseSubscriptions parses every --to value, formatted as
// <pubsub-name>:<topic>[:/route][?key1=value1&key2=value2].
// Metadata given in the query replaces the --metadata defaults for that target;
// it is only applied from --metadata when the query is omitted.
func parseSubscriptions(values []string, defaultMetadata map[string]string) ([]subscription, error) {
	subs := make([]subscription, 0, len(values))
	topics := map[string]struct{}{}
	routes := map[string]struct{}{}

	for _, v := range values {
		target, rawQuery, hasQuery := strings.Cut(v, "?")

		// The pubsub name and topic must not contain ':'; the route may.
		parts := strings.SplitN(target, ":", 3)
		if len(parts) < 2 {
			return nil, fmt.Errorf("invalid --to %q: expected <pubsub-name>:<topic>[:/route][?key=value&...]", v)
		}

		s := subscription{
			PubsubName: strings.TrimSpace(parts[0]),
			Topic:      strings.TrimSpace(parts[1]),
		}
		if s.PubsubName == "" || s.Topic == "" {
			return nil, fmt.Errorf("invalid --to %q: pubsub name and topic must not be empty", v)
		}

		if len(parts) == 3 {
			s.Route = strings.TrimSpace(parts[2])
			if !strings.HasPrefix(s.Route, "/") {
				return nil, fmt.Errorf("invalid --to %q: route must start with '/'", v)
			}
		} else {
			s.Route = "/_" + s.PubsubName + "/" + s.Topic
		}
		if strings.ContainsAny(s.Route, " \t\r\n{}") {
			return nil, fmt.Errorf("invalid --to %q: route must not contain whitespace or braces", v)
		}

		if hasQuery {
			query, err := url.ParseQuery(rawQuery)
			if err != nil {
				return nil, fmt.Errorf("invalid --to %q: bad metadata: %w", v, err)
			}
			s.Metadata = make(map[string]string, len(query))
			for k, vals := range query {
				if k == "" {
					return nil, fmt.Errorf("invalid --to %q: empty metadata key", v)
				}
				s.Metadata[k] = vals[len(vals)-1] // last value wins
			}
		} else if len(defaultMetadata) > 0 {
			s.Metadata = make(map[string]string, len(defaultMetadata))
			for k, val := range defaultMetadata {
				s.Metadata[k] = val
			}
		}

		key := s.PubsubName + ":" + s.Topic
		if _, ok := topics[key]; ok {
			return nil, fmt.Errorf("duplicate --to for pubsub %q and topic %q", s.PubsubName, s.Topic)
		}
		if _, ok := routes[s.Route]; ok {
			return nil, fmt.Errorf("route %q is used by more than one --to", s.Route)
		}
		topics[key] = struct{}{}
		routes[s.Route] = struct{}{}
		subs = append(subs, s)
	}
	return subs, nil
}

// runSubscribe starts the daprd sidecar and the subscriber app (this binary
// re-executed as "subscribe serve") and blocks until a shutdown signal.
func runSubscribe(subs []subscription) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("could not find the dapr executable: %w", err)
	}

	appPort := subscribeAppPort
	payload, err := json.Marshal(subs)
	if err != nil {
		return err
	}

	daprDirPath, err := standalone.GetDaprRuntimePath(cmdruntime.GetDaprRuntimePath())
	if err != nil {
		return fmt.Errorf("failed to get Dapr install directory: %w", err)
	}

	runConfig := standalone.RunConfig{
		AppID:             subscribeAppID,
		AppPort:           appPort,
		UnixDomainSocket:  subscribeSocket,
		Command: []string{
			exe, "subscribe", "serve",
			"--port", strconv.Itoa(appPort),
			"--subscriptions", string(payload),
		},
		SharedRunConfig: standalone.SharedRunConfig{
			ConfigFile:         standalone.GetDaprConfigPath(daprDirPath),
			ComponentsPath:     standalone.GetDaprComponentsPath(daprDirPath),
			LogLevel:           "info",
			DaprdInstallPath:   cmdruntime.GetDaprRuntimePath(),
		},
	}

	if subscribeSocket != "" {
		// The helpers shared with "dapr run" read this package-level variable.
		unixDomainSocket = subscribeSocket
	}

	// Validate modifies the ports to free ports and checks the app id.
	if err = runConfig.Validate(); err != nil {
		return fmt.Errorf("invalid run configuration: %w", err)
	}

	sigCh := make(chan os.Signal, 1)
	daprsyscall.SetupShutdownNotify(sigCh)
	// Same as "dapr run -f": put daprd and the app in a process group of their own.
	daprsyscall.CreateProcessGroupID()

	runState, err := startDaprdAndAppProcesses(&runConfig, "", sigCh, os.Stdout, os.Stderr, os.Stdout, os.Stderr)
	if err != nil {
		return err
	}

	// Metadata API is only available once the app is listening, which is guaranteed here.
	putCLIProcessIDInMeta(runState, os.Getpid())
	if runState.AppCMD.Command != nil {
		putAppCommandInMeta(runConfig, runState)
		if runState.AppCMD.Command.Process != nil {
			putAppProcessIDInMeta(runState)
		}
	}

	for _, s := range subs {
		print.InfoStatusEvent(os.Stdout, "Subscribed to %s/%s on route %s", s.PubsubName, s.Topic, s.Route)
	}
	print.SuccessStatusEvent(os.Stdout, "You're up and running! Received events will appear here.\n")

	<-sigCh
	fmt.Println()
	print.InfoStatusEvent(os.Stdout, "Received signal to stop Dapr and the subscriber app. Shutting down.")

	exitWithError := stopDaprdAndAppProcesses(runState)

	if subscribeSocket != "" {
		for _, s := range []string{"http", "grpc"} {
			os.Remove(utils.GetSocket(subscribeSocket, runState.AppID, s))
		}
	}

	if exitWithError {
		return errors.New("error while shutting down")
	}
	return nil
}

// serveSubscriptions runs the subscriber app: it tells Dapr which routes to
// call and prints every event it receives.
func serveSubscriptions(port int, subs []subscription) error {
	mux := http.NewServeMux()

	mux.HandleFunc("/dapr/subscribe", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(subs)
	})

	for _, s := range subs {
		route := s.Route
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

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() { errCh <- srv.ListenAndServe() }()

	select {
	case err := <-errCh:
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
	SubscribeCmd.Flags().StringVarP(&subscribeAppID, "app-id", "a", "", "The ID of the subscriber app")
	SubscribeCmd.Flags().IntVarP(&subscribeAppPort, "app-port", "p", -1, "The port the subscriber app listens on, 0 picks a free port")
	SubscribeCmd.Flags().StringVarP(&subscribeSocket, "unix-domain-socket", "u", "", "Path to a unix domain socket dir. If specified, Dapr API servers will use Unix Domain Sockets")
	SubscribeCmd.Flags().StringToStringVarP(&subscribeMetadata, "metadata", "m", nil, "Default subscription metadata as key=value, repeatable; a repeated key keeps the last value. Used for every --to without its own ?key=value metadata")
	SubscribeCmd.Flags().StringArrayVarP(&subscribeTo, "to", "t", nil, "Target as <pubsub-name>:<topic>[:/route][?key1=value1&key2=value2], repeatable. The route defaults to /_<pubsub-name>/<topic>")
	SubscribeCmd.Flags().BoolP("help", "h", false, "Print this help message")
	SubscribeCmd.MarkFlagRequired("app-id")
	SubscribeCmd.MarkFlagRequired("to")

	subscribeServeCmd.Flags().IntVar(&subscribeServePort, "port", 0, "The port to listen on")
	subscribeServeCmd.Flags().StringVar(&subscribeServeSubs, "subscriptions", "[]", "The JSON serialized subscriptions")
	SubscribeCmd.AddCommand(subscribeServeCmd)

	RootCmd.AddCommand(SubscribeCmd)
}
