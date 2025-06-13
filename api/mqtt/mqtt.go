package mqtt

import (
	"fmt"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

var client mqtt.Client

func InitializeMQTT(broker string) {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(broker)

	client = mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	fmt.Println("MQTT client connected to broker: ", broker)
}

func Publish(topic string, message string) {
	if client == nil {
		fmt.Println("MQTT client is not initialized")
		return
	}

	token := client.Publish(topic, 0, false, message)
	token.Wait()
	if token.Error() != nil {
		fmt.Printf("Error publishing message: %v\n", token.Error())
	} else {
		fmt.Printf("Published message: %s to topic: %s\n", message, topic)
	}
}

func Subscribe(topic string, handler mqtt.MessageHandler) {
	if client == nil {
		fmt.Println("MQTT client is not initialized")
		return
	}

	token := client.Subscribe(topic, 0, handler)
	token.Wait()
	if token.Error() != nil {
		fmt.Printf("Error subscribing to topic: %v\n", topic)
	} else {
		fmt.Printf("Subscribed to topic: %s\n", topic)
	}
}
