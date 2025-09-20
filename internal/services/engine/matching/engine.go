package matching

import (
	"trading/internal/domain"
	"trading/internal/services/engine/orderbook"
	"trading/internal/services/engine/portfolio"
	"trading/internal/services/shared/validators"
)

// Service implementa o motor de correspondência
type Service struct {
	portfolioService  *portfolio.Service
	orderBookManager  *orderbook.Manager
	businessValidator *validators.BusinessValidator
}

// MatchResult representa o resultado de uma operação de matching
type MatchResult struct {
	Order    *domain.Order   `json:"order"`
	Trades   []*domain.Trade `json:"trades"`
	Status   string          `json:"status"`
	Message  string          `json:"message"`
	Rejected bool            `json:"rejected,omitempty"`
	Reason   string          `json:"reason,omitempty"`
}

// NewService cria um novo serviço de matching
func NewService() *Service {
	return &Service{
		portfolioService:  portfolio.NewService(),
		orderBookManager:  orderbook.NewManager(),
		businessValidator: validators.NewBusinessValidator(),
	}
}

// NewServiceWithDependencies cria um novo serviço de matching com dependências injetadas
func NewServiceWithDependencies(
	portfolioService *portfolio.Service,
	orderBookManager *orderbook.Manager,
	businessValidator *validators.BusinessValidator,
) *Service {
	return &Service{
		portfolioService:  portfolioService,
		orderBookManager:  orderBookManager,
		businessValidator: businessValidator,
	}
}

// ProcessOrder processa uma ordem através do matching engine
func (s *Service) ProcessOrder(order *domain.Order) *MatchResult {
	// 1. Validar ordem
	if err := s.businessValidator.ValidateOrder(order); err != nil {
		return &MatchResult{
			Order:    order,
			Trades:   []*domain.Trade{},
			Status:   "rejected",
			Message:  err.Error(),
			Rejected: true,
			Reason:   err.Error(),
		}
	}

	// 2. Validar com portfolio service
	if err := s.portfolioService.ValidateOrder(order); err != nil {
		return &MatchResult{
			Order:    order,
			Trades:   []*domain.Trade{},
			Status:   "rejected",
			Message:  err.Error(),
			Rejected: true,
			Reason:   err.Error(),
		}
	}

	var trades []*domain.Trade

	// 3. Buscar correspondências no order book
	for order.RemainingQuantity > 0 {
		matchingOrder := s.orderBookManager.FindBestMatch(order)
		if matchingOrder == nil {
			// Sem correspondências, adicionar ao order book
			break
		}

		// 4. Executar trade
		tradeQuantity := min(order.RemainingQuantity, matchingOrder.RemainingQuantity)
		tradePrice := matchingOrder.Price // Price-time priority: use existing order price

		var trade *domain.Trade
		if order.Side == domain.BUY {
			trade = domain.NewTrade(order, matchingOrder, tradeQuantity, tradePrice)
		} else {
			trade = domain.NewTrade(matchingOrder, order, tradeQuantity, tradePrice)
		}

		// 5. Atualizar portfolios
		if err := s.portfolioService.ExecuteTrade(trade); err != nil {
			return &MatchResult{
				Order:    order,
				Trades:   trades,
				Status:   "rejected",
				Message:  err.Error(),
				Rejected: true,
				Reason:   err.Error(),
			}
		}

		trades = append(trades, trade)

		// 6. Atualizar quantidades restantes
		order.RemainingQuantity -= tradeQuantity
		matchingOrder.RemainingQuantity -= tradeQuantity

		// 7. Remover ordem do order book se totalmente executada
		if matchingOrder.RemainingQuantity == 0 {
			s.orderBookManager.RemoveOrder(matchingOrder.Symbol, matchingOrder.ID)
		}
	}

	// 8. Adicionar ordem restante ao order book se necessário
	if order.RemainingQuantity > 0 {
		s.orderBookManager.AddOrder(order)
		return &MatchResult{
			Order:   order,
			Trades:  trades,
			Status:  "partial",
			Message: "Ordem parcialmente executada e adicionada ao livro",
		}
	}

	// 9. Ordem totalmente executada
	return &MatchResult{
		Order:   order,
		Trades:  trades,
		Status:  "filled",
		Message: "Ordem totalmente executada",
	}
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
