/*
Copyright 2023 The Dapr Authors
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

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type handler struct{}

type Metrics struct {
	MetricsID int `json:"metricsID"`
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Received request: ", r.Method)
	defer r.Body.Close()
	var metrics Metrics
	err := json.NewDecoder(r.Body).Decode(&metrics)
	if err != nil {
		fmt.Println("Error decoding body: ", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	fmt.Println("Received metrics: ", metrics)
	w.WriteHeader(http.StatusOK)
}

func main() {
	fmt.Println("Starting server in port 9081...")
	StartServer(9081, &handler{})
}

// StartServer starts a HTTP or HTTP2 server
func StartServer(port int, handler http.Handler) {
	// Create a listener
	addr := fmt.Sprintf(":%d", port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Failed to create listener: %v", err)
	}
	//nolint:gosec
	server := &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	// Stop the server when we get a termination signal
	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, syscall.SIGKILL, syscall.SIGTERM, syscall.SIGINT) //nolint:staticcheck
	go func() {
		// Wait for cancelation signal
		<-stopCh
		log.Println("Shutdown signal received")
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		server.Shutdown(ctx)
	}()

	err = server.Serve(ln)

	if err != http.ErrServerClosed {
		log.Fatalf("Failed to run server: %v", err)
	}

	log.Println("Server shut down")
}
