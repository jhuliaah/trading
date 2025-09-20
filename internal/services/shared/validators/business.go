package validators

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
	"trading/internal/domain"
)

// StockInfo representa informações de uma ação
type StockInfo struct {
	Company     string  `json:"company"`
	Sector      string  `json:"sector"`
	MinPrice    float64 `json:"min_price"`
	MarketCap   string  `json:"market_cap"`
	Description string  `json:"description"`
}

// StockData representa a estrutura do arquivo JSON de ações
type StockData struct {
	Metadata struct {
		Version     string `json:"version"`
		LastUpdated string `json:"last_updated"`
		Market      string `json:"market"`
		Currency    string `json:"currency"`
		TotalSymbols int   `json:"total_symbols"`
		Description string `json:"description"`
	} `json:"metadata"`
	Stocks map[string]StockInfo `json:"stocks"`
}

// BusinessValidator implementa validações de regras de negócio
type BusinessValidator struct {
	stocks map[string]StockInfo // symbol -> StockInfo
}

// NewBusinessValidator cria um novo validador de negócio
func NewBusinessValidator() *BusinessValidator {
	validator := &BusinessValidator{
		stocks: make(map[string]StockInfo),
	}

	// Carregar dados de ações do arquivo JSON
	if err := validator.loadStocksFromJSON(); err != nil {
		// Log error but don't fail - could be handled differently in production
		fmt.Printf("Warning: Could not load stocks from JSON: %v\n", err)
	}

	return validator
}

// loadStocksFromJSON carrega ações do arquivo JSON
func (v *BusinessValidator) loadStocksFromJSON() error {
	data, err := os.ReadFile("data/stocks.json")
	if err != nil {
		return fmt.Errorf("failed to read stocks.json: %w", err)
	}

	var stockData StockData
	if err := json.Unmarshal(data, &stockData); err != nil {
		return fmt.Errorf("failed to parse stocks.json: %w", err)
	}

	for symbol, info := range stockData.Stocks {
		v.stocks[symbol] = info
	}

	return nil
}

// ValidateOrder valida uma ordem completa
func (v *BusinessValidator) ValidateOrder(order *domain.Order) error {
	// 1. Validar campos básicos
	if order == nil {
		return domain.ErrInvalidOrder
	}
	
	if order.UserID == "" {
		return domain.ErrInvalidUser
	}
	
	if order.Symbol == "" {
		return domain.ErrInvalidSymbol
	}
	
	if order.Quantity <= 0 {
		return domain.ErrInvalidQuantity
	}
	
	if order.Price <= 0 {
		return domain.ErrInvalidPrice
	}
	
	// 2. Validar se o símbolo existe
	if err := v.ValidateSymbol(order.Symbol); err != nil {
		return err
	}
	
	// 3. Validar preço mínimo
	if err := v.ValidateMinPrice(order.Symbol, order.Price); err != nil {
		return err
	}
	
	// 4. Validar horário de mercado
	if err := v.ValidateMarketHours(); err != nil {
		return err
	}
	
	return nil
}

// ValidateSymbol valida se o símbolo existe
func (v *BusinessValidator) ValidateSymbol(symbol string) error {
	// Normalizar símbolo (uppercase)
	symbol = strings.ToUpper(symbol)
	
	if _, exists := v.stocks[symbol]; !exists {
		return domain.ErrInvalidSymbol
	}
	
	return nil
}

// ValidateMinPrice valida se o preço está acima do mínimo
func (v *BusinessValidator) ValidateMinPrice(symbol string, price float64) error {
	// Normalizar símbolo (uppercase)
	symbol = strings.ToUpper(symbol)
	
	stockInfo, exists := v.stocks[symbol]
	if !exists {
		return domain.ErrInvalidSymbol
	}
	
	if price < stockInfo.MinPrice {
		return domain.ErrPriceTooLow
	}
	
	return nil
}

// ValidateMarketHours valida se o mercado está aberto
func (v *BusinessValidator) ValidateMarketHours() error {
	now := time.Now()
	
	// Converter para EST (UTC-5 during standard time, UTC-4 during daylight time)
	est := time.FixedZone("EST", -5*60*60)
	estTime := now.In(est)
	
	// Verificar se é dia útil (segunda a sábado para o evento - simplified weekend check)
	weekday := estTime.Weekday()
	if weekday == time.Sunday {
		return domain.ErrMarketClosed
	}
	
	// Verificar horário de funcionamento (9:30-16:00 EST)
	hour := estTime.Hour()
	minute := estTime.Minute()
	
	// Antes de 9:30 AM
	if hour < 9 || (hour == 9 && minute < 30) {
		return domain.ErrMarketClosed
	}
	
	// Depois de 4:00 PM
	if hour >= 16 {
		return domain.ErrMarketClosed
	}
	
	// TODO: Verificar feriados da NYSE (simplified for now)
	
	return nil
}
