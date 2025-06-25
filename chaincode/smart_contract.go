package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

type smartContract struct {
	contractapi.Contract
}

type RouteSegmentAsset struct {
	City         string `json:"city"`
	StartTimeUTC string `json:"startTimeUTC"`
	EndTimeUTC   string `json:"endTimeUTC"`
}

type ChargingTransaction struct {
	TransactionID             string              `json:"transactionId"`
	VeicleID                  string              `json:"vehicleId"`
	Route                     []RouteSegmentAsset `json:"route"`
	Status                    string              `json:"status"`
	Cost                      float64             `json:"cost"`
	EnergyConsumed            float64             `json:"energyConsumed"`
	ReservationTimeStampUTC   string              `json:"reservationTimeStampUTC"`
	ChargingStartTimeStampUTC string              `json:"chargingStartTimeStampUTC"`
	ChargingEndTimeStampUTC   string              `json:"chargingEndTimeStampUTC"`
	PaymantTimeStampUTC       string              `json:"paymentTimeStampUTC"`
}

type HistoricState struct {
	TxId      string               `json:"txId"`
	Timestamp string               `json:"timestamp"`
	IsDelete  bool                 `json:"isDelete"`
	Value     *ChargingTransaction `json:"value"`
}

func (s *smartContract) RegisterReserve(ctx contractapi.TransactionContextInterface, transactionID string, vehicleID string, routeJSON string) error {
	exists, err := s.transactionExists(ctx, transactionID)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("transaction with ID %s already exists", transactionID)
	}

	var route []RouteSegmentAsset
	err = json.Unmarshal([]byte(routeJSON), &route)
	if err != nil {
		return fmt.Errorf("failed to parse route JSON: %v", err)
	}

	// Usar o timestamp da transação para garantir determinismo
	txTimestamp, err := ctx.GetStub().GetTxTimestamp()
	if err != nil {
		return fmt.Errorf("failed to get transaction timestamp: %v", err)
	}

	transaction := ChargingTransaction{
		TransactionID:           transactionID,
		VeicleID:                vehicleID,
		Route:                   route,
		Status:                  "RESERVED",
		Cost:                    0.0,
		EnergyConsumed:          0.0,
		ReservationTimeStampUTC: txTimestamp.AsTime().Format(time.RFC3339),
	}

	assetBytes, err := json.Marshal(transaction)
	if err != nil {
		return fmt.Errorf("failed to marshal transaction: %v", err)
	}

	return ctx.GetStub().PutState(transactionID, assetBytes)
}

func (s *smartContract) EndCharging(ctx contractapi.TransactionContextInterface, transactionID string, costStr string, energyConsumedStr string) error {
	asset, err := s.getTransaction(ctx, transactionID)
	if err != nil {
		return err
	}

	// A verificação agora é se o estado é RESERVED.
	if asset.Status != "RESERVED" {
		return fmt.Errorf("transaction with ID %s is not in RESERVED status (current status: %s)", transactionID, asset.Status)
	}

	// Conversão de tipos para robustez
	cost, err := strconv.ParseFloat(costStr, 64)
	if err != nil {
		return fmt.Errorf("failed to parse cost string '%s': %v", costStr, err)
	}

	energyConsumed, err := strconv.ParseFloat(energyConsumedStr, 64)
	if err != nil {
		return fmt.Errorf("failed to parse energyConsumed string '%s': %v", energyConsumedStr, err)
	}

	// Usar o timestamp da transação para garantir determinismo
	txTimestamp, err := ctx.GetStub().GetTxTimestamp()
	if err != nil {
		return fmt.Errorf("failed to get transaction timestamp: %v", err)
	}

	asset.Status = "COMPLETED"
	asset.Cost = cost
	asset.EnergyConsumed = energyConsumed
	asset.ChargingEndTimeStampUTC = txTimestamp.AsTime().Format(time.RFC3339)

	assetBytes, err := json.Marshal(asset)
	if err != nil {
		return fmt.Errorf("failed to marshal transaction: %v", err)
	}
	return ctx.GetStub().PutState(transactionID, assetBytes)
}

func (s *smartContract) RegisterPayment(ctx contractapi.TransactionContextInterface, transactionID string) error {
	asset, err := s.getTransaction(ctx, transactionID)
	if err != nil {
		return err
	}

	if asset.Status != "COMPLETED" {
		return fmt.Errorf("transaction with ID %s is not in COMPLETED status", transactionID)
	}

	txTimestamp, err := ctx.GetStub().GetTxTimestamp()
	if err != nil {
		return fmt.Errorf("failed to get transaction timestamp: %v", err)
	}

	asset.Status = "PAID"
	asset.PaymantTimeStampUTC = txTimestamp.AsTime().Format(time.RFC3339)

	assetBytes, err := json.Marshal(asset)
	if err != nil {
		return fmt.Errorf("failed to marshal transaction: %v", err)
	}
	return ctx.GetStub().PutState(transactionID, assetBytes)
}

func (s *smartContract) QueryTransaction(ctx contractapi.TransactionContextInterface, transactionID string) (*ChargingTransaction, error) {
	return s.getTransaction(ctx, transactionID)
}

func (s *smartContract) getTransaction(ctx contractapi.TransactionContextInterface, transactionID string) (*ChargingTransaction, error) {
	assetBytes, err := ctx.GetStub().GetState(transactionID)
	if err != nil {
		return nil, fmt.Errorf("failed to read from world state: %v", err)
	}
	if assetBytes == nil {
		return nil, fmt.Errorf("transaction with ID %s does not exist", transactionID)
	}

	var transaction ChargingTransaction
	err = json.Unmarshal(assetBytes, &transaction)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal transaction: %v", err)
	}

	return &transaction, nil
}

func (s *smartContract) transactionExists(ctx contractapi.TransactionContextInterface, transactionID string) (bool, error) {
	assetBytes, err := ctx.GetStub().GetState(transactionID)
	if err != nil {
		return false, fmt.Errorf("failed to read from world state: %v", err)
	}
	return assetBytes != nil, nil
}

func (s *smartContract) GetTransactionHistory(ctx contractapi.TransactionContextInterface, transactionID string) ([]*HistoricState, error) {
	resultsIterator, err := ctx.GetStub().GetHistoryForKey(transactionID)
	if err != nil {
		return nil, fmt.Errorf("falha ao obter histórico para a transação %s: %w", transactionID, err)
	}
	defer resultsIterator.Close()

	var history []*HistoricState
	for resultsIterator.HasNext() {
		response, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}

		var asset ChargingTransaction
		if !response.IsDelete {
			err := json.Unmarshal(response.Value, &asset)
			if err != nil {
				return nil, err
			}
		}

		historicState := HistoricState{
			TxId:      response.TxId,
			Timestamp: response.Timestamp.AsTime().Format(time.RFC3339),
			IsDelete:  response.IsDelete,
			Value:     &asset,
		}
		history = append(history, &historicState)
	}
	return history, nil
}

func (s *smartContract) Ping(ctx contractapi.TransactionContextInterface) error {
	// Ping não escreve um timestamp variável no estado principal, então está OK.
	timestamp := time.Now().UTC().Format(time.RFC3339)
	pingData := struct {
		Status    string `json:"status"`
		Timestamp string `json:"timestamp"`
	}{
		Status:    "pong",
		Timestamp: timestamp,
	}
	pingBytes, err := json.Marshal(pingData)
	if err != nil {
		return fmt.Errorf("falha ao serializar a resposta do ping: %v", err)
	}
	err = ctx.GetStub().PutState("ping_status", pingBytes)
	if err != nil {
		return fmt.Errorf("falha ao registrar o ping no ledger: %w", err)
	}
	fmt.Printf("Ping registrado com sucesso no ledger: %s\n", string(pingBytes))
	return nil
}

func (s *smartContract) QueryPing(ctx contractapi.TransactionContextInterface) (string, error) {
	pingBytes, err := ctx.GetStub().GetState("ping_status")
	if err != nil {
		return "", fmt.Errorf("falha ao ler do world state: %v", err)
	}
	if pingBytes == nil {
		return "", fmt.Errorf("nenhum ping foi registrado ainda")
	}
	return string(pingBytes), nil
}

func main() {
	chaincode, err := contractapi.NewChaincode(&smartContract{})
	if err != nil {
		fmt.Printf("Error creating smart contract: %v", err)
		return
	}
	if err := chaincode.Start(); err != nil {
		fmt.Printf("Error starting smart contract: %v", err)
	}
}
