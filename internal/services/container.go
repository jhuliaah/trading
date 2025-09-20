package services

import (
	"trading/internal/services/engine/matching"
	"trading/internal/services/engine/orderbook"
	"trading/internal/services/engine/portfolio"
	"trading/internal/services/shared/validators"
)

// Container manages all service dependencies
type Container struct {
	PortfolioService  *portfolio.Service
	OrderBookManager  *orderbook.Manager
	BusinessValidator *validators.BusinessValidator
	MatchingEngine    *matching.Service
}

// NewContainer creates a new service container with shared instances
func NewContainer() *Container {
	// Create shared instances
	portfolioService := portfolio.NewService()
	orderBookManager := orderbook.NewManager()
	businessValidator := validators.NewBusinessValidator()
	
	// Create matching engine with shared dependencies
	matchingEngine := matching.NewServiceWithDependencies(
		portfolioService,
		orderBookManager,
		businessValidator,
	)

	return &Container{
		PortfolioService:  portfolioService,
		OrderBookManager:  orderBookManager,
		BusinessValidator: businessValidator,
		MatchingEngine:    matchingEngine,
	}
}