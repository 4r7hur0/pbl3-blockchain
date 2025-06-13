package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync" // 1. Importe o pacote sync
	"time"

	"github.com/4r7hur0/PBL-2/api/mqtt"
	"github.com/4r7hur0/PBL-2/schemas"
)

type ReservationWindow struct {
	StartTimeUTC  time.Time
	EndTimeUTC    time.Time
	TransactionID string
	Status        string // "prepared", "committed", "charged", "aborted"
}

type ChargingPointWorker struct {
	ID           string
	Reservations []ReservationWindow
	mu           sync.Mutex // 2. Adicione o Mutex à struct
}

func (cpw *ChargingPointWorker) isAvailable(window schemas.ReservationWindow) bool {
	// Esta função não precisa do lock aqui porque ela será chamada
	// de dentro de um trecho de código que já está protegido pelo lock.
	for _, r := range cpw.Reservations {
		if r.Status != "aborted" && r.Status != "charged" &&
			!(window.EndTimeUTC.Before(r.StartTimeUTC) || window.StartTimeUTC.After(r.EndTimeUTC)) {
			return false
		}
	}
	return true
}

func (cpw *ChargingPointWorker) handleMQTTMessage(payload string) {
	var msg map[string]interface{}
	if err := json.Unmarshal([]byte(payload), &msg); err != nil {
		log.Printf("Erro ao decodificar mensagem MQTT: %v", err)
		return
	}

	cmd, _ := msg["command"].(string)
	switch cmd {
	// REMOVIDO: O case "QUERY_AVAILABILITY" não é mais necessário.

	case "PREPARE_RESERVE_WINDOW":
		var window schemas.ReservationWindow
		b, _ := json.Marshal(msg["window"])
		json.Unmarshal(b, &window)
		txID, _ := msg["transaction_id"].(string)
		
		responseTopic, rt_ok := msg["response_topic"].(string)
		if !rt_ok {
			log.Printf("ERRO: Mensagem PREPARE_RESERVE_WINDOW sem response_topic. TX: %s", txID)
			return
		}

		var success bool
		// *** Início da Seção Crítica Atômica ***
		cpw.mu.Lock()
		if cpw.isAvailable(window) {
			cpw.Reservations = append(cpw.Reservations, ReservationWindow{
				StartTimeUTC:  window.StartTimeUTC,
				EndTimeUTC:    window.EndTimeUTC,
				TransactionID: txID,
				Status:        "prepared", // Marca como preparado
			})
			success = true
			log.Printf("[%s] SUCESSO PREPARE para TX: %s. Janela: %v", cpw.ID, txID, window)
		} else {
			success = false
			log.Printf("[%s] FALHA PREPARE para TX: %s. Conflito de janela.", cpw.ID, txID)
		}
		cpw.mu.Unlock()
		// *** Fim da Seção Crítica Atômica ***

		// Monta a resposta
		resp := map[string]interface{}{
			"command":        "PREPARE_RESPONSE",
			"success":        success,
			"transaction_id": txID,
			"worker_id":      cpw.ID,
		}
		respBytes, _ := json.Marshal(resp)

		// Publica a resposta (sucesso ou falha) no tópico de resposta
		mqtt.Publish(responseTopic, string(respBytes))

	case "COMMIT":
		txID, _ := msg["transaction_id"].(string)
		cpw.mu.Lock()
		for i, r := range cpw.Reservations {
			if r.TransactionID == txID && r.Status == "prepared" {
				cpw.Reservations[i].Status = "committed"
				log.Printf("[%s] SUCESSO COMMIT para TX: %s", cpw.ID, txID)
			}
		}
		cpw.mu.Unlock()

	case "ABORT":
		txID, _ := msg["transaction_id"].(string)
		cpw.mu.Lock()
		// Em vez de remover, marcamos como abortada para manter histórico se necessário.
		// Para limpar a lista, você poderia usar a lógica de remoção.
		for i, r := range cpw.Reservations {
			if r.TransactionID == txID && r.Status == "prepared" {
				cpw.Reservations[i].Status = "aborted"
				log.Printf("[%s] SUCESSO ABORT para TX: %s", cpw.ID, txID)
			}
		}
		cpw.mu.Unlock()
	}
}


// Rotina para detectar passagem do tempo e cobrar
func (cpw *ChargingPointWorker) monitorPassageAndCharge() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		<-ticker.C
		now := time.Now().UTC()

		cpw.mu.Lock() // Protege a leitura e modificação das reservas
		for i, r := range cpw.Reservations {
			if r.Status == "committed" && now.After(r.EndTimeUTC) {
				// Gera custo fixo
				cost := 20.0
				cpw.Reservations[i].Status = "charged"
				// Publica evento para API
				event := map[string]interface{}{
					"command":        "VEHICLE_PASSED_AND_CHARGED",
					"transaction_id": r.TransactionID,
					"cost":           cost,
					"window": map[string]interface{}{
						"start_time_utc": r.StartTimeUTC,
						"end_time_utc":   r.EndTimeUTC,
					},
					"worker_id": cpw.ID,
				}
				eventBytes, _ := json.Marshal(event)
				mqtt.Publish(fmt.Sprintf("enterprise/%s/cp/%s/event", os.Getenv("ENTERPRISE_NAME"), cpw.ID), string(eventBytes))
				log.Printf("Reserva %s cobrada e notificada para API.", r.TransactionID)
			}
		}
		cpw.mu.Unlock()
	}
}

func main() {
	workerID := os.Getenv("WORKER_ID")
	if workerID == "" {
		workerID = "CP001"
	}
	// Inicializa o worker com o mutex
	cpw := &ChargingPointWorker{
		ID: workerID,
		mu: sync.Mutex{},
	}
	mqtt.InitializeMQTT("tcp://mosquitto:1883")
	commandTopic := fmt.Sprintf("enterprise/%s/cp/%s/command", os.Getenv("ENTERPRISE_NAME"), workerID)
	msgChan := mqtt.StartListening(commandTopic, 10)
	log.Printf("ChargingPointWorker %s iniciado. Escutando em %s", workerID, commandTopic)

	// Inicia rotina de monitoramento de passagem e cobrança
	go cpw.monitorPassageAndCharge()

	for msg := range msgChan {
		cpw.handleMQTTMessage(msg)
	}
}
