package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/4r7hur0/PBL-2/schemas"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// messageHandler processes incoming messages from the subscribed topic
func messageHandler(client mqtt.Client, msg mqtt.Message) {
	var enterprise schemas.Enterprises

	err := json.Unmarshal(msg.Payload(), &enterprise)
	if err != nil {
		fmt.Printf("Error deserializing message: %v\n", err)
		return
	}
	addEnterprise(enterprise)
}

// addEnterprise adds an enterprise name to the global list and prints the list
func addEnterprise(enterprise schemas.Enterprises) {
	mu.Lock()
	defer mu.Unlock()
	enterprises = append(enterprises, enterprise)
}

func handleJourneyFinished(finishedChan chan struct{}) func(mqtt.Client, mqtt.Message) {
	return func(client mqtt.Client, msg mqtt.Message) {
		var payload map[string]string
		json.Unmarshal(msg.Payload(), &payload)

		log.Println("=======================================================================")
		log.Println("🎉🎉🎉 MENSAGEM DE FIM DE TRAJETO RECEBIDA! 🎉🎉🎉")
		log.Printf("ID da Transação: %s", payload["transaction_id"])
		log.Println("=======================================================================")

		// Envia o sinal para o canal para desbloquear o loop principal
		finishedChan <- struct{}{}
	}
}
