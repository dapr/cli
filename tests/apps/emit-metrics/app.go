package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"
)

type Metrics struct {
	MetricsID int `json:"metricsID"`
}

func main() {
	var host string
	var port string
	client := http.Client{}
	if val, ok := os.LookupEnv("DAPR_HTTP_PORT"); !ok {
		log.Fatalf("DAPR_HTTP_PORT not automatically injected")
	} else {
		log.Println("DAPR_HTTP_PORT set to", val)
		port = val
	}
	// DAPR_HOST_ADD needs to be an env set in dapr.yaml file
	if val, ok := os.LookupEnv("DAPR_HOST_ADD"); !ok {
		log.Fatalf("DAPR_HOST_ADD not set")
	} else {
		log.Println("DAPR_HOST_ADD set to", val)
		host = val
	}
	finalURL := "http://" + host + ":" + port + "/metrics"
	log.Println("Sending metrics to ", finalURL)
	for i := 0; i < 2000; i++ {
		time.Sleep(1 * time.Second)
		metrics := Metrics{
			MetricsID: i,
		}
		b, err := json.Marshal(metrics)
		if err != nil {
			log.Println("Got error while marshalling metrics ", err)
			continue
		}
		// Send metrics to Dapr
		req, _ := http.NewRequest(http.MethodPost, finalURL, bytes.NewBuffer(b))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("dapr-app-id", "processor")
		r, err := client.Do(req)
		if err != nil {
			log.Println("Got error while sending a request to 'processor' app ", err)
			continue
		}
		defer r.Body.Close()
		if r.StatusCode != http.StatusOK {
			log.Printf("Error sending metrics with %d to 'processor' app got status code %d\n", i, r.StatusCode)
			log.Printf("Status %s \n", r.Status)
			continue
		}
		log.Printf("Metrics with ID %d sent \n", i)
	}
}
