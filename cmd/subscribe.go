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
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/phayes/freeport"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	daprRuntime "github.com/dapr/dapr/pkg/runtime"

	cmdruntime "github.com/dapr/cli/cmd/runtime"
	"github.com/dapr/cli/pkg/print"
	runExec "github.com/dapr/cli/pkg/runexec"
	"github.com/dapr/cli/pkg/runfileconfig"
	"github.com/dapr/cli/pkg/standalone"
	daprsyscall "github.com/dapr/cli/pkg/syscall"
	"github.com/dapr/cli/utils"
)

var (
	subscribeAppID                string
	subscribeSocket               string
	subscribeMetadata             map[string]string
	subscribeTo                   []string
	subscribeConfigFile           string
	subscribeHTTPPort             int
	subscribeGRPCPort             int
	subscribeMetricsPort          int
	subscribeLogLevel             string
	subscribeResourcesPaths       []string
	subscribeSchedulerAddress     string
	subscribeAPIListenAddresses   string
	subscribeEnableAPILogging     bool
	subscribeServePort            int
	subscribeServeSubscriptionsJS string
)

// Flags shared with "dapr run" for the sidecar.
var (
	subscribeInternalGRPCPort   int
	subscribeProfilePort        int
	subscribeEnableProfiling    bool
	subscribeMaxRequestBodySize string
	subscribeReadBufferSize     string
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

# Subscribe with a specific Dapr HTTP port
dapr subscribe --app-id myapp --dapr-http-port 3500 --to target:sample

# Subscribe using Unix domain socket
dapr subscribe --unix-domain-socket /tmp --app-id myapp --to target:sample

# Subscribe specifying dapr runtime installation directory
dapr subscribe --app-id myapp --to target:sample --runtime-path /usr/local/dapr
`,
	Args: cobra.NoArgs,
	PreRun: func(cmd *cobra.Command, args []string) {
		viper.BindPFlag("placement-host-address", cmd.Flags().Lookup("placement-host-address"))
	},
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

		if err = runSubscribe(cmd, subs); err != nil {
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
		if err := json.Unmarshal([]byte(subscribeServeSubscriptionsJS), &subs); err != nil {
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

// buildSubscribeApp creates the runfileconfig.App describing the daprd sidecar
// and the subscriber app (this binary re-executed as "subscribe serve").
func buildSubscribeApp(cmd *cobra.Command) (runfileconfig.App, error) {
	daprDirPath, err := standalone.GetDaprRuntimePath(cmdruntime.GetDaprRuntimePath())
	if err != nil {
		return runfileconfig.App{}, fmt.Errorf("failed to get Dapr install directory: %w", err)
	}

	app := runfileconfig.App{
		RunConfig: standalone.RunConfig{
			AppID:            subscribeAppID,
			UnixDomainSocket: subscribeSocket,
		},
	}

	// Set defaults for everything not given below.
	app.SetDefaultFromSchema()

	app.ConfigFile = subscribeConfigFile
	if app.ConfigFile == "" {
		app.ConfigFile = standalone.GetDaprConfigPath(daprDirPath)
	}
	app.ComponentsPath = standalone.GetDaprComponentsPath(daprDirPath)
	app.ResourcesPaths = subscribeResourcesPaths
	app.LogLevel = subscribeLogLevel
	app.HTTPPort = subscribeHTTPPort
	app.GRPCPort = subscribeGRPCPort
	app.MetricsPort = subscribeMetricsPort
	app.InternalGRPCPort = subscribeInternalGRPCPort
	app.ProfilePort = subscribeProfilePort
	app.EnableProfiling = subscribeEnableProfiling
	app.MaxRequestBodySize = subscribeMaxRequestBodySize
	app.HTTPReadBufferSize = subscribeReadBufferSize
	app.EnableAPILogging = subscribeEnableAPILogging
	app.APIListenAddresses = subscribeAPIListenAddresses
	app.DaprdInstallPath = cmdruntime.GetDaprRuntimePath()

	// Logs of both daprd and the subscriber app go to the console.
	app.AppLogDestination = standalone.Console
	app.DaprdLogDestination = standalone.Console

	if subscribeSocket != "" {
		// use unix domain socket means no port any more.
		app.HTTPPort = 0
		app.GRPCPort = 0
		// The put*InMeta helpers shared with "dapr run" read this package-level variable.
		unixDomainSocket = subscribeSocket
	}

	// placement-host-address flag handling: only set pointer if flag was explicitly changed
	if cmd.Flags().Changed("placement-host-address") {
		val := viper.GetString("placement-host-address")
		app.PlacementHostAddr = &val // may be empty => disable
	}

	// scheduler-host-address flag handling: only set pointer if flag was explicitly changed
	if cmd.Flags().Changed("scheduler-host-address") {
		val := subscribeSchedulerAddress
		app.SchedulerHostAddress = &val // may be empty => disable
	}

	return app, nil
}

// runSubscribe starts the daprd sidecar and the subscriber app and blocks until a shutdown signal.
func runSubscribe(cmd *cobra.Command, subs []subscription) error {
	app, err := buildSubscribeApp(cmd)
	if err != nil {
		return err
	}

	// setup shutdown notify channel.
	sigCh := make(chan os.Signal, 1)
	daprsyscall.SetupShutdownNotify(sigCh)

	// Creates a separate process group ID for current process, so that daprd and
	// the app (and their grandchildren) can be controlled together.
	daprsyscall.CreateProcessGroupID()

	print.StatusEvent(os.Stdout, print.LogInfo, "Validating config and starting app %q", app.AppID)

	// Adjust scheduler host address defaults (pointer-aware)
	var schedIn string
	if app.SchedulerHostAddress != nil {
		schedIn = *app.SchedulerHostAddress
	}
	schedOut := validateSchedulerHostAddress(daprVer.RuntimeVersion, schedIn)
	if schedOut != "" {
		app.SchedulerHostAddress = &schedOut
	}
	// Validate validates the configs and allocates the Dapr sidecar ports.
	if err = app.Validate(); err != nil {
		return fmt.Errorf("error validating run config for app %q: %w", app.AppID, err)
	}

	// Unlike `dapr run` without an app command, subscribe always has an
	// application. Allocate its port so daprd initializes the app channel.
	app.AppPort, err = freeport.GetFreePort()
	if err != nil {
		return fmt.Errorf("failed to allocate subscriber app port: %w", err)
	}

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("could not find the dapr executable: %w", err)
	}
	payload, err := json.Marshal(subs)
	if err != nil {
		return err
	}
	app.Command = []string{exe, "subscribe", "serve", "--port", strconv.Itoa(app.AppPort), "--subscriptions", string(payload)}

	// Get Run Config for the subscriber app and reuse the same process startup
	// and output forwarding used by `dapr run`.
	runConfig := app.RunConfig
	daprdLogWriter := runfileconfig.GetLogWriter(nil, app.DaprdLogDestination)
	appDaprdWriter := runExec.GetAppDaprdWriter(app, false)
	appLogWriter := runfileconfig.GetLogWriter(nil, app.AppLogDestination)
	customAppLogWriter := print.CustomLogWriter{W: appLogWriter}

	runState, err := startDaprdAndAppProcesses(&runConfig, app.AppDirPath, sigCh,
		daprdLogWriter, daprdLogWriter, customAppLogWriter, customAppLogWriter)
	if err != nil {
		print.StatusEvent(appDaprdWriter, print.LogFailure, "Error starting Dapr and app (%q): %s", app.AppID, err.Error())
		return err
	}

	// Metadata API is only available if app has started listening to port, so wait for app to start before calling metadata API.
	putCLIProcessIDInMeta(runState, os.Getpid())

	if runState.AppCMD.Command != nil {
		putAppCommandInMeta(runConfig, runState)

		if runState.AppCMD.Command.Process != nil {
			putAppProcessIDInMeta(runState)
			// Attach a windows job object to the app process.
			utils.AttachJobObjectToProcess(strconv.Itoa(os.Getpid()), runState.AppCMD.Command.Process)
		}
	}

	for _, s := range subs {
		print.StatusEvent(runState.DaprCMD.OutputWriter, print.LogInfo, "Subscribed to %s/%s on route %s", s.PubsubName, s.Topic, s.Route)
	}
	print.StatusEvent(runState.DaprCMD.OutputWriter, print.LogSuccess, "You're up and running! Received events will appear here.\n")

	// Wait for the signal to stop.
	<-sigCh
	// To add a new line in Stdout.
	fmt.Println()
	print.InfoStatusEvent(os.Stdout, "Received signal to stop Dapr and the subscriber app. Shutting down Dapr and the subscriber app.")

	// Stop daprd and the subscriber app using the same shutdown path as `dapr run`.
	stopDaprdAndAppProcesses(runState)

	if runConfig.UnixDomainSocket != "" {
		for _, s := range []string{"http", "grpc"} {
			os.Remove(utils.GetSocket(runConfig.UnixDomainSocket, runConfig.AppID, s))
		}
	}

	return nil
}

// serveSubscriptions runs the subscriber app: it tells Dapr which routes to
// call and prints every event it receives.
func serveSubscriptions(port int, subs []subscription) error {
	if port <= 0 || port > 65535 {
		return fmt.Errorf("invalid subscriber HTTP port %d", port)
	}

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

			// The parent process captures the subscriber process's stdout and
			// forwards it to the terminal, just like `dapr run` app logs.
			fmt.Printf("%s %s %s\n", time.Now().Format(time.RFC3339), route, body)

			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"status":"SUCCESS"}`)
		})
	}

	return http.ListenAndServe(fmt.Sprintf(":%d", port), mux)
}

func init() {
	SubscribeCmd.Flags().StringVarP(&subscribeAppID, "app-id", "a", "", "The ID of the subscriber app")
	SubscribeCmd.Flags().StringVarP(&subscribeConfigFile, "config", "c", "", "Dapr configuration file")
	SubscribeCmd.Flags().IntVarP(&subscribeHTTPPort, "dapr-http-port", "H", -1, "The HTTP port for Dapr to listen on")
	SubscribeCmd.Flags().IntVarP(&subscribeGRPCPort, "dapr-grpc-port", "G", -1, "The gRPC port for Dapr to listen on")
	SubscribeCmd.Flags().IntVarP(&subscribeMetricsPort, "metrics-port", "M", -1, "The port of metrics on dapr")
	SubscribeCmd.Flags().IntVarP(&subscribeInternalGRPCPort, "dapr-internal-grpc-port", "I", -1, "The gRPC port for the Dapr internal API to listen on")
	SubscribeCmd.Flags().BoolVar(&subscribeEnableProfiling, "enable-profiling", false, "Enable pprof profiling via an HTTP endpoint")
	SubscribeCmd.Flags().IntVarP(&subscribeProfilePort, "profile-port", "", -1, "The port for the profile server to listen on")
	SubscribeCmd.Flags().StringVarP(&subscribeMaxRequestBodySize, "max-body-size", "", strconv.Itoa(daprRuntime.DefaultMaxRequestBodySize>>20)+"Mi", "Max size of request body in MB")
	SubscribeCmd.Flags().StringVarP(&subscribeReadBufferSize, "read-buffer-size", "", strconv.Itoa(daprRuntime.DefaultReadBufferSize>>10)+"Ki", "HTTP header read buffer in KB")
	SubscribeCmd.Flags().StringVarP(&subscribeLogLevel, "log-level", "", "info", "The log verbosity. Valid values are: debug, info, warn, error, fatal, or panic")
	SubscribeCmd.Flags().StringSliceVarP(&subscribeResourcesPaths, "resources-path", "", []string{}, "The path for resources directory")
	SubscribeCmd.Flags().String("placement-host-address", "localhost", "The address of the placement service. Format is either <hostname> for default port or <hostname>:<port> for custom port. Set to an empty string to disable placement")
	SubscribeCmd.Flags().StringVarP(&subscribeSchedulerAddress, "scheduler-host-address", "", "localhost", "The address of the scheduler service. Format is either <hostname> for default port or <hostname>:<port> for custom port. Set to an empty string to disable scheduler")
	SubscribeCmd.Flags().StringVar(&subscribeAPIListenAddresses, "dapr-listen-addresses", "", "Comma separated list of IP addresses that sidecar will listen to")
	SubscribeCmd.Flags().BoolVar(&subscribeEnableAPILogging, "enable-api-logging", false, "Log API calls at INFO verbosity. Valid values are: true or false")
	SubscribeCmd.Flags().StringVarP(&subscribeSocket, "unix-domain-socket", "u", "", "Path to a unix domain socket dir. If specified, Dapr API servers will use Unix Domain Sockets")
	SubscribeCmd.Flags().StringToStringVarP(&subscribeMetadata, "metadata", "m", nil, "Default subscription metadata as key=value, repeatable; a repeated key keeps the last value. Used for every --to without its own ?key=value metadata")
	SubscribeCmd.Flags().StringArrayVarP(&subscribeTo, "to", "t", nil, "Target as <pubsub-name>:<topic>[:/route][?key1=value1&key2=value2], repeatable. The route defaults to /_<pubsub-name>/<topic>")
	SubscribeCmd.Flags().BoolP("help", "h", false, "Print this help message")
	SubscribeCmd.MarkFlagRequired("app-id")
	SubscribeCmd.MarkFlagRequired("to")

	subscribeServeCmd.Flags().IntVar(&subscribeServePort, "port", 0, "The port to listen on")
	subscribeServeCmd.Flags().StringVar(&subscribeServeSubscriptionsJS, "subscriptions", "[]", "The JSON serialized subscriptions")
	SubscribeCmd.AddCommand(subscribeServeCmd)

	RootCmd.AddCommand(SubscribeCmd)
}
