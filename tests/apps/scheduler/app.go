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
	"os"
	"time"

	"github.com/dapr/durabletask-go/workflow"
	"github.com/dapr/go-sdk/client"
	"github.com/dapr/kit/ptr"
	"github.com/dapr/kit/signals"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func main() {
	const port = 9095

	ctx := signals.Context()

	fmt.Printf("Starting server in port %v...\n", port)

	regCh := make(chan struct{})
	mux := http.NewServeMux()
	mux.HandleFunc("/dapr/config", func(w http.ResponseWriter, r *http.Request) {
		close(regCh)
		w.Write([]byte(`{"entities": ["myactortype"]}`))
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {})

	go func() {
		log.Printf("Waiting for registration call...")
		select {
		case <-regCh:
			log.Printf("Registration call received")
		case <-ctx.Done():
			log.Printf("Context done while waiting for registration call")
			return
		}
		register(ctx)
	}()

	StartServer(ctx, port, mux)
}

func register(ctx context.Context) {
	log.Printf("Registering jobs, reminders and workflows")

	grpcPort := os.Getenv("DAPR_GRPC_PORT")
	if grpcPort == "" {
		grpcPort = "3510"
	}
	addr := "127.0.0.1:" + grpcPort
	log.Printf("Creating client to %s", addr)
	cl, err := client.NewClientWithAddress(addr)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Client created")

	ds := time.Now().Format(time.RFC3339)

	data, err := anypb.New(wrapperspb.String("hello"))
	if err != nil {
		log.Fatal(err)
	}

	if err = cl.ScheduleJobAlpha1(ctx, &client.Job{
		Name:     "test1",
		Schedule: ptr.Of("@every 100m"),
		Repeats:  ptr.Of(uint32(1234)),
		DueTime:  ptr.Of(ds),
		Data:     data,
	}); err != nil {
		log.Fatal(err)
	}

	log.Printf("Scheduled job test1")

	if err = cl.ScheduleJobAlpha1(ctx, &client.Job{
		Name:     "test2",
		Schedule: ptr.Of("@every 100m"),
		Repeats:  ptr.Of(uint32(56788)),
		DueTime:  ptr.Of(ds),
		TTL:      ptr.Of("10000s"),
		Data:     data,
	}); err != nil {
		log.Fatal(err)
	}

	log.Printf("Scheduled job test2")

	if err = cl.RegisterActorReminder(ctx, &client.RegisterActorReminderRequest{
		ActorType: "myactortype",
		ActorID:   "actorid1",
		Name:      "test1",
		DueTime:   ds,
		Period:    "R100/PT10000S",
	}); err != nil {
		log.Fatal(err)
	}

	log.Printf("Scheduled actor reminder test1")

	if err = cl.RegisterActorReminder(ctx, &client.RegisterActorReminderRequest{
		ActorType: "myactortype",
		ActorID:   "actorid2",
		Name:      "test2",
		DueTime:   ds,
		Period:    "R100/PT10000S",
	}); err != nil {
		log.Fatal(err)
	}

	log.Printf("Scheduled actor reminder test2")

	r := workflow.NewRegistry()

	if err := r.AddWorkflow(W1); err != nil {
		log.Fatal(err)
	}
	if err := r.AddWorkflow(W2); err != nil {
		log.Fatal(err)
	}
	if err := r.AddActivity(A1); err != nil {
		log.Fatal(err)
	}

	wf, err := client.NewWorkflowClient()
	if err != nil {
		log.Fatal(err)
	}

	if err = wf.StartWorker(ctx, r); err != nil {
		log.Fatal(err)
	}

	if _, err = wf.ScheduleWorkflow(ctx, "W1", workflow.WithInstanceID("abc1")); err != nil {
		log.Fatal(err)
	}

	log.Printf("Scheduled workflow W1 with id abc1")

	if _, err = wf.ScheduleWorkflow(ctx, "W1", workflow.WithInstanceID("abc2")); err != nil {
		log.Fatal(err)
	}

	log.Printf("Scheduled workflow W1 with id abc2")

	if _, err = wf.ScheduleWorkflow(ctx, "W2", workflow.WithInstanceID("xyz1")); err != nil {
		log.Fatal(err)
	}

	log.Printf("Scheduled workflow W2 with id xyz1")

	if _, err = wf.ScheduleWorkflow(ctx, "W2", workflow.WithInstanceID("xyz2")); err != nil {
		log.Fatal(err)
	}

	log.Printf("Scheduled workflow W2 with id xyz2")
}

// StartServer starts a HTTP or HTTP2 server
func StartServer(ctx context.Context, port int, handler http.Handler) {
	// Create a listener
	addr := fmt.Sprintf(":%d", port)

	log.Println("Starting server on ", addr)

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

	log.Printf("Server listening on %s", addr)
	err = server.Serve(ln)

	if err != http.ErrServerClosed {
		log.Fatalf("Failed to run server: %v", err)
	}

	log.Println("Server shut down")
}

func W1(ctx *workflow.WorkflowContext) (any, error) {
	return nil, ctx.CreateTimer(time.Hour * 50).Await(nil)
}

func W2(ctx *workflow.WorkflowContext) (any, error) {
	return nil, ctx.CallActivity("A1").Await(nil)
}

func A1(ctx workflow.ActivityContext) (any, error) {
	<-ctx.Context().Done()
	return nil, nil
}
