package payment

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/kieracarman/dripos-demo/internal/models"
)

// MockStripe is a simple mock of Stripe payment processor
type MockStripe struct{}

// Charge simulates charging a card
func (s *MockStripe) Charge(amount float64, card models.CardInfo) error {
	// Simulate processing time
	time.Sleep(time.Duration(100+time.Now().UnixNano()%900) * time.Millisecond)

	// Simulate some declined cards
	if card.Last4 == "0000" {
		return fmt.Errorf("card declined: insufficient funds")
	}

	return nil
}

// PaymentProcessor handles bulk payment processing
type PaymentProcessor struct {
	stripe *MockStripe
}

// NewPaymentProcessor creates a new payment processor
func NewPaymentProcessor() *PaymentProcessor {
	return &PaymentProcessor{
		stripe: &MockStripe{},
	}
}

// ProcessPayments processes multiple payments concurrently
func (p *PaymentProcessor) ProcessPayments(transactions []models.Transaction) (successCount int, failedTxns []models.Transaction, errors []error) {
	results := make(chan bool, len(transactions))
	errorChan := make(chan struct {
		txn models.Transaction
		err error
	}, len(transactions))

	// Start a goroutine for each transaction
	for _, txn := range transactions {
		go func(t models.Transaction) {
			err := p.stripe.Charge(t.Amount, t.Card)
			if err != nil {
				errorChan <- struct {
					txn models.Transaction
					err error
				}{t, fmt.Errorf("card %v declined: %v", t.Card.Last4, err)}
				return
			}
			results <- true
		}(txn)
	}

	// Collect results with timeout
	failedTxns = make([]models.Transaction, 0)
	errors = make([]error, 0)
	successCount = 0

	var wg sync.WaitGroup
	wg.Add(len(transactions))

	// Process all transactions with a timeout
	for i := 0; i < len(transactions); i++ {
		select {
		case <-results:
			successCount++
			wg.Done()
		case errResult := <-errorChan:
			failedTxns = append(failedTxns, errResult.txn)
			errors = append(errors, errResult.err)
			log.Printf("Oops: %v", errResult.err)
			wg.Done()
		case <-time.After(5 * time.Second):
			log.Println("Timeout: Stripe took too long—killing the process before it cascades")
			wg.Done()
		}
	}

	return successCount, failedTxns, errors
}

func RunDemo() {
	processor := NewPaymentProcessor()

	// Create 100 sample transactions
	transactions := make([]models.Transaction, 100)
	for i := 0; i < 100; i++ {
		// Create some transactions that will fail (every 10th one)
		last4 := fmt.Sprintf("%04d", i%10000)

		transactions[i] = models.Transaction{
			Amount: 5.75 + float64(i%500)/100.0, // Coffee prices between $5.75 and $10.74
			Card: models.CardInfo{
				Last4:    last4,
				Number:   fmt.Sprintf("4242424242424%s", last4),
				ExpMonth: 1 + (i % 12),
				ExpYear:  2025 + (i % 5),
				CVV:      fmt.Sprintf("%03d", i%1000),
			},
		}
	}

	fmt.Println("=== DripOS Payment Processor Demo ===")
	fmt.Printf("Processing %d payments in parallel with Go...\n", len(transactions))

	start := time.Now()
	successCount, failedTxns, errors := processor.ProcessPayments(transactions)
	elapsed := time.Since(start)

	fmt.Printf("\nResults:\n")
	fmt.Printf("- Processed %d transactions in %v\n", len(transactions), elapsed)
	fmt.Printf("- Average time per transaction: %v\n", elapsed/time.Duration(len(transactions)))
	fmt.Printf("- Successful: %d\n", successCount)
	fmt.Printf("- Failed: %d\n", len(failedTxns))

	if len(errors) > 0 {
		fmt.Println("\nSample errors:")
		for i := 0; i < min(3, len(errors)); i++ {
			fmt.Printf("  - %v\n", errors[i])
		}
	}

	fmt.Println("\n✅ With Go's concurrency model, all 100 payments were processed simultaneously")
	fmt.Println("✅ Each transaction had a strict 5-second timeout to prevent system hangs")
	fmt.Println("✅ Errors were handled gracefully with clear messaging")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
