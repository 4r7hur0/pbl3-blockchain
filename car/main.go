package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/4r7hur0/PBL-2/schemas"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func main() {
	// Initialize the MQTT client
	broker := os.Getenv("MQTT_BROKER")
	if broker == "" {
		broker = "tcp://localhost:1883" // Default broker address
	}
	client := initializeMQTTClient(broker)

	CarID := generateCarID()
	fmt.Printf("Car ID: %s\n", CarID)

	// Channel to receive messages from the MQTT broker
	responseChannel := make(chan schemas.RouteReservationOptions)
	finalResponse := make(chan schemas.ReservationStatus)
	journeyFinishedChan := make(chan struct{})

	topic := fmt.Sprintf("car/reservation/status/%s", CarID)

	go func() {
		subscribeToTopic(client, topic, func(c mqtt.Client, m mqtt.Message) {
			var resp schemas.ReservationStatus
			err := json.Unmarshal(m.Payload(), &resp)
			if err != nil {
				fmt.Printf("Error deserializing message: %v\n", err)
				return
			}
			finalResponse <- resp
		})
	}()

	go func() {
		// Subscribe to the topic
		subscribeToTopic(client, "car/enterprises", messageHandler)
	}()

	go func() {
		journeyFinishedTopic := fmt.Sprintf("car/journey/finished/%s", CarID)
		// Passamos o canal para o handler.
		subscribeToTopic(client, journeyFinishedTopic, handleJourneyFinished(journeyFinishedChan))
	}()

	// Go rounine for messages from topic carID
	go func() {
		subscribeToTopic(client, CarID, func(c mqtt.Client, m mqtt.Message) {
			var resp schemas.RouteReservationOptions
			err := json.Unmarshal(m.Payload(), &resp)
			if err != nil {
				fmt.Printf("Error deserializing message: %v\n", err)
				return
			}
			responseChannel <- resp
		})
	}()

	// Initialize battery level and discharge rate
	batteryLevel := initializeBatteryLevel()
	dischargeRate := initializeDischargeRate()
	fmt.Printf("Battery level: %d%%\n", batteryLevel)
	fmt.Printf("Discharge rate: %s\n", dischargeRate)

	var selectedEnterprise *schemas.Enterprises
	for {
		selectedEnterprise = chooseRandomEnterprise()
		if selectedEnterprise != nil {
			fmt.Printf("Selected enterprise: %s\n", selectedEnterprise.Name)
			break
		} else {
			fmt.Println("No enterprise available. Retrying in 5 seconds...")
			time.Sleep(5 * time.Second)
		}
	}

	// Main loop to choose random cities and publish charging requests
	for {
		origin, destination := ChooseTwoRandomCities()
		if origin == "" && destination == "" {
			fmt.Println("No cities available. Retrying in 5 seconds...")
			time.Sleep(5 * time.Second)
			continue
		}

		fmt.Printf("Origin: %s, Destination: %s\n", origin, destination)

		// Publish the charging request
		PublishChargingRequest(client, origin, destination, CarID, selectedEnterprise.Name)
		fmt.Println("Waiting for response...")
		// Wait for a response from the MQTT broker
		// This is a blocking call, so it will wait until a message is received
		response := <-responseChannel
		var validRoutes [][]schemas.RouteSegment
		for _, route := range response.Routes {
			if len(route) > 1 {
				validRoutes = append(validRoutes, route)
			}
		}

		// Verifica se, após o filtro, restou alguma rota válida
		if len(validRoutes) == 0 {
			log.Println("AVISO: Nenhuma rota com mais de um segmento foi oferecida pela API. Tentando novamente...")
			time.Sleep(10 * time.Second) // Espera um pouco antes de tentar de novo
			continue                     // Pula para a próxima iteração do loop, reiniciando o processo
		}
		rand.Seed(time.Now().UnixNano())
		randomIndex := rand.Intn(len(validRoutes))
		selectedRoute := validRoutes[randomIndex]
		fmt.Println("\nChoose route:")
		if len(selectedRoute) == 0 {
			fmt.Println("  No route segments provided.")
		} else {
			for i, segment := range selectedRoute {
				start := segment.ReservationWindow.StartTimeUTC.Format("15:04")
				end := segment.ReservationWindow.EndTimeUTC.Format("15:04")
				date := segment.ReservationWindow.StartTimeUTC.Format("02/01/2006")

				fmt.Printf("  step %d: City: %s, window reserve: %s at %s - %s\n", i+1, segment.City, start, end, date)
			}
		}

		// Publish the route reservation
		chosenRouteMsg := schemas.ChosenRouteMsg{
			RequestID: response.RequestID,
			VehicleID: CarID,
			Route:     selectedRoute,
		}

		payload, err := json.Marshal(chosenRouteMsg)
		if err != nil {
			fmt.Printf("Error serializing message: %v\n", err)
			continue
		}

		token := client.Publish(fmt.Sprintf("car/route/%s", selectedEnterprise.Name), 0, false, payload)
		token.Wait()
		if token.Error() != nil {
			fmt.Printf("Error publishing message: %v\n", token.Error())
			continue
		}
		//fmt.Printf("Route reservation published: %s\n", string(payload))
		fmt.Println("\nReserva de rota publicada:")

		for i, segment := range selectedRoute {
			start := segment.ReservationWindow.StartTimeUTC.Format("15:04")
			end := segment.ReservationWindow.EndTimeUTC.Format("15:04")
			date := segment.ReservationWindow.StartTimeUTC.Format("02/01/2006")
			fmt.Printf("  step %d: %s | window: %s at %s - %s\n", i+1, segment.City, start, end, date)
		}

		fmt.Println("\nWaiting for response...")
		finalMsg := <-finalResponse
		fmt.Printf("Response received: %v\n", finalMsg.Message)

		<-journeyFinishedChan

		// Esta parte do código só será executada após o sinal ser recebido.
		log.Println("✅ Sinal de FIM DE TRAJETO recebido!")
		log.Println("----------------------------------------------------")
		log.Println("Preparando para iniciar um novo ciclo em 10 segundos...")
		time.Sleep(10 * time.Second)
	}

}
