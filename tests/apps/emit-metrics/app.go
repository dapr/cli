package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

type Metrics struct {
	MetricsID int `json:"metricsID"`
}

func main() {
	var baseURL string
	client := http.Client{}
	val, ok := os.LookupEnv("DAPR_HTTP_PORT")
	if !ok {
		fmt.Println("DAPR_HTTP_PORT not set defaulting to 3500")
		baseURL = "http://localhost:3500"
	} else {
		fmt.Println("DAPR_HTTP_PORT set to ", val)
		baseURL = "http://localhost:" + val
	}
	finalURL := baseURL + "/metrics"
	fmt.Println("Sending metrics to ", finalURL)
	for i := 0; i < 2000; i++ {
		time.Sleep(1 * time.Second)
		metrics := Metrics{
			MetricsID: i,
		}
		b, err := json.Marshal(metrics)
		if err != nil {
			fmt.Println("Got error while marshalling metrics ", err)
			continue
		}
		// Send metrics to Dapr
		req, _ := http.NewRequest(http.MethodPost, finalURL, bytes.NewBuffer(b))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("dapr-app-id", "processor")
		r, err := client.Do(req)
		if err != nil {
			fmt.Println("Got error while sending a request to 'processor' app ", err)
			continue
		}
		defer r.Body.Close()
		if r.StatusCode != http.StatusOK {
			fmt.Printf("Error sending metrics with %d to 'processor' app got status code %d\n", i, r.StatusCode)
			fmt.Printf("Status %s \n", r.Status)
			continue
		}
		fmt.Printf("Metrics with ID %d sent \n", i)
	}
}
