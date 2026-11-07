/*
Copyright 2021 The Dapr Authors
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
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

type subscribeTestProcess struct {
	instances []ListOutput
	err       error
}

func (p *subscribeTestProcess) List() ([]ListOutput, error) {
	return p.instances, p.err
}

type failingSubscribeWriter struct{}

func (failingSubscribeWriter) Write([]byte) (int, error) {
	return 0, errors.New("writer failed")
}

func TestValidateSubscribeOptions(t *testing.T) {
	valid := func() SubscribeOptions {
		return SubscribeOptions{
			SubscribeAppID: "orders",
			PubsubName:     "pubsub",
			Topic:          "orders",
			Writer:         io.Discard,
		}
	}

	tests := []struct {
		name    string
		mutate  func(*SubscribeOptions)
		wantErr string
	}{
		{
			name: "missing app id",
			mutate: func(o *SubscribeOptions) {
				o.SubscribeAppID = ""
			},
			wantErr: "subscribeAppID is missing",
		},
		{
			name: "missing pubsub",
			mutate: func(o *SubscribeOptions) {
				o.PubsubName = ""
			},
			wantErr: "pubsubName is missing",
		},
		{
			name: "missing topic",
			mutate: func(o *SubscribeOptions) {
				o.Topic = ""
			},
			wantErr: "topic is missing",
		},
		{
			name: "missing output writer",
			mutate: func(o *SubscribeOptions) {
				o.Writer = nil
			},
			wantErr: "subscription output writer is missing",
		},
		{
			name: "invalid output",
			mutate: func(o *SubscribeOptions) {
				o.Output = "verbose"
			},
			wantErr: "invalid output",
		},
		{
			name: "invalid response",
			mutate: func(o *SubscribeOptions) {
				o.Response = "ignore"
			},
			wantErr: "invalid response",
		},
		{
			name: "empty routes",
			mutate: func(o *SubscribeOptions) {
				o.Routes = &SubscriptionRoutes{}
			},
			wantErr: "routes must define a default path",
		},
		{
			name: "relative default path",
			mutate: func(o *SubscribeOptions) {
				o.Routes = &SubscriptionRoutes{Default: "orders"}
			},
			wantErr: "invalid default route",
		},
		{
			name: "rule missing match",
			mutate: func(o *SubscribeOptions) {
				o.Routes = &SubscriptionRoutes{Rules: []SubscriptionRouteRule{{Path: "/orders"}}}
			},
			wantErr: "route rule 0 is missing match",
		},
		{
			name: "reserved subscription path",
			mutate: func(o *SubscribeOptions) {
				o.Routes = &SubscriptionRoutes{Default: "/dapr/subscribe"}
			},
			wantErr: "subscription routes must not use /dapr/subscribe",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := valid()
			tt.mutate(&opts)
			err := validateSubscribeOptions(&opts)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestSubscribeReturnsListErrors(t *testing.T) {
	wantErr := errors.New("list failed")
	client := &Standalone{process: &subscribeTestProcess{err: wantErr}}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := client.Subscribe(context.Background(), SubscribeOptions{
		SubscribeAppID: "orders",
		PubsubName:     "pubsub",
		Topic:          "orders",
		Writer:         io.Discard,
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected list error %v, got %v", wantErr, err)
	}

	// A cancelled context must be returned before any process lookup.
	client = &Standalone{process: &subscribeTestProcess{}}
	err = client.Subscribe(ctx, SubscribeOptions{
		SubscribeAppID: "orders",
		PubsubName:     "pubsub",
		Topic:          "orders",
		Writer:         io.Discard,
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestSubscribeRequiresRunningInstanceAndAppPort(t *testing.T) {
	tests := []struct {
		name      string
		instances []ListOutput
		wantErr   string
	}{
		{
			name:      "app not found",
			instances: []ListOutput{{AppID: "another-app", AppPort: 12345}},
			wantErr:   "couldn't find a running Dapr instance",
		},
		{
			name:      "app port missing",
			instances: []ListOutput{{AppID: "orders"}},
			wantErr:   "does not have a valid app port configured",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Standalone{process: &subscribeTestProcess{instances: tt.instances}}
			err := client.Subscribe(context.Background(), SubscribeOptions{
				SubscribeAppID: "orders",
				PubsubName:     "pubsub",
				Topic:          "orders",
				Writer:         io.Discard,
			})
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestSubscribeHTTPHandlersAndOutput(t *testing.T) {
	port := getFreeTCPPort(t)
	var output bytes.Buffer
	client := &Standalone{process: &subscribeTestProcess{instances: []ListOutput{{
		AppID:   "orders",
		AppPort: port,
	}}}}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- client.Subscribe(ctx, SubscribeOptions{
			SubscribeAppID: "orders",
			PubsubName:     "target",
			Topic:          "sample",
			Metadata:       map[string]string{"rawPayload": "true"},
			Routes: &SubscriptionRoutes{
				Rules:   []SubscriptionRouteRule{{Match: `event.type == "widget"`, Path: "/widgets"}},
				Default: "/products",
			},
			Output:   SubscribeOutputAll,
			Response: SubscribeResponseSuccess,
			Writer:   &output,
		})
	}()

	baseURL := "http://127.0.0.1:" + strconv.Itoa(port)
	waitForHTTPServer(t, baseURL+"/dapr/subscribe")

	response, err := http.Get(baseURL + "/dapr/subscribe")
	if err != nil {
		t.Fatalf("GET /dapr/subscribe failed: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.StatusCode)
	}

	var subscriptions []Subscription
	if err := json.NewDecoder(response.Body).Decode(&subscriptions); err != nil {
		t.Fatalf("decode subscription response: %v", err)
	}
	if len(subscriptions) != 1 {
		t.Fatalf("expected one subscription, got %d", len(subscriptions))
	}
	if subscriptions[0].PubsubName != "target" || subscriptions[0].Topic != "sample" {
		t.Fatalf("unexpected subscription: %+v", subscriptions[0])
	}
	if subscriptions[0].Routes.Default != "/products" || len(subscriptions[0].Routes.Rules) != 1 {
		t.Fatalf("unexpected routes: %+v", subscriptions[0].Routes)
	}
	if subscriptions[0].Metadata["rawPayload"] != "true" {
		t.Fatalf("unexpected metadata: %+v", subscriptions[0].Metadata)
	}

	cloudEvent := `{"id":"123","source":"test","specversion":"1.0","type":"widget","data":{"name":"gear"}}`
	request, err := http.NewRequest(http.MethodPost, baseURL+"/widgets", strings.NewReader(cloudEvent))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/cloudevents+json")
	request.Header.Set("Authorization", "should-not-be-printed")
	postResponse, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("POST event failed: %v", err)
	}
	defer postResponse.Body.Close()
	if postResponse.StatusCode != http.StatusOK {
		t.Fatalf("expected event status 200, got %d", postResponse.StatusCode)
	}

	var ack struct {
		Status SubscribeResponse `json:"status"`
	}
	if err := json.NewDecoder(postResponse.Body).Decode(&ack); err != nil {
		t.Fatalf("decode event acknowledgement: %v", err)
	}
	if ack.Status != SubscribeResponseSuccess {
		t.Fatalf("expected SUCCESS acknowledgement, got %q", ack.Status)
	}

	var record struct {
		Time     string         `json:"time"`
		Route    string         `json:"route"`
		Metadata map[string]any `json:"metadata"`
		Data     map[string]any `json:"data"`
	}
	if err := json.Unmarshal(bytes.TrimSpace(output.Bytes()), &record); err != nil {
		t.Fatalf("decode output record %q: %v", output.String(), err)
	}
	if record.Time == "" || record.Route != "/widgets" {
		t.Fatalf("unexpected output time/route: %+v", record)
	}
	if record.Metadata["type"] != "widget" || record.Metadata["id"] != "123" {
		t.Fatalf("unexpected CloudEvent metadata: %+v", record.Metadata)
	}
	if _, ok := record.Metadata["data"]; ok {
		t.Fatalf("event data must not be repeated in metadata: %+v", record.Metadata)
	}
	eventData, ok := record.Data["data"].(map[string]any)
	if !ok || eventData["name"] != "gear" {
		t.Fatalf("unexpected event data: %+v", record.Data)
	}
	if strings.Contains(output.String(), "should-not-be-printed") {
		t.Fatal("authorization header leaked to output")
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Subscribe returned an error after cancellation: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("subscription server did not stop after context cancellation")
	}
}

func TestSubscribeResponseModes(t *testing.T) {
	for _, responseMode := range []SubscribeResponse{
		SubscribeResponseSuccess,
		SubscribeResponseRetry,
		SubscribeResponseDrop,
	} {
		t.Run(string(responseMode), func(t *testing.T) {
			var output bytes.Buffer
			handler := &subscriptionHTTPHandler{
				subscription: Subscription{PubsubName: "pubsub", Topic: "topic", Routes: SubscriptionRoutes{Default: "/topic"}},
				output:       SubscribeOutputData,
				response:     responseMode,
				writer:       &output,
			}
			request := httptest.NewRequest(http.MethodPost, "/topic", strings.NewReader(`{"message":"hello"}`))
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request)

			if recorder.Code != http.StatusOK {
				t.Fatalf("expected status 200, got %d", recorder.Code)
			}
			var ack struct {
				Status SubscribeResponse `json:"status"`
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &ack); err != nil {
				t.Fatalf("decode acknowledgement: %v", err)
			}
			if ack.Status != responseMode {
				t.Fatalf("expected %q, got %q", responseMode, ack.Status)
			}
		})
	}
}

func TestSubscribeRejectsWrongMethod(t *testing.T) {
	handler := &subscriptionHTTPHandler{
		subscription: Subscription{PubsubName: "pubsub", Topic: "topic", Routes: SubscriptionRoutes{Default: "/topic"}},
		output:       SubscribeOutputData,
		response:     SubscribeResponseSuccess,
		writer:       io.Discard,
	}

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/dapr/subscribe", strings.NewReader("")))
	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 for POST /dapr/subscribe, got %d", recorder.Code)
	}

	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/topic", strings.NewReader("")))
	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 for GET /topic, got %d", recorder.Code)
	}
}

func TestSubscribeOutputWriterFailureRetriesDelivery(t *testing.T) {
	handler := &subscriptionHTTPHandler{
		subscription: Subscription{PubsubName: "pubsub", Topic: "topic", Routes: SubscriptionRoutes{Default: "/topic"}},
		output:       SubscribeOutputData,
		response:     SubscribeResponseSuccess,
		writer:       failingSubscribeWriter{},
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/topic", strings.NewReader("event")))
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 after output failure, got %d", recorder.Code)
	}
}

func TestSubscribeSerializesConcurrentOutput(t *testing.T) {
	var output bytes.Buffer
	handler := &subscriptionHTTPHandler{
		subscription: Subscription{PubsubName: "pubsub", Topic: "topic", Routes: SubscriptionRoutes{Default: "/topic"}},
		output:       SubscribeOutputData,
		response:     SubscribeResponseSuccess,
		writer:       &output,
	}

	const deliveries = 32
	var wg sync.WaitGroup
	for i := 0; i < deliveries; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			body := fmt.Sprintf("message-%02d", i)
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/topic", strings.NewReader(body)))
			if recorder.Code != http.StatusOK {
				t.Errorf("delivery %d returned status %d", i, recorder.Code)
			}
		}(i)
	}
	wg.Wait()

	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != deliveries {
		t.Fatalf("expected %d complete output lines, got %d: %q", deliveries, len(lines), output.String())
	}
	seen := make(map[string]bool, deliveries)
	for _, line := range lines {
		if seen[line] {
			t.Fatalf("duplicate output line %q", line)
		}
		seen[line] = true
	}
}

func TestValidateSubscriptionRoutePath(t *testing.T) {
	tests := []struct {
		path    string
		wantErr bool
	}{
		{path: "/orders"},
		{path: "/orders/{id}"},
		{path: "/orders/*"},
		{path: "orders", wantErr: true},
		{path: "/orders?status=ready", wantErr: true},
		{path: "/orders#fragment", wantErr: true},
		{path: "/orders\\admin", wantErr: true},
		{path: "/orders\nadmin", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			err := validateSubscriptionRoutePath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateSubscriptionRoutePath(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
			}
		})
	}
}

func getFreeTCPPort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve an available TCP port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		t.Fatalf("release reserved TCP port: %v", err)
	}
	return port
}

func waitForHTTPServer(t *testing.T, endpoint string) {
	t.Helper()
	client := &http.Client{Timeout: 100 * time.Millisecond}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		response, err := client.Get(endpoint)
		if err == nil {
			_ = response.Body.Close()
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("subscription server did not start at %s", endpoint)
}
