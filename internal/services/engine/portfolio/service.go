package portfolio

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"trading/internal/domain"
)

// UserData representa a estrutura do arquivo JSON de usuários
type UserData struct {
	Metadata struct {
		Version     string `json:"version"`
		LastUpdated string `json:"last_updated"`
		TotalUsers  int    `json:"total_users"`
		Description string `json:"description"`
	} `json:"metadata"`
	Users []User `json:"users"`
}

// Service gerencia portfolios dos usuários
type Service struct {
	users      map[string]User       // userID -> User
	portfolios map[string]*domain.Portfolio // userID -> Portfolio
	mutex      sync.RWMutex
}

// User representa dados de usuário
type User struct {
	ID               string         `json:"id"`
	Name             string         `json:"name"`
	Email            string         `json:"email"`
	Profile          string         `json:"profile"`
	Cash             float64        `json:"cash"`
	MaxOrderValue    float64        `json:"max_order_value"`
	Description      string         `json:"description"`
	Status           string         `json:"status"`
	InitialPositions map[string]int `json:"initial_positions,omitempty"`
}

// NewService cria um novo serviço de portfolio
func NewService() *Service {
	service := &Service{
		users:      make(map[string]User),
		portfolios: make(map[string]*domain.Portfolio),
	}

	// Carrega dados de usuários do arquivo JSON
	if err := service.loadUsersFromJSON(); err != nil {
		// Log error but don't fail - could be handled differently in production
		fmt.Printf("Warning: Could not load users from JSON: %v\n", err)
	}

	return service
}

// loadUsersFromJSON carrega usuários do arquivo JSON
func (s *Service) loadUsersFromJSON() error {
	data, err := os.ReadFile("data/users.json")
	if err != nil {
		return fmt.Errorf("failed to read users.json: %w", err)
	}

	var userData UserData
	if err := json.Unmarshal(data, &userData); err != nil {
		return fmt.Errorf("failed to parse users.json: %w", err)
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	for _, user := range userData.Users {
		s.users[user.ID] = user
	}

	return nil
}

// GetPortfolio retorna o portfolio de um usuário
func (s *Service) GetPortfolio(userID string) (*domain.Portfolio, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// 1. Verificar se usuário existe
	user, exists := s.users[userID]
	if !exists {
		return nil, domain.ErrUserNotFound
	}

	// 2. Verificar se já existe portfolio em memória
	if portfolio, exists := s.portfolios[userID]; exists {
		return portfolio, nil
	}

	// 3. Criar novo portfolio com dados do usuário
	portfolio := domain.NewPortfolio(userID, user.Cash)

	// 4. Aplicar posições iniciais se existirem
	if user.InitialPositions != nil {
		for symbol, quantity := range user.InitialPositions {
			portfolio.Positions[symbol] = quantity
		}
	}

	// 5. Salvar portfolio em memória
	s.portfolios[userID] = portfolio

	return portfolio, nil
}

// ValidateOrder valida se o usuário pode fazer a ordem
func (s *Service) ValidateOrder(order *domain.Order) error {
	// 1. Verificar se usuário existe
	user, err := s.GetUser(order.UserID)
	if err != nil {
		return err
	}

	// 2. Obter portfolio do usuário
	portfolio, err := s.GetPortfolio(order.UserID)
	if err != nil {
		return err
	}

	// 3. Validar campos básicos da ordem
	if order.Quantity <= 0 {
		return domain.ErrInvalidQuantity
	}
	if order.Price <= 0 {
		return domain.ErrInvalidPrice
	}
	if order.Side != domain.BUY && order.Side != domain.SELL {
		return domain.ErrInvalidOrderSide
	}

	// 4. Validar valor máximo da ordem baseado no perfil do usuário
	orderValue := order.GetValue()
	if orderValue > user.MaxOrderValue {
		return domain.ErrExceedsLimit
	}

	// 5. Validar saldo/posição baseado no tipo de ordem
	switch order.Side {
	case domain.BUY:
		if !portfolio.HasSufficientCash(orderValue) {
			return domain.ErrInsufficientBalance
		}
	case domain.SELL:
		if !portfolio.HasSufficientPosition(order.Symbol, order.Quantity) {
			return domain.ErrInsufficientPosition
		}
	}

	return nil
}

// ExecuteTrade executa uma negociação atualizando os portfolios
func (s *Service) ExecuteTrade(trade *domain.Trade) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// 1. Obter portfolios do comprador e vendedor
	buyerPortfolio, exists := s.portfolios[trade.BuyerID]
	if !exists {
		return fmt.Errorf("buyer portfolio not found for user %s", trade.BuyerID)
	}

	sellerPortfolio, exists := s.portfolios[trade.SellerID]
	if !exists {
		return fmt.Errorf("seller portfolio not found for user %s", trade.SellerID)
	}

	// 2. Executar compra no portfolio do comprador
	err := buyerPortfolio.ExecuteBuy(trade.Symbol, trade.Quantity, trade.Price)
	if err != nil {
		return fmt.Errorf("failed to execute buy for user %s: %w", trade.BuyerID, err)
	}

	// 3. Executar venda no portfolio do vendedor
	err = sellerPortfolio.ExecuteSell(trade.Symbol, trade.Quantity, trade.Price)
	if err != nil {
		// Reverter a compra se a venda falhar
		buyerPortfolio.ExecuteSell(trade.Symbol, trade.Quantity, trade.Price)
		return fmt.Errorf("failed to execute sell for user %s: %w", trade.SellerID, err)
	}

	return nil
}

// GetUser retorna dados do usuário
func (s *Service) GetUser(userID string) (User, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	user, exists := s.users[userID]
	if !exists {
		return User{}, domain.ErrUserNotFound
	}

	return user, nil
}
