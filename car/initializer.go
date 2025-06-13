package main

import (
	"fmt"
	"math/rand"
	"time"
)

// Generate Car ID in the format "CAR" followed by 4 random letters or numbers
func generateCarID() string {
	rand.Seed(time.Now().UnixNano())
	chars := "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	id := "CAR"
	for i := 0; i < 4; i++ {
		id += string(chars[rand.Intn(len(chars))])
	}
	return id
}

// Initialize Battery level the battery level with a random value
func initializeBatteryLevel() int {
	rand.Seed(time.Now().UnixNano())
	batteryLevel := rand.Intn(51) + 50 // Random value between 50 and 100
	return batteryLevel
}

// Initialize Discharge rate
func initializeDischargeRate() string {
	rand.Seed(time.Now().UnixNano())
	dischargeRate := rand.Intn(21) + 10 // Random value between 10 and 30
	return fmt.Sprintf("%d%%", dischargeRate)
}
