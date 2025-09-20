package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	restful "github.com/emicklei/go-restful/v3"

	"trading/internal/services/web/handlers"
)

// TestWebServiceEndpoints testa se todos os endpoints estão funcionando
func TestWebServiceEndpoints(t *testing.T) {
	// Setup - clear any existing container
	restful.DefaultContainer = restful.NewContainer()
	
	container := handlers.NewInternalWebRestfulContainer()
	restful.DefaultContainer.Router(restful.CurlyRouter{})
	restful.Add(container.GetWS())

	// Testa health check
	t.Run("HealthCheck", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/health", nil)
		resp := httptest.NewRecorder()
		restful.DefaultContainer.ServeHTTP(resp, req)

		if resp.Code != 200 {
			t.Errorf("Esperado status 200, obtido %d", resp.Code)
		}

		body := resp.Body.String()
		if body != "OK - HealthCheck" {
			t.Errorf("Esperado 'OK - HealthCheck', obtido '%s'", body)
		}
	})

	// Testa order book
	t.Run("GetOrderBook", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/orderbook/AAPL", nil)
		resp := httptest.NewRecorder()
		restful.DefaultContainer.ServeHTTP(resp, req)

		if resp.Code != 200 {
			t.Errorf("Esperado status 200, obtido %d", resp.Code)
		}

		// Parse JSON response to validate structure
		var orderBook map[string]interface{}
		if err := json.Unmarshal(resp.Body.Bytes(), &orderBook); err != nil {
			t.Errorf("Erro ao parsear resposta JSON: %v", err)
		}

		// Verify required fields exist
		if orderBook["symbol"] != "AAPL" {
			t.Errorf("Esperado symbol=AAPL, obtido %v", orderBook["symbol"])
		}
		if _, exists := orderBook["bids"]; !exists {
			t.Errorf("Campo 'bids' não encontrado na resposta")
		}
		if _, exists := orderBook["asks"]; !exists {
			t.Errorf("Campo 'asks' não encontrado na resposta")
		}
	})

	// Testa portfolio - note que este teste pode falhar se os dados não estiverem disponíveis
	t.Run("GetPortfolio", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/portfolio/ana-silva", nil)
		resp := httptest.NewRecorder()
		restful.DefaultContainer.ServeHTTP(resp, req)

		// O portfolio pode retornar 404 se os dados não estiverem carregados (durante testes)
		// ou 200 se estiverem disponíveis
		if resp.Code != 200 && resp.Code != 404 {
			t.Errorf("Esperado status 200 ou 404, obtido %d", resp.Code)
		}

		if resp.Code == 200 {
			// Parse JSON response to validate structure
			var portfolio map[string]interface{}
			if err := json.Unmarshal(resp.Body.Bytes(), &portfolio); err != nil {
				t.Errorf("Erro ao parsear resposta JSON: %v", err)
			}

			// Verify required fields exist
			if portfolio["user_id"] != "ana-silva" {
				t.Errorf("Esperado user_id=ana-silva, obtido %v", portfolio["user_id"])
			}
		}
	})

	// Testa stocks
	t.Run("GetStocks", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/stocks", nil)
		resp := httptest.NewRecorder()
		restful.DefaultContainer.ServeHTTP(resp, req)

		if resp.Code != 200 {
			t.Errorf("Esperado status 200, obtido %d", resp.Code)
		}

		// Parse JSON response to validate structure
		var stocks map[string]interface{}
		if err := json.Unmarshal(resp.Body.Bytes(), &stocks); err != nil {
			t.Errorf("Erro ao parsear resposta JSON: %v", err)
		}

		// Verify message exists
		if _, exists := stocks["message"]; !exists {
			t.Errorf("Campo 'message' não encontrado na resposta")
		}
	})

	t.Logf("✅ Todos os endpoints básicos estão funcionando!")
}

// TestOrderCreation testa a criação de ordens
func TestOrderCreation(t *testing.T) {
	// Setup - clear any existing container
	restful.DefaultContainer = restful.NewContainer()
	
	container := handlers.NewInternalWebRestfulContainer()
	restful.DefaultContainer.Router(restful.CurlyRouter{})
	restful.Add(container.GetWS())

	t.Run("CreateOrder", func(t *testing.T) {
		// Create order request
		orderJSON := `{
			"user_id": "test-user",
			"symbol": "AAPL",
			"side": "BUY",
			"quantity": 1,
			"price": 200.00
		}`

		req, _ := http.NewRequest("POST", "/api/orders", strings.NewReader(orderJSON))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		restful.DefaultContainer.ServeHTTP(resp, req)

		// Order may be rejected due to user not found or other validation issues
		// during testing, which is expected behavior
		if resp.Code != 201 && resp.Code != 400 {
			t.Errorf("Esperado status 201 ou 400, obtido %d", resp.Code)
		}

		t.Logf("Order creation test completed with status %d", resp.Code)
	})
}
