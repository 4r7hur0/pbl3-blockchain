package mqtt

import (
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func StartListening(topic string, bufferSize int) chan string {
	// Create a buffered channel to hold incoming messages
	messageChannel := make(chan string, bufferSize)

	// Subscribe to the specified topic
	go func() {
		Subscribe(topic, func(client mqtt.Client, msg mqtt.Message) {
			message := string(msg.Payload())
			//fmt.Printf("Received message: %s from topic: %s\n", message, msg.Topic())
			messageChannel <- message
		})
	}()

	return messageChannel
}
