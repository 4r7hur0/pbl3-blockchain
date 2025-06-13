package main

import (
	"encoding/json"
	"fmt"
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

	// Enterprises to publish
	enterprises := []schemas.Enterprises{
		{Name: "SolAtlantico", City: "Salvador"},
		{Name: "SertaoCarga", City: "Feira de Santana"},
		{Name: "CacauPower", City: "Ilheus"},
	}

	topic := "car/enterprises"
	for {
		// Publish enterprises to the topic
		publishEnterprises(client, topic, enterprises)
		time.Sleep(10 * time.Second) // Sleep for 10 seconds before publishing againsss
	}
}

// initializeMQTTClient initializes and connects an MQTT client
func initializeMQTTClient(broker string) mqtt.Client {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(broker)

	for {
		client := mqtt.NewClient(opts)
		if token := client.Connect(); token.Wait() && token.Error() != nil {
			fmt.Printf("%v", token.Error())
			time.Sleep(5 * time.Second)
			continue
		}
		fmt.Println("MQTT client connected to broker:", broker)
		return client
	}

}

// publishEnterprises publishes a list of enterprises to a given topic
func publishEnterprises(client mqtt.Client, topic string, enterprises []schemas.Enterprises) {
	for _, en := range enterprises {
		// Serialize the struct to JSON
		payload, err := json.Marshal(en)
		if err != nil {
			fmt.Printf("Error serializing enterprise: %v\n", err.Error())
			continue
		}

		token := client.Publish(topic, 0, false, payload)
		token.Wait()
		if token.Error() != nil {
			fmt.Printf("Error publishing message: %v\n", token.Error())
		} else {
			fmt.Printf("Published message: %s to topic: %s\n", en, topic)
		}
	}
}
