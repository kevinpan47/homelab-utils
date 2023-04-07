package main

import (
	"context"
	"fmt"
	"log"
	"net/smtp"
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

	smtpSender := os.Getenv("SMTP_SENDER")
	smtpReceiver := os.Getenv("SMTP_RECEIVER")
	smtpPassword := os.Getenv("SMTP_PASSWORD")
	smtpServer := os.Getenv("SMTP_SERVER")
	smtpPort, err := strconv.Atoi(os.Getenv("SMTP_PORT"))

	if err != nil {
		smtpPort = 587
	}

	ticker := time.NewTicker(time.Duration(pollingRate) * time.Second)
	ctx := context.Background()

	// Set up authentication using a service account
	client, err := compute.NewService(ctx, option.WithCredentialsFile(serviceAccountKey))
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	notify := false

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
					} else {
						notify = true
					}
				} else {
					// Get the instance's public IP address.
					var publicIPAddress string = ""
					for _, iface := range instance.NetworkInterfaces {
						if iface.AccessConfigs != nil && len(iface.AccessConfigs) > 0 {
							publicIPAddress = iface.AccessConfigs[0].NatIP
							break
						}
					}

					fmt.Printf("Instance %s is running at %s\n", instanceName, publicIPAddress)

					if notify {
						err := sendEmail(
							smtpSender, smtpPassword, smtpReceiver,
							"GCP Proxy server NEW IP", publicIPAddress,
							smtpServer, smtpPort)

						if err != nil {
							log.Fatalf("Failed to send email notification: %v", err)
						} else {
							notify = false
						}
					}
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

func sendEmail(from, password, to, subject, body, smtpServer string, smtpPort int) error {
	// Set up the authentication information.
	auth := smtp.PlainAuth("", from, password, smtpServer)

	// Set up the email message.
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s", from, to, subject, body)

	// Send the email.
	err := smtp.SendMail(fmt.Sprintf("%s:%d", smtpServer, smtpPort), auth, from, []string{to}, []byte(msg))
	if err != nil {
		fmt.Printf("Notification sent to %s", to)
		return err
	}

	return nil
}
