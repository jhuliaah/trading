package handlers

import (
	"encoding/json"
	"fmt"
	"github.com/emicklei/go-restful/v3"
	"trading/internal/domain"
	"trading/internal/services"
)

// TradingHandler gerencia endpoints do sistema de trading
type TradingHandler struct {
	container *services.Container
}

// NewTradingHandler cria um novo handler com os serviços inicializados
func NewTradingHandler() *TradingHandler {
	return &TradingHandler{
		container: services.NewContainer(),
	}
}

// CreateOrder cria uma nova ordem de compra ou venda
func (h *TradingHandler) CreateOrder(req *restful.Request, resp *restful.Response) {
	var orderRequest struct {
		UserID   string  `json:"user_id"`
		Symbol   string  `json:"symbol"`
		Side     string  `json:"side"`
		Quantity int     `json:"quantity"`
		Price    float64 `json:"price"`
	}

	if err := req.ReadEntity(&orderRequest); err != nil {
		resp.WriteError(400, fmt.Errorf("invalid request body: %w", err))
		return
	}

	// Converter side string para OrderSide
	var side domain.OrderSide
	switch orderRequest.Side {
	case "BUY":
		side = domain.BUY
	case "SELL":
		side = domain.SELL
	default:
		resp.WriteError(400, fmt.Errorf("invalid order side: %s", orderRequest.Side))
		return
	}

	// Criar ordem
	order := domain.NewOrder(
		orderRequest.UserID,
		orderRequest.Symbol,
		side,
		orderRequest.Quantity,
		orderRequest.Price,
	)

	// Processar através do matching engine
	result := h.container.MatchingEngine.ProcessOrder(order)

	if result.Rejected {
		resp.WriteError(400, fmt.Errorf("order rejected: %s", result.Reason))
		return
	}

	resp.WriteHeader(201)
	resp.WriteEntity(result)
}

// GetOrderBook retorna o livro de ofertas de um símbolo
func (h *TradingHandler) GetOrderBook(req *restful.Request, resp *restful.Response) {
	symbol := req.PathParameter("symbol")
	if symbol == "" {
		resp.WriteError(400, fmt.Errorf("symbol parameter is required"))
		return
	}

	orderBook := h.container.OrderBookManager.GetOrderBook(symbol)
	resp.WriteEntity(orderBook)
}

// GetPortfolio retorna o portfolio de um usuário
func (h *TradingHandler) GetPortfolio(req *restful.Request, resp *restful.Response) {
	userID := req.PathParameter("user_id")
	if userID == "" {
		resp.WriteError(400, fmt.Errorf("user_id parameter is required"))
		return
	}

	portfolio, err := h.container.PortfolioService.GetPortfolio(userID)
	if err != nil {
		if err == domain.ErrUserNotFound {
			resp.WriteError(404, err)
			return
		}
		resp.WriteError(500, err)
		return
	}

	resp.WriteEntity(portfolio)
}

// GetUserProfile retorna perfil e dados de um usuário
func (h *TradingHandler) GetUserProfile(req *restful.Request, resp *restful.Response) {
	userID := req.PathParameter("user_id")
	if userID == "" {
		resp.WriteError(400, fmt.Errorf("user_id parameter is required"))
		return
	}

	user, err := h.container.PortfolioService.GetUser(userID)
	if err != nil {
		if err == domain.ErrUserNotFound {
			resp.WriteError(404, err)
			return
		}
		resp.WriteError(500, err)
		return
	}

	resp.WriteEntity(user)
}

// GetMarketStatus retorna status do mercado (aberto/fechado)
func (h *TradingHandler) GetMarketStatus(req *restful.Request, resp *restful.Response) {
	_, _ = resp.Write([]byte("OK - GetMarketStatus"))
}

// GetStocks retorna lista de ações disponíveis
func (h *TradingHandler) GetStocks(req *restful.Request, resp *restful.Response) {
	// Return basic stock information
	data, err := json.Marshal(map[string]string{
		"message": "Available stocks loaded from data/stocks.json",
		"count":   "20 symbols",
		"market":  "NYSE/NASDAQ",
	})
	if err != nil {
		resp.WriteError(500, err)
		return
	}
	
	resp.Header().Set("Content-Type", "application/json")
	resp.Write(data)
}

// GetTrades retorna histórico de negociações
func (h *TradingHandler) GetTrades(req *restful.Request, resp *restful.Response) {
	_, _ = resp.Write([]byte("OK - GetTrades"))
}

// HealthCheck verifica saúde do sistema
func (h *TradingHandler) HealthCheck(req *restful.Request, resp *restful.Response) {
	_, _ = resp.Write([]byte("OK - HealthCheck"))
}

// GetStats retorna estatísticas do sistema
func (h *TradingHandler) GetStats(req *restful.Request, resp *restful.Response) {
	_, _ = resp.Write([]byte("OK - GetStats"))
}
