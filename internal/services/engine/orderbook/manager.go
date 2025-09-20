package orderbook

import (
	"sort"
	"sync"
	"trading/internal/domain"
)

// OrderBook representa o livro de ofertas de um símbolo
type OrderBook struct {
	Symbol string          `json:"symbol"`
	Bids   []*domain.Order `json:"bids"` // Ordens de compra (preço decrescente)
	Asks   []*domain.Order `json:"asks"` // Ordens de venda (preço crescente)
	mutex  sync.RWMutex    `json:"-"`
}

// Manager gerencia livros de ofertas
type Manager struct {
	orderBooks map[string]*OrderBook // symbol -> OrderBook
	mutex      sync.RWMutex
}

// NewManager cria um novo manager de order book
func NewManager() *Manager {
	return &Manager{
		orderBooks: make(map[string]*OrderBook),
	}
}

// GetOrderBook retorna o livro de ofertas de um símbolo
func (m *Manager) GetOrderBook(symbol string) *OrderBook {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// 1. Buscar livro existente
	if book, exists := m.orderBooks[symbol]; exists {
		// Fazer uma cópia thread-safe para retornar
		book.mutex.RLock()
		defer book.mutex.RUnlock()
		
		bids := make([]*domain.Order, len(book.Bids))
		copy(bids, book.Bids)
		
		asks := make([]*domain.Order, len(book.Asks))
		copy(asks, book.Asks)
		
		return &OrderBook{
			Symbol: symbol,
			Bids:   bids,
			Asks:   asks,
		}
	}

	// 2. Criar novo livro se não existir
	newBook := &OrderBook{
		Symbol: symbol,
		Bids:   []*domain.Order{},
		Asks:   []*domain.Order{},
	}
	
	m.orderBooks[symbol] = newBook
	return newBook
}

// AddOrder adiciona uma ordem ao livro
func (m *Manager) AddOrder(order *domain.Order) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// 1. Obter ou criar livro do símbolo
	book, exists := m.orderBooks[order.Symbol]
	if !exists {
		book = &OrderBook{
			Symbol: order.Symbol,
			Bids:   []*domain.Order{},
			Asks:   []*domain.Order{},
		}
		m.orderBooks[order.Symbol] = book
	}

	book.mutex.Lock()
	defer book.mutex.Unlock()

	// 2. Adicionar ordem na lista correta
	switch order.Side {
	case domain.BUY:
		book.Bids = append(book.Bids, order)
		// 3. Ordenar bids por preço decrescente (price-time priority)
		sort.Slice(book.Bids, func(i, j int) bool {
			if book.Bids[i].Price == book.Bids[j].Price {
				// Se preços iguais, ordenar por tempo (mais antigo primeiro)
				return book.Bids[i].CreatedAt.Before(book.Bids[j].CreatedAt)
			}
			// Preço decrescente para bids
			return book.Bids[i].Price > book.Bids[j].Price
		})
	case domain.SELL:
		book.Asks = append(book.Asks, order)
		// 3. Ordenar asks por preço crescente (price-time priority)
		sort.Slice(book.Asks, func(i, j int) bool {
			if book.Asks[i].Price == book.Asks[j].Price {
				// Se preços iguais, ordenar por tempo (mais antigo primeiro)
				return book.Asks[i].CreatedAt.Before(book.Asks[j].CreatedAt)
			}
			// Preço crescente para asks
			return book.Asks[i].Price < book.Asks[j].Price
		})
	}
}

// RemoveOrder remove uma ordem do livro
func (m *Manager) RemoveOrder(symbol, orderID string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	book, exists := m.orderBooks[symbol]
	if !exists {
		return
	}

	book.mutex.Lock()
	defer book.mutex.Unlock()

	// 1. Buscar e remover da lista de bids
	for i, order := range book.Bids {
		if order.ID == orderID {
			book.Bids = append(book.Bids[:i], book.Bids[i+1:]...)
			return
		}
	}

	// 2. Buscar e remover da lista de asks
	for i, order := range book.Asks {
		if order.ID == orderID {
			book.Asks = append(book.Asks[:i], book.Asks[i+1:]...)
			return
		}
	}
}

// FindBestMatch encontra a melhor correspondência para uma ordem
func (m *Manager) FindBestMatch(order *domain.Order) *domain.Order {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	book, exists := m.orderBooks[order.Symbol]
	if !exists {
		return nil
	}

	book.mutex.RLock()
	defer book.mutex.RUnlock()

	switch order.Side {
	case domain.BUY:
		// Para ordem de compra, buscar a melhor oferta de venda (menor preço)
		if len(book.Asks) == 0 {
			return nil
		}
		
		bestAsk := book.Asks[0] // Asks já estão ordenados por preço crescente
		// Verificar se preços são compatíveis (preço de compra >= preço de venda)
		if order.Price >= bestAsk.Price {
			return bestAsk
		}
		
	case domain.SELL:
		// Para ordem de venda, buscar a melhor oferta de compra (maior preço)
		if len(book.Bids) == 0 {
			return nil
		}
		
		bestBid := book.Bids[0] // Bids já estão ordenados por preço decrescente
		// Verificar se preços são compatíveis (preço de venda <= preço de compra)
		if order.Price <= bestBid.Price {
			return bestBid
		}
	}

	return nil
}
