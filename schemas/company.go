package schemas

import (
	"time"
)

// --- ESTRUTURAS DE COMUNICAÇÃO CARRO <-> API (via MQTT) ---

// RouteRequest é a solicitação inicial do carro para uma rota.
type RouteRequest struct {
	VehicleID   string `json:"vehicle_id"`
	Origin      string `json:"origin"`
	Destination string `json:"destination"`
}

// RouteReservationOptions contém as opções de rotas que a API envia ao carro.
type RouteReservationOptions struct {
	RequestID string           `json:"request_id"` // ID único para esta requisição de rota
	VehicleID string           `json:"vehicle_id"`
	Routes    [][]RouteSegment `json:"routes"`
}

// ChosenRouteMsg é a mensagem que o carro envia de volta com a rota escolhida.
type ChosenRouteMsg struct {
	RequestID string         `json:"request_id"`
	VehicleID string         `json:"vehicle_id"`
	Route     []RouteSegment `json:"route"`
}

// ReservationStatus é a mensagem final da API para o carro, confirmando ou negando a reserva.
type ReservationStatus struct {
	TransactionID  string         `json:"transaction_id"`
	VehicleID      string         `json:"vehicle_id"`
	RequestID      string         `json:"request_id"`
	Status         string         `json:"status"` // Ex: "CONFIRMED", "REJECTED"
	Message        string         `json:"message"`
	ConfirmedRoute []RouteSegment `json:"confirmed_route,omitempty"` // Rota confirmada, se aplicável
}

// ReservationEndMessage é enviada quando uma janela de reserva expira.
type ReservationEndMessage struct {
	VehicleID     string    `json:"vehicle_id"`
	TransactionID string    `json:"transaction_id"`
	EndTimeUTC    time.Time `json:"end_time_utc"`
	Message       string    `json:"message"` // Ex: "Reserva encerrada"
}

// --- ESTRUTURAS DE COMUNICAÇÃO INTER-APIs (Two-Phase Commit via HTTP) ---

// RemotePrepareRequest é o payload para a chamada /2pc_remote/prepare.
type RemotePrepareRequest struct {
	TransactionID     string            `json:"transaction_id"`
	VehicleID         string            `json:"vehicle_id"`
	RequestID         string            `json:"request_id"`
	City              string            `json:"city"`
	ReservationWindow ReservationWindow `json:"reservation_window"`
	CoordinatorURL    string            `json:"coordinator_url"` // Campo adicionado
}

// RemotePrepareResponse é a resposta para a chamada /2pc_remote/prepare.
type RemotePrepareResponse struct {
	Status        string `json:"status"` // "PREPARED" ou "REJECTED"
	TransactionID string `json:"transaction_id"`
	Reason        string `json:"reason,omitempty"`
}

// RemoteCommitAbortRequest é o payload para as chamadas /2pc_remote/commit e /2pc_remote/abort.
type RemoteCommitAbortRequest struct {
	TransactionID string `json:"transaction_id"`
}

// CostUpdatePayload é o payload para a chamada /cost-update.
type CostUpdatePayload struct {
	TransactionID string  `json:"transaction_id"`
	SegmentCity   string  `json:"segment_city"`
	Cost          float64 `json:"cost"`
}

// --- ESTRUTURAS DO REGISTRY DE SERVIÇOS ---

// RegisterRequest é o payload para registrar uma API no serviço de Registry.
type RegisterRequest struct {
	CityManaged    string `json:"city_managed"`    // A cidade que esta API gerencia
	ApiURL         string `json:"api_url"`         // A URL base da API (ex: http://solatlantico:8080)
	EnterpriseName string `json:"enterprise_name"` // Nome da empresa/API
}

// DiscoverResponse é a resposta do serviço de Registry para uma consulta.
type DiscoverResponse struct {
	CityName       string `json:"city_name"`
	ApiURL         string `json:"api_url"`
	EnterpriseName string `json:"enterprise_name,omitempty"`
	Found          bool   `json:"found"`
}

// --- ESTRUTURAS E COMPONENTES COMUNS ---

// Enterprises representa uma empresa disponível no sistema.
type Enterprises struct {
	Name string `json:"name"`
	City string `json:"city"`
}

// RouteSegment define um trecho da rota a ser reservado.
type RouteSegment struct {
	City              string            `json:"city"`
	ReservationWindow ReservationWindow `json:"reservation_window"`
}

// ReservationWindow define o início e o fim de uma reserva.
type ReservationWindow struct {
	StartTimeUTC time.Time `json:"start_time_utc"` // Formato: "YYYY-MM-DDTHH:mm:ssZ"
	EndTimeUTC   time.Time `json:"end_time_utc"`   // Formato: "YYYY-MM-DDTHH:mm:ssZ"
}

// ActiveReservation representa o estado de uma reserva no StateManager da API.
type ActiveReservation struct {
	TransactionID     string            `json:"transaction_id"`
	VehicleID         string            `json:"vehicle_id"`
	RequestID         string            `json:"request_id"`
	City              string            `json:"city"`
	ReservationWindow ReservationWindow `json:"reservation_window"`
	Status            string            `json:"status"`    // Ex: "PREPARED", "COMMITTED"
	CoordinatorURL    string            `json:"-"`         // URL do coordenador, não precisa ser exposto no JSON de status.
	WorkerID          string            `json:"worker_id"` // ID do worker que processou a reserva
}

// TransactionState representa o estado de uma transação na Blockchain.
type TransactionState struct {
	Status    string         `json:"status"` // PREPARED, COMMITTED, ABORTED
	Details   []RouteSegment `json:"details"`
	Timestamp time.Time      `json:"timestamp"`
}

// ErrorResponse é uma resposta de erro genérica.
type ErrorResponse struct {
	Status        string `json:"status"`
	TransactionID string `json:"transaction_id,omitempty"`
	Reason        string `json:"reason"`
}

// --- CONSTANTES ---

const (
	// Status de Reserva (2PC e StateManager)
	StatusReservationPrepared  = "PREPARED"
	StatusReservationCommitted = "COMMITTED"
	StatusAborted              = "ABORTED"
	StatusRejected             = "REJECTED"

	// Status para o Carro
	StatusConfirmed = "CONFIRMED"

	// Outros
	ISOFormat = "2006-01-02T15:04:05Z"
)
