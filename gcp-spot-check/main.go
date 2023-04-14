package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net/smtp"
	"os"
	"strconv"
	"strings"

	"google.golang.org/api/compute/v1"
	"google.golang.org/api/option"
)

func main() {
	log.Println("Running GCP Spot check")

	log.SetFlags(log.LstdFlags)
	err := loadEnvFromFile("/app/.env")
	if err != nil {
		log.Fatalln("Error loading environment file:", err)
		return
	}

	// Replace these values with your own
	projectID := os.Getenv("PROJECT_ID")
	zone := os.Getenv("ZONE")
	instanceName := os.Getenv("INSTANCE_NAME")

	smtpSender := os.Getenv("SMTP_SENDER")
	smtpReceiver := os.Getenv("SMTP_RECEIVER")
	smtpPassword := os.Getenv("SMTP_PASSWORD")
	smtpServer := os.Getenv("SMTP_SERVER")
	smtpPort, err := strconv.Atoi(os.Getenv("SMTP_PORT"))

	if err != nil {
		smtpPort = 587
	}

	ctx := context.Background()

	// Set up authentication using a service account
	client, err := compute.NewService(ctx, option.WithCredentialsFile("/app/credentials.json"))
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	notify := false
	publicIPAddress := ""

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
		log.Printf("Starting instance %s...\n", instanceName)
		// Wait for the operation to complete
		_, err = client.ZoneOperations.Wait(projectID, zone, op.Name).Context(ctx).Do()
		if err != nil {
			log.Fatalf("Failed to wait for operation: %v", err)
		} else {
			notify = true
		}
	}

	// Get the instance's public IP address.
	instance, err = client.Instances.Get(projectID, zone, instanceName).Context(ctx).Do()
	if err != nil {
		log.Fatalf("Failed to get instance status: %v", err)
	}

	for _, iface := range instance.NetworkInterfaces {
		if iface.AccessConfigs != nil && len(iface.AccessConfigs) > 0 {
			publicIPAddress = iface.AccessConfigs[0].NatIP
			break
		}
	}

	log.Printf("Instance %s is running at %s\n", instanceName, publicIPAddress)

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

func loadEnvFromFile(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key, value := parts[0], parts[1]
		os.Setenv(key, value)
	}
	return nil
}

func sendEmail(from, password, to, subject, body, smtpServer string, smtpPort int) error {
	// Set up the authentication information.
	auth := smtp.PlainAuth("", from, password, smtpServer)

	// Set up the email message.
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s", from, to, subject, body)

	// Send the email.
	err := smtp.SendMail(fmt.Sprintf("%s:%d", smtpServer, smtpPort), auth, from, []string{to}, []byte(msg))
	if err != nil {
		log.Printf("Notification sent to %s", to)
		return err
	}

	return nil
}
