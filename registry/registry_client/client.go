// PBL-2/registry/registry_client/client.go
package registry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
	"io"
	"net/url"

	"github.com/4r7hur0/PBL-2/schemas" // Ajuste o caminho do import!
)

type RegistryClient struct {
	RegistryBaseURL string
	HttpClient      *http.Client
}

func NewRegistryClient(registryURL string) *RegistryClient {
	if registryURL == "" {
		log.Println("[RegistryClient] AVISO: URL do Registry não fornecida. Usando padrão http://localhost:9000")
		registryURL = "http://localhost:9000"
	}
	return &RegistryClient{
		RegistryBaseURL: registryURL,
		HttpClient:      &http.Client{Timeout: 5 * time.Second},
	}
}

func (rc *RegistryClient) RegisterService(enterpriseName, cityManaged, apiURL string) error {
	payload := schemas.RegisterRequest{
		EnterpriseName: enterpriseName,
		CityManaged:    cityManaged,
		ApiURL:         apiURL,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("falha ao serializar payload de registro: %w", err)
	}

	reqURL := fmt.Sprintf("%s/register", rc.RegistryBaseURL)
	resp, err := rc.HttpClient.Post(reqURL, "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("falha ao enviar requisição de registro para %s: %w", reqURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Tentar ler o corpo do erro
		var errorBody map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errorBody)
		return fmt.Errorf("falha ao registrar serviço. Status: %s, Corpo: %v", resp.Status, errorBody)
	}

	log.Printf("[RegistryClient] Serviço '%s' para cidade '%s' em '%s' registrado com sucesso.", enterpriseName, cityManaged, apiURL)
	return nil
}

func (rc *RegistryClient) DiscoverService(cityName string) (schemas.DiscoverResponse, error) {
    var discoverResp schemas.DiscoverResponse
		encodedCityName := url.QueryEscape(cityName)
    reqURL := fmt.Sprintf("%s/discover?city=%s", rc.RegistryBaseURL, encodedCityName)

		log.Printf("[RegistryClient] Enviando GET para: %s (Nome original da cidade: '%s')", reqURL, cityName)
    log.Printf("[RegistryClient] Enviando requisição de descoberta para: %s", reqURL) 
    resp, err := rc.HttpClient.Get(reqURL)
    if err != nil {
        return discoverResp, fmt.Errorf("falha ao enviar requisição de descoberta para %s: %w", reqURL, err)
    }
    defer resp.Body.Close()

    // Ler e logar o corpo da resposta ANTES de decodificar
    bodyBytes, readErr := io.ReadAll(resp.Body)
    if readErr != nil {
        log.Printf("[RegistryClient] Erro ao ler o corpo da resposta para %s: %v", cityName, readErr)
        // Recriar o resp.Body para a tentativa de Decode, embora possa não ser útil se a leitura falhou
         resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes)) // Tenta repopular, mas pode já estar comprometido
        return discoverResp, fmt.Errorf("falha ao ler corpo da resposta de descoberta: %w", readErr)
    }
    // Importante: Depois de ler com io.ReadAll, resp.Body está "gasto".
    // Precisamos substituí-lo por um novo reader com o mesmo conteúdo para o json.Decoder.
    resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

    log.Printf("[RegistryClient] Resposta recebida de %s. Status: %s, Corpo Bruto: '%s'", reqURL, resp.Status, string(bodyBytes))

    // Tentar decodificar
    if err := json.NewDecoder(resp.Body).Decode(&discoverResp); err != nil {
        // O erro que você está vendo acontece aqui. O log acima do corpo bruto é crucial.
        log.Printf("[RegistryClient] Erro ao decodificar JSON para %s. Erro: %v", cityName, err)
        // Retornar o erro original de decodificação, a discoverResp estará no seu valor zero (Found: false).
        return discoverResp, fmt.Errorf("falha ao decodificar resposta de descoberta (%s): %w. Corpo: %s", resp.Status, err, string(bodyBytes))
    }

    // Se a decodificação foi bem-sucedida, discoverResp está preenchida.
    // A lógica abaixo para verificar resp.StatusCode e discoverResp.Found ainda é válida.
    if resp.StatusCode != http.StatusOK || !discoverResp.Found {
        log.Printf("[RegistryClient] Serviço para cidade '%s' não encontrado (Status HTTP: %s, Payload Found: %v)", cityName, resp.Status, discoverResp.Found)
        // Retorna a resposta (que pode ter Found: false) e nenhum erro de *comunicação*
        return discoverResp, nil
    }

    log.Printf("[RegistryClient] Serviço para cidade '%s' descoberto: %+v", cityName, discoverResp)
    return discoverResp, nil
}