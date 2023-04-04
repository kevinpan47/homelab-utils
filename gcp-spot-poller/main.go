package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"time"

	"google.golang.org/api/compute/v1"
	"google.golang.org/api/option"
)

func main() {
	// Replace these values with your own
	projectID := os.Getenv("PROJECT_ID")
	zone := os.Getenv("ZONE")
	instanceName := os.Getenv("INSTANCE_NAME")
	serviceAccountKey := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	pollingRate, err := strconv.Atoi(os.Getenv("POLLING_RATE"))

	if err != nil {
		pollingRate = 60
	}

	ticker := time.NewTicker(time.Duration(pollingRate) * time.Second)
	ctx := context.Background()

	// Set up authentication using a service account
	client, err := compute.NewService(ctx, option.WithCredentialsFile(serviceAccountKey))
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	fmt.Printf("Polling %s every %d seconds.\n", instanceName, pollingRate)
	go func() {
		for {
			select {
			case <-ticker.C:
				// Check the status of the instance
				instance, err := client.Instances.Get(projectID, zone, instanceName).Context(ctx).Do()
				if err != nil {
					log.Fatalf("Failed to get instance status: %v", err)
				}
				if instance.Status == "TERMINATED" {
					// Restart the instance
					op, err := client.Instances.Start(projectID, zone, instanceName).Context(ctx).Do()
					if err != nil {
						log.Fatalf("Failed to start instance: %v", err)
					}
					fmt.Printf("Starting instance %s...\n", instanceName)
					// Wait for the operation to complete
					_, err = client.ZoneOperations.Wait(projectID, zone, op.Name).Context(ctx).Do()
					if err != nil {
						log.Fatalf("Failed to wait for operation: %v", err)
					}
				} else {
					fmt.Printf("Instance %s is running.\n", instanceName)
				}
			}
		}
	}()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// Wait for interrupt signal or user input to exit
	select {
	case <-interrupt:
		// Stop the ticker when the interrupt signal is received
		ticker.Stop()
		fmt.Println("Interrupt signal received. Exiting...")
	}
}
