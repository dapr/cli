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

package standalone

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	// maxSubscriptionEventSize limits the amount of memory a single delivery can
	// consume while it is being read by the CLI.
	maxSubscriptionEventSize = 16 << 20

	subscriptionShutdownTimeout = 5 * time.Second
)

// SubscriptionRoutes describes the routes used by a Dapr programmatic
// subscription. A default route is optional when at least one rule is present.
type SubscriptionRoutes struct {
	Rules   []SubscriptionRouteRule `json:"rules,omitempty"`
	Default string                  `json:"default,omitempty"`
}

// SubscriptionRouteRule routes an event matching Match to Path.
type SubscriptionRouteRule struct {
	Match string `json:"match"`
	Path  string `json:"path"`
}

// Subscription is the programmatic subscription returned by GET /dapr/subscribe.
type Subscription struct {
	PubsubName string             `json:"pubsubname"`
	Topic      string             `json:"topic"`
	Routes     SubscriptionRoutes `json:"routes"`
	Metadata   map[string]string  `json:"metadata,omitempty"`
}

// SubscribeOutput determines which event fields are written to the output.
type SubscribeOutput string

const (
	SubscribeOutputData     SubscribeOutput = "data"
	SubscribeOutputTime     SubscribeOutput = "time"
	SubscribeOutputMetadata SubscribeOutput = "metadata"
	SubscribeOutputAll      SubscribeOutput = "all"
)

// SubscribeResponse is the processing status returned to Dapr after delivery.
type SubscribeResponse string

const (
	SubscribeResponseSuccess SubscribeResponse = "SUCCESS"
	SubscribeResponseRetry   SubscribeResponse = "RETRY"
	SubscribeResponseDrop    SubscribeResponse = "DROP"
)

// SubscribeOptions configures a local HTTP application that exposes a Dapr
// programmatic pub/sub subscription and receives events on the app port used
// by the selected Dapr sidecar.
type SubscribeOptions struct {
	SubscribeAppID string
	PubsubName     string
	Topic          string
	Metadata       map[string]string
	Routes         *SubscriptionRoutes
	Output         SubscribeOutput
	Response       SubscribeResponse
	Writer         io.Writer
}

// Subscribe starts an HTTP application for the selected Dapr instance. It
// serves GET /dapr/subscribe and accepts event deliveries through POST requests
// on the configured application port. It runs until ctx is cancelled or the
// HTTP server stops unexpectedly.
func (s *Standalone) Subscribe(ctx context.Context, opts SubscribeOptions) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := validateSubscribeOptions(&opts); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	instances, err := s.process.List()
	if err != nil {
		return err
	}

	instance, err := getDaprInstance(instances, opts.SubscribeAppID)
	if err != nil {
		return err
	}
	if instance.AppPort < 1 || instance.AppPort > 65535 {
		return fmt.Errorf("Dapr instance %q does not have a valid app port configured; start it with --app-port", opts.SubscribeAppID)
	}

	listenerAddress := net.JoinHostPort("127.0.0.1", strconv.Itoa(instance.AppPort))
	listener, err := net.Listen("tcp", listenerAddress)
	if err != nil {
		return fmt.Errorf("failed to listen on %s for app ID %q: %w", listenerAddress, opts.SubscribeAppID, err)
	}

	subscription := Subscription{
		PubsubName: opts.PubsubName,
		Topic:      opts.Topic,
		Routes:     *opts.Routes,
		Metadata:   opts.Metadata,
	}
	handler := &subscriptionHTTPHandler{
		subscription: subscription,
		output:       opts.Output,
		response:     opts.Response,
		writer:       opts.Writer,
	}

	server := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- server.Serve(listener)
	}()

	select {
	case err := <-serveErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("subscription HTTP server failed: %w", err)
		}
		return nil
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), subscriptionShutdownTimeout)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			_ = server.Close()
			return fmt.Errorf("failed to shut down subscription HTTP server: %w", err)
		}

		err := <-serveErr
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("subscription HTTP server failed: %w", err)
		}
		return nil
	}
}

func validateSubscribeOptions(opts *SubscribeOptions) error {
	if opts.SubscribeAppID == "" {
		return errors.New("subscribeAppID is missing")
	}
	if opts.PubsubName == "" {
		return errors.New("pubsubName is missing")
	}
	if opts.Topic == "" {
		return errors.New("topic is missing")
	}
	if opts.Writer == nil {
		return errors.New("subscription output writer is missing")
	}

	if opts.Output == "" {
		opts.Output = SubscribeOutputData
	}
	switch opts.Output {
	case SubscribeOutputData, SubscribeOutputTime, SubscribeOutputMetadata, SubscribeOutputAll:
	default:
		return fmt.Errorf("invalid output %q: expected data, time, metadata, or all", opts.Output)
	}

	if opts.Response == "" {
		opts.Response = SubscribeResponseSuccess
	}
	opts.Response = SubscribeResponse(strings.ToUpper(string(opts.Response)))
	switch opts.Response {
	case SubscribeResponseSuccess, SubscribeResponseRetry, SubscribeResponseDrop:
	default:
		return fmt.Errorf("invalid response %q: expected success, retry, or drop", opts.Response)
	}

	if opts.Routes == nil {
		// A convenient route keeps the command usable without requiring a
		// separate route flag. Dapr will POST events for this topic here.
		opts.Routes = &SubscriptionRoutes{Default: "/" + strings.TrimLeft(opts.Topic, "/")}
	}
	if opts.Routes.Default == "" && len(opts.Routes.Rules) == 0 {
		return errors.New("routes must define a default path or at least one rule")
	}
	if opts.Routes.Default != "" {
		if err := validateSubscriptionRoutePath(opts.Routes.Default); err != nil {
			return fmt.Errorf("invalid default route: %w", err)
		}
	}
	for i, rule := range opts.Routes.Rules {
		if strings.TrimSpace(rule.Match) == "" {
			return fmt.Errorf("route rule %d is missing match", i)
		}
		if err := validateSubscriptionRoutePath(rule.Path); err != nil {
			return fmt.Errorf("invalid path for route rule %d: %w", i, err)
		}
	}

	if _, exists := subscriptionRoutePath(opts.Routes, "/dapr/subscribe"); exists {
		return errors.New("subscription routes must not use /dapr/subscribe")
	}

	return nil
}

func validateSubscriptionRoutePath(route string) error {
	if route == "" || !strings.HasPrefix(route, "/") {
		return errors.New("path must be non-empty and start with '/'")
	}
	if strings.ContainsAny(route, "?#\\") {
		return errors.New("path must not contain a query, fragment, or backslash")
	}
	for _, r := range route {
		if r < 0x20 || r == 0x7f {
			return errors.New("path must not contain control characters")
		}
	}
	return nil
}

// subscriptionRoutePath reports whether path is present in the default route
// or any configured rule path.
func subscriptionRoutePath(routes *SubscriptionRoutes, path string) (string, bool) {
	if routes.Default == path {
		return routes.Default, true
	}
	for _, rule := range routes.Rules {
		if rule.Path == path {
			return rule.Path, true
		}
	}
	return "", false
}

type subscriptionHTTPHandler struct {
	subscription Subscription
	output       SubscribeOutput
	response     SubscribeResponse
	writer       io.Writer
	writeMu      sync.Mutex
}

func (h *subscriptionHTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/dapr/subscribe" {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode([]Subscription{h.subscription}); err != nil {
			// The response has already started; there is no useful recovery action.
			return
		}
		return
	}

	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxSubscriptionEventSize))
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			http.Error(w, "event body exceeds the maximum supported size", http.StatusRequestEntityTooLarge)
		} else {
			http.Error(w, "could not read event body", http.StatusBadRequest)
		}
		return
	}

	if err := h.writeEvent(r, body); err != nil {
		http.Error(w, "could not write event to output", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(struct {
		Status SubscribeResponse `json:"status"`
	}{Status: h.response})
}

func (h *subscriptionHTTPHandler) writeEvent(r *http.Request, body []byte) error {
	receivedAt := time.Now().UTC()
	metadata := getSubscriptionEventMetadata(body, r.Header)

	var record []byte
	switch h.output {
	case SubscribeOutputData:
		record = append(record, body...)
		if !bytes.HasSuffix(record, []byte("\n")) {
			record = append(record, '\n')
		}
	case SubscribeOutputTime:
		record = append(record, receivedAt.Format(time.RFC3339Nano)...)
		record = append(record, '\t')
		record = append(record, body...)
		if !bytes.HasSuffix(record, []byte("\n")) {
			record = append(record, '\n')
		}
	case SubscribeOutputMetadata:
		metadataJSON, err := json.Marshal(metadata)
		if err != nil {
			return err
		}
		record = append(record, metadataJSON...)
		record = append(record, '\t')
		record = append(record, body...)
		if !bytes.HasSuffix(record, []byte("\n")) {
			record = append(record, '\n')
		}
	case SubscribeOutputAll:
		var data any = string(body)
		if json.Valid(body) {
			data = json.RawMessage(body)
		}
		all, err := json.Marshal(struct {
			Time     string         `json:"time"`
			Route    string         `json:"route"`
			Metadata map[string]any `json:"metadata,omitempty"`
			Data     any            `json:"data"`
		}{
			Time:     receivedAt.Format(time.RFC3339Nano),
			Route:    r.URL.Path,
			Metadata: metadata,
			Data:     data,
		})
		if err != nil {
			return err
		}
		record = append(all, '\n')
	default:
		return fmt.Errorf("unsupported output mode %q", h.output)
	}

	// A single writer may receive concurrent deliveries. Serialize complete
	// records so output from different messages cannot interleave.
	h.writeMu.Lock()
	defer h.writeMu.Unlock()
	_, err := h.writer.Write(record)
	return err
}

func getSubscriptionEventMetadata(body []byte, headers http.Header) map[string]any {
	var event map[string]any
	if json.Unmarshal(body, &event) == nil && event != nil {
		if _, hasSpecVersion := event["specversion"]; hasSpecVersion {
			metadata := make(map[string]any, len(event)-1)
			for key, value := range event {
				if key != "data" {
					metadata[key] = value
				}
			}
			return metadata
		}
	}

	metadata := make(map[string]any, len(headers))
	for key, values := range headers {
		switch http.CanonicalHeaderKey(key) {
		case "Authorization", "Cookie", "Dapr-Api-Token", "Set-Cookie":
			// Do not print credentials or cookie values to the terminal.
			continue
		}
		if len(values) == 1 {
			metadata[key] = values[0]
		} else if len(values) > 1 {
			metadata[key] = values
		}
	}
	return metadata
}
