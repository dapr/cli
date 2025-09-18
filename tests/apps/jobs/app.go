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
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/dapr/go-sdk/client"
	"github.com/dapr/kit/signals"
)

func main() {
	const port = 9084

	ctx := signals.Context()

	fmt.Printf("Starting server in port %v...\n", port)

	regCh := make(chan struct{})
	mux := http.NewServeMux()
	mux.HandleFunc("/dapr/config", func(w http.ResponseWriter, r *http.Request) {
		close(regCh)
		w.Write([]byte(`{"entities": ["myactortype"]}`))
	})
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {})

	go func() {
		<-regCh
		register(ctx)
	}()

	StartServer(ctx, port, mux)
}

func register(ctx context.Context) {
	log.Printf("Registering jobs & reminders")

	cl, err := client.NewClient()
	if err != nil {
		log.Fatal(err)
	}

	ds := time.Now().Format(time.RFC3339)

	if err = cl.ScheduleJobAlpha1(ctx, &client.Job{
		Name:     "test1",
		Schedule: "@every 100m",
		Repeats:  1234,
		DueTime:  ds,
	}); err != nil {
		log.Fatal(err)
	}

	if err = cl.ScheduleJobAlpha1(ctx, &client.Job{
		Name:     "test2",
		Schedule: "@every 100m",
		Repeats:  56788,
		DueTime:  ds,
		TTL:      "10000s",
	}); err != nil {
		log.Fatal(err)
	}

	if err = cl.RegisterActorReminder(ctx, &client.RegisterActorReminderRequest{
		ActorType: "myactortype",
		ActorID:   "actorid1",
		Name:      "test1",
		DueTime:   ds,
		Period:    "R100/PT10000S",
	}); err != nil {
		log.Fatal(err)
	}

	if err = cl.RegisterActorReminder(ctx, &client.RegisterActorReminderRequest{
		ActorType: "myactortype",
		ActorID:   "actorid2",
		Name:      "test2",
		DueTime:   ds,
		Period:    "R100/PT10000S",
	}); err != nil {
		log.Fatal(err)
	}
}

// StartServer starts a HTTP or HTTP2 server
func StartServer(ctx context.Context, port int, handler http.Handler) {
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

	go func() {
		// Wait for cancelation signal
		<-ctx.Done()
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
