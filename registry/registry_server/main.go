package main

import (
	"log"
	"net/http"
	"os"
	"sync"

	// Importe o novo schemas se o criou
	"github.com/4r7hur0/PBL-2/schemas" // Ajuste o caminho do import!
	"github.com/gin-gonic/gin"
)

type ServiceInfo struct {
	CityManaged    string
	ApiURL         string
	EnterpriseName string
	// Poderia adicionar um timestamp de último heartbeat aqui para remoção de inativos
}

var (
	// Protege o acesso concorrente ao mapa de registros
	registry      map[string]ServiceInfo // Chave: Nome da Cidade
	registryMutex = &sync.RWMutex{}
)

func main() {
	registry = make(map[string]ServiceInfo)
	registryPort := os.Getenv("REGISTRY_PORT")
	if registryPort == "" {
		registryPort = "9000" // Porta padrão para o serviço de registro
	}

	log.Printf("Servidor de Registry iniciando na porta %s", registryPort)

	r := gin.Default()

	r.POST("/register", handleRegister)
	r.GET("/discover", handleDiscover) // Ex: /discover?city=Salvador
	r.GET("/services", handleListServices) // Endpoint para listar todos os serviços registrados

	if err := r.Run(":" + registryPort); err != nil {
		log.Fatalf("Falha ao iniciar o servidor de Registry: %v", err)
	}
}

func handleRegister(c *gin.Context) {
	var req schemas.RegisterRequest // Usando o schema
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("[Registry] Erro no payload de registro: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Payload inválido", "details": err.Error()})
		return
	}

	if req.CityManaged == "" || req.ApiURL == "" || req.EnterpriseName == "" {
		log.Printf("[Registry] Falha no registro: campos obrigatórios ausentes. Payload: %+v", req)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Campos 'city_managed', 'api_url', e 'enterprise_name' são obrigatórios."})
		return
	}

	registryMutex.Lock()
	defer registryMutex.Unlock()

	registry[req.CityManaged] = ServiceInfo{
		CityManaged:    req.CityManaged,
		ApiURL:         req.ApiURL,
		EnterpriseName: req.EnterpriseName,
	}

	log.Printf("[Registry] Serviço Registrado: Empresa '%s' para cidade '%s' em %s", req.EnterpriseName, req.CityManaged, req.ApiURL)
	c.JSON(http.StatusOK, gin.H{"message": "Serviço registrado com sucesso", "city": req.CityManaged, "url": req.ApiURL})
}

func handleDiscover(c *gin.Context) {
	cityName := c.Query("city")
	if cityName == "" {
		c.JSON(http.StatusBadRequest, schemas.DiscoverResponse{Found: false, CityName: cityName, ApiURL: ""})
		return
	}

	registryMutex.RLock()
	defer registryMutex.RUnlock()

	service, found := registry[cityName]
	if !found {
		log.Printf("[Registry] Descoberta FALHOU para cidade '%s'", cityName)
		c.JSON(http.StatusNotFound, schemas.DiscoverResponse{Found: false, CityName: cityName, ApiURL: ""})
		return
	}

	log.Printf("[Registry] Descoberta SUCESSO para cidade '%s': %s (%s)", cityName, service.ApiURL, service.EnterpriseName)
	c.JSON(http.StatusOK, schemas.DiscoverResponse{
		Found:          true,
		CityName:       service.CityManaged,
		ApiURL:         service.ApiURL,
		EnterpriseName: service.EnterpriseName,
	})
}

func handleListServices(c *gin.Context) {
	registryMutex.RLock()
	defer registryMutex.RUnlock()

	// Para evitar expor a estrutura interna do mapa diretamente e para ter uma lista
	var servicesList []ServiceInfo
	for _, service := range registry {
		servicesList = append(servicesList, service)
	}
	c.JSON(http.StatusOK, servicesList)
}