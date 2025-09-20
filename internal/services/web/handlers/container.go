package handlers

import (
	"log"
	"time"

	restful "github.com/emicklei/go-restful/v3"
)

// InternalWebRestfulContainer gerencia o container RESTful
type InternalWebRestfulContainer struct {
	webService     *restful.WebService
	tradingHandler *TradingHandler
}

// NewInternalWebRestfulContainer cria um novo container RESTful
func NewInternalWebRestfulContainer() *InternalWebRestfulContainer {
	container := &InternalWebRestfulContainer{
		tradingHandler: NewTradingHandler(),
	}

	// Configura web service
	container.setupWebService()

	return container
}

// GetWS retorna o web service configurado
func (c *InternalWebRestfulContainer) GetWS() *restful.WebService {
	return c.webService
}

// setupWebService configura rotas e middleware
func (c *InternalWebRestfulContainer) setupWebService() {
	ws := new(restful.WebService)

	// Configurações básicas
	ws.Path("/api").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON).
		Doc("Trading System API")

	// Middleware de CORS
	ws.Filter(c.corsFilter)

	// Middleware de logging
	ws.Filter(c.loggingFilter)

	// Health check
	ws.Route(ws.GET("/health").To(c.tradingHandler.HealthCheck).
		Doc("Health check endpoint").
		Returns(200, "OK", nil))

	// Rotas de ordens
	ws.Route(ws.POST("/orders").To(c.tradingHandler.CreateOrder).
		Doc("Create a new order").
		Returns(201, "Order created", nil).
		Returns(400, "Bad request", nil))

	// Rotas de order book
	ws.Route(ws.GET("/orderbook/{symbol}").To(c.tradingHandler.GetOrderBook).
		Doc("Get order book for symbol").
		Param(ws.PathParameter("symbol", "Stock symbol").DataType("string")).
		Returns(200, "OK", nil))

	// Rotas de portfolio
	ws.Route(ws.GET("/portfolio/{user_id}").To(c.tradingHandler.GetPortfolio).
		Doc("Get user portfolio").
		Param(ws.PathParameter("user_id", "User ID").DataType("string")).
		Returns(200, "OK", nil).
		Returns(404, "User not found", nil))

	// Rotas de usuário
	ws.Route(ws.GET("/users/{user_id}").To(c.tradingHandler.GetUserProfile).
		Doc("Get user profile").
		Param(ws.PathParameter("user_id", "User ID").DataType("string")).
		Returns(200, "OK", nil))

	// Rotas de mercado
	ws.Route(ws.GET("/market/status").To(c.tradingHandler.GetMarketStatus).
		Doc("Get market status").
		Returns(200, "OK", nil))

	// Rotas de ações
	ws.Route(ws.GET("/stocks").To(c.tradingHandler.GetStocks).
		Doc("Get available stocks").
		Returns(200, "OK", nil))

	// Rotas de trades
	ws.Route(ws.GET("/trades").To(c.tradingHandler.GetTrades).
		Doc("Get trades history").
		Returns(200, "OK", nil))

	// Rotas de estatísticas
	ws.Route(ws.GET("/stats").To(c.tradingHandler.GetStats).
		Doc("Get system statistics").
		Returns(200, "OK", nil))

	c.webService = ws
}

// corsFilter implementa CORS middleware
func (c *InternalWebRestfulContainer) corsFilter(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
	resp.Header().Set("Access-Control-Allow-Origin", "*")
	resp.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	resp.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if req.Request.Method == "OPTIONS" {
		resp.WriteHeader(200)
		return
	}

	chain.ProcessFilter(req, resp)
}

// loggingFilter implementa logging middleware
func (c *InternalWebRestfulContainer) loggingFilter(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
	start := time.Now()
	chain.ProcessFilter(req, resp)
	duration := time.Since(start)

	// Log da requisição
	log.Printf("📡 %s %s - %d - %v",
		req.Request.Method,
		req.Request.URL.Path,
		resp.StatusCode(),
		duration)
}
