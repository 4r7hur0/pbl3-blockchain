// PBL-2/api/state/manager.go
package state

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/4r7hur0/PBL-2/api/mqtt"
	"github.com/4r7hur0/PBL-2/schemas"
	"github.com/google/uuid"
)

type TransactionProgress struct {
	TotalSegments     int
	CompletedSegments map[string]schemas.CostUpdatePayload // Mapeia cidade -> dados do custo
	VehicleID         string
	mu                sync.Mutex
}

type CityState struct {
	MaxPosts           int
	ActiveReservations []schemas.ActiveReservation
}

type StateManager struct {
	ownedCity               string
	cityData                *CityState
	enterpriseName          string
	cityDataMux             *sync.Mutex
	myAPIURL                string
	cpWorkerIDs             []string
	CoordinatedTransactions map[string]*TransactionProgress
}

func NewStateManager(ownedCity string, initialPosts int, myAPIURL string, workerIDs []string) *StateManager {
	log.Printf("[StateManager] Inicializando para a cidade: %s com %d postos.", ownedCity, initialPosts)

	// Extrai o nome da empresa da URL da API (ex: http://solatlantico:8080)
	// Esta é uma forma simples; pode ser melhorado se a URL for mais complexa.
	var entName string
	if u, err := url.Parse(myAPIURL); err == nil {
		entName = strings.Split(u.Hostname(), ".")[0]
	}

	return &StateManager{
		ownedCity:      ownedCity,
		enterpriseName: entName, // Adicionado
		myAPIURL:       myAPIURL,
		cpWorkerIDs:    workerIDs, // Adicionado
		cityData: &CityState{
			MaxPosts:           initialPosts,
			ActiveReservations: []schemas.ActiveReservation{},
		},
		cityDataMux:             &sync.Mutex{},
		CoordinatedTransactions: make(map[string]*TransactionProgress),
	}
}

// PrepareReservation verifica e "pré-aloca" um posto na cidade gerenciada.
// VERSÃO ATUALIZADA DA FUNÇÃO PrepareReservation
func (m *StateManager) PrepareReservation(transactionID, vehicleID, requestID string, window schemas.ReservationWindow, coordinatorURL string) (bool, error) {
	m.cityDataMux.Lock()
	defer m.cityDataMux.Unlock()

	// 1. Verificar se já existe uma reserva PREPARADA para esta mesma transação.
	// Isso evita tentar preparar múltiplos workers para a mesma TX na mesma API.
	for _, res := range m.cityData.ActiveReservations {
		if res.TransactionID == transactionID && res.Status == schemas.StatusReservationPrepared {
			log.Printf("[StateManager-%s] TX[%s]: AVISO - Tentativa de preparar uma transação já preparada localmente.", m.ownedCity, transactionID)
			return true, nil // Considera sucesso, pois já está preparado.
		}
	}

	// 2. Tentar preparar um worker disponível. A verificação de capacidade é delegada.
	preparedWorkerID, err := m.attemptToPrepareWorker(transactionID, window)
	if err != nil {
		log.Printf("[StateManager-%s] TX[%s]: FALHA PREPARE - Não foi possível preparar um worker: %v", m.ownedCity, transactionID, err)
		return false, err
	}

	// 3. Sucesso! Adicionar a reserva como PREPARED no estado local do StateManager.
	newRes := schemas.ActiveReservation{
		TransactionID:     transactionID,
		VehicleID:         vehicleID,
		RequestID:         requestID,
		City:              m.ownedCity,
		ReservationWindow: window,
		Status:            schemas.StatusReservationPrepared,
		CoordinatorURL:    coordinatorURL,
		WorkerID:          preparedWorkerID, // Salva o ID do worker que confirmou a preparação.
	}
	m.cityData.ActiveReservations = append(m.cityData.ActiveReservations, newRes)
	log.Printf("[StateManager-%s] TX[%s]: SUCESSO PREPARE. Worker '%s' alocado. Reserva: %+v", m.ownedCity, transactionID, preparedWorkerID, newRes)
	return true, nil
}

// NOVA FUNÇÃO para tentar preparar um worker diretamente.
func (m *StateManager) attemptToPrepareWorker(transactionID string, window schemas.ReservationWindow) (string, error) {
	// Itera sobre todos os workers gerenciados por esta API
	for _, workerID := range m.cpWorkerIDs {
		log.Printf("[StateManager-%s] TX[%s]: Tentando preparar o worker '%s'", m.ownedCity, transactionID, workerID)

		// Cria um tópico de resposta único para esta tentativa específica
		responseTopic := fmt.Sprintf("enterprise/%s/cp/%s/response/%s", m.enterpriseName, workerID, uuid.New().String())

		// Prepara para escutar a resposta
		respChan := mqtt.StartListening(responseTopic, 1)

		// Monta e publica a mensagem de preparação
		prepareMsg := map[string]interface{}{
			"command":        "PREPARE_RESERVE_WINDOW",
			"window":         window,
			"transaction_id": transactionID,
			"response_topic": responseTopic,
		}
		prepareBytes, _ := json.Marshal(prepareMsg)
		commandTopic := fmt.Sprintf("enterprise/%s/cp/%s/command", m.enterpriseName, workerID)
		mqtt.Publish(commandTopic, string(prepareBytes))

		// Aguarda a resposta do worker ou um timeout
		select {
		case respPayload := <-respChan:
			var resp map[string]interface{}
			if err := json.Unmarshal([]byte(respPayload), &resp); err != nil {
				log.Printf("[StateManager-%s] TX[%s]: Erro ao decodificar resposta de PREPARE do worker '%s': %v", m.ownedCity, transactionID, workerID, err)
				mqtt.Unsubscribe(responseTopic) // << CORREÇÃO: Limpa antes de continuar
				continue                        // Tenta o próximo worker
			}

			// Verifica se o worker respondeu com sucesso
			if success, ok := resp["success"].(bool); ok && success {
				log.Printf("[StateManager-%s] TX[%s]: SUCESSO! Worker '%s' preparado.", m.ownedCity, transactionID, workerID)
				mqtt.Unsubscribe(responseTopic) // << CORREÇÃO: Limpa antes de retornar
				return workerID, nil            // Sucesso! Retorna o ID do worker e encerra a função.
			} else {
				log.Printf("[StateManager-%s] TX[%s]: Worker '%s' respondeu com falha (provavelmente ocupado).", m.ownedCity, transactionID, workerID)
				mqtt.Unsubscribe(responseTopic) // << CORREÇÃO: Limpa antes de continuar
				continue                        // Continua para tentar o próximo worker
			}

		case <-time.After(5 * time.Second): // Timeout para a resposta
			log.Printf("[StateManager-%s] TX[%s]: Timeout esperando resposta de PREPARE do worker '%s'", m.ownedCity, transactionID, workerID)
			mqtt.Unsubscribe(responseTopic) // << CORREÇÃO: Limpa antes de continuar
			continue                        // Continua para tentar o próximo worker
		}
	}

	// Se o loop terminar, nenhum worker conseguiu ser preparado.
	return "", fmt.Errorf("nenhum charging point worker disponível ou falha na comunicação na cidade %s", m.ownedCity)
}

func (m *StateManager) CommitReservation(transactionID string) {
	m.cityDataMux.Lock()
	defer m.cityDataMux.Unlock()

	found := false
	for i, res := range m.cityData.ActiveReservations {
		if res.TransactionID == transactionID && res.Status == schemas.StatusReservationPrepared {
			m.cityData.ActiveReservations[i].Status = schemas.StatusReservationCommitted

			// Notifica o worker específico que foi reservado!
			if res.WorkerID != "" {
				m.sendCommandToWorker(res.WorkerID, transactionID, "COMMIT")
			}

			log.Printf("[StateManager-%s] TX[%s]: SUCESSO COMMIT. Reserva: %+v", m.ownedCity, transactionID, m.cityData.ActiveReservations[i])
			found = true
			// Não precisa retornar, pode haver múltiplos segmentos para a mesma TX (embora não neste modelo de cidade única por API)
		}
	}
	if !found {
		log.Printf("[StateManager-%s] TX[%s]: AVISO COMMIT - Nenhuma reserva PREPARED encontrada para este TransactionID.", m.ownedCity, transactionID)
	}
}

func (m *StateManager) AbortReservation(transactionID string) {
	m.cityDataMux.Lock()
	defer m.cityDataMux.Unlock()

	var keptReservations []schemas.ActiveReservation
	aborted := false
	for _, res := range m.cityData.ActiveReservations {
		if res.TransactionID == transactionID && res.Status == schemas.StatusReservationPrepared {

			if res.WorkerID != "" {
				m.sendCommandToWorker(res.WorkerID, transactionID, "ABORT")
			}

			log.Printf("[StateManager-%s] TX[%s]: SUCESSO ABORT. Removendo reserva: %+v", m.ownedCity, transactionID, res)
			aborted = true
		} else {
			keptReservations = append(keptReservations, res)
		}
	}
	m.cityData.ActiveReservations = keptReservations
	if !aborted {
		log.Printf("[StateManager-%s] TX[%s]: AVISO ABORT - Nenhuma reserva PREPARED encontrada para este TransactionID.", m.ownedCity, transactionID)
	}
}

// GetCoordinatorURL encontra e retorna a URL da API coordenadora para uma dada transação.
func (m *StateManager) GetCoordinatorURL(transactionID string) (string, bool) {
	m.cityDataMux.Lock()
	defer m.cityDataMux.Unlock()

	for _, res := range m.cityData.ActiveReservations {
		if res.TransactionID == transactionID {
			// Retorna a URL e um booleano indicando que foi encontrada.
			return res.CoordinatorURL, true
		}
	}

	// Retorna uma string vazia e false se a transação não for encontrada.
	return "", false
}

// IsCoordinator retorna true se esta instância é a coordenadora da transação.
func (m *StateManager) IsCoordinator(transactionID string) bool {
	m.cityDataMux.Lock()
	defer m.cityDataMux.Unlock()

	for _, res := range m.cityData.ActiveReservations {
		if res.TransactionID == transactionID {
			// Se a URL do coordenador for vazia ou "localhost" ou igual à URL desta instância, considere coordenador.
			// Adapte conforme sua lógica de identificação.
			return res.CoordinatorURL == m.myAPIURL // ou compare com sua URL real
		}
	}
	return false
}

// GetCityAvailability - pode ser útil para um endpoint de status
func (m *StateManager) GetCityAvailability() (string, int, []schemas.ActiveReservation) {
	m.cityDataMux.Lock()
	defer m.cityDataMux.Unlock()
	// Retorna uma cópia para evitar race conditions se o chamador modificar o slice
	reservationsCopy := make([]schemas.ActiveReservation, len(m.cityData.ActiveReservations))
	copy(reservationsCopy, m.cityData.ActiveReservations)
	return m.ownedCity, m.cityData.MaxPosts, reservationsCopy
}

func (m *StateManager) sendCommandToWorker(workerID, transactionID, command string) {
	msg := map[string]interface{}{
		"command":        command,
		"transaction_id": transactionID,
	}
	msgBytes, _ := json.Marshal(msg)
	topic := fmt.Sprintf("enterprise/%s/cp/%s/command", m.enterpriseName, workerID)
	mqtt.Publish(topic, string(msgBytes))
	log.Printf("[StateManager-%s] TX[%s]: Comando '%s' enviado para worker '%s'", m.ownedCity, transactionID, command, workerID)
}

// FinalizeReservation atualiza o status de uma reserva para um estado final, como "charged".
func (m *StateManager) FinalizeReservation(transactionID, finalStatus string) {
	m.cityDataMux.Lock()
	defer m.cityDataMux.Unlock()

	found := false
	for i, res := range m.cityData.ActiveReservations {
		if res.TransactionID == transactionID {
			log.Printf("[StateManager-%s] TX[%s]: Reserva encontrada, mudando status de '%s' para '%s'.", m.ownedCity, transactionID, res.Status, finalStatus)
			m.cityData.ActiveReservations[i].Status = finalStatus
			found = true
			break
		}
	}

	if !found {
		log.Printf("[StateManager-%s] TX[%s]: AVISO FinalizeReservation - Nenhuma reserva encontrada para este TransactionID.", m.ownedCity, transactionID)
	}
	// Para o futuro: Você pode querer adicionar uma outra rotina de limpeza que remove
	// reservas no estado "charged" ou "aborted" após algum tempo (ex: 24 horas) para não
	// manter a memória a crescer indefinidamente.
}

func (m *StateManager) StartCoordinatingTransaction(txID, vehicleID string, route []schemas.RouteSegment) {
	m.cityDataMux.Lock()
	defer m.cityDataMux.Unlock()

	if _, exists := m.CoordinatedTransactions[txID]; exists {
		return // Já está a ser rastreada
	}

	m.CoordinatedTransactions[txID] = &TransactionProgress{
		TotalSegments:     len(route),
		CompletedSegments: make(map[string]schemas.CostUpdatePayload),
		VehicleID:         vehicleID,
	}
	log.Printf("[StateManager-%s] TX[%s]: Começando a coordenar transação com %d segmentos.", m.ownedCity, txID, len(route))
}

func (m *StateManager) RecordSegmentCompletion(payload schemas.CostUpdatePayload) (bool, float64) {
	txProgress, exists := m.CoordinatedTransactions[payload.TransactionID]
	if !exists {
		return false, 0 // Não sou o coordenador desta transação
	}

	txProgress.mu.Lock()
	defer txProgress.mu.Unlock()

	// Evita processar o mesmo segmento duas vezes
	if _, done := txProgress.CompletedSegments[payload.SegmentCity]; done {
		return false, 0
	}

	txProgress.CompletedSegments[payload.SegmentCity] = payload
	log.Printf("[StateManager-%s] TX[%s]: Registado segmento completo de '%s'. (%d/%d)", m.enterpriseName, payload.TransactionID, payload.SegmentCity, len(txProgress.CompletedSegments), txProgress.TotalSegments)

	// Verifica se todos os segmentos estão completos
	if len(txProgress.CompletedSegments) == txProgress.TotalSegments {
		var totalCost float64
		for _, p := range txProgress.CompletedSegments {
			totalCost += p.Cost
		}
		log.Printf("[StateManager-%s] TX[%s]: TODOS OS SEGMENTOS COMPLETOS! Custo total: %.2f", m.enterpriseName, payload.TransactionID, totalCost)
		return true, totalCost
	}

	return false, 0
}

func (sm *StateManager) GetVehicleIDForTransaction(txID string) (string, bool) {
	sm.cityDataMux.Lock()
	defer sm.cityDataMux.Unlock()

	details, found := sm.CoordinatedTransactions[txID]
	if !found {
		return "", false
	}
	return details.VehicleID, true
}
