package models

import "time"

// Transaction represents a payment transaction
type Transaction struct {
	Amount float64
	Card   CardInfo
}

// CardInfo represents credit card information
type CardInfo struct {
	Last4    string
	Number   string
	ExpMonth int
	ExpYear  int
	CVV      string
}

// MenuItem represents an item on the menu
type MenuItem struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Price       float64   `json:"price"`
	Category    string    `json:"category,omitempty"`
	Variations  []string  `json:"variations,omitempty"`
	Modifiers   []string  `json:"modifiers,omitempty"`
	UpdatedAt   time.Time `json:"updatedAt,omitempty"`
}

// InventoryItem represents an item in the inventory
type InventoryItem struct {
	ID          string    `json:"id"`
	StoreID     string    `json:"storeId"`
	ProductID   string    `json:"productId"`
	Name        string    `json:"name"`
	Quantity    int       `json:"quantity"`
	LastUpdated time.Time `json:"lastUpdated"`
}

// OrderItem represents an item in an order
type OrderItem struct {
	MenuItemID string
	Name       string
	Quantity   int
	Price      float64
	Modifiers  []string
}

// Order represents a customer order
type Order struct {
	ID        string
	StoreID   string
	Items     []OrderItem
	Total     float64
	CreatedAt time.Time
	Status    string
	Payment   Transaction
}
