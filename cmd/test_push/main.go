package main

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"pocketscribe/internal/services"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Error loading .env file: %v", err)
		log.Println("Continuing with environment variables...")
	}

	// Read APNS configuration from environment
	token := os.Getenv("APNS_TEST_TOKEN")
	bundleID := os.Getenv("APNS_BUNDLE_ID")
	production := os.Getenv("APNS_PRODUCTION") == "true"

	// Validate required variables
	if token == "" {
		log.Fatal("APNS_TEST_TOKEN environment variable is required")
	}
	if bundleID == "" {
		log.Fatal("APNS_BUNDLE_ID environment variable is required")
	}

	// Create APNS service
	apnsService := services.NewAPNSService(token, bundleID, production)

	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║         Push Notification Test Tool                       ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Printf("Configuration:\n")
	fmt.Printf("  Bundle ID:    %s\n", bundleID)
	fmt.Printf("  Environment:  %s\n", map[bool]string{true: "Production", false: "Sandbox"}[production])
	fmt.Printf("  Device Token: %s\n", apnsService.GetDeviceToken())
	fmt.Println()

	// Check if service was created successfully
	if apnsService == nil {
		log.Fatal("❌ Failed to create APNS service. Check your token.")
	}

	// Send test notification
	fmt.Println("Sending test push notification...")
	fmt.Println()

	err := apnsService.SendArticleReadyNotification(
		12345,
		"How to Build Great Products",
	)

	if err != nil {
		fmt.Println()
		fmt.Println("❌ Failed to send notification")
		fmt.Println()
		fmt.Println("Possible issues:")
		fmt.Println("  1. Token is invalid or expired")
		fmt.Println("  2. Device token is invalid or expired")
		fmt.Println("  3. Bundle ID doesn't match the app's bundle identifier")
		fmt.Println()
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println("✅ Successfully sent push notification!")
	fmt.Println()
	fmt.Println("Check your iOS device for the notification.")
}
