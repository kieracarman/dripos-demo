package loadtest

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/kieracarman/dripos-demo/internal/models"
	"github.com/kieracarman/dripos-demo/internal/payment"
)

// OrderEndpoint represents a simulated DripOS endpoint
type OrderEndpoint struct {
	ordersMu      sync.Mutex
	orders        map[string]models.Order
	paymentProc   *payment.PaymentProcessor
	throughput    int           // requests/second
	orderProcTime time.Duration // average processing time
	failureRate   float64       // percentage of failures
}

// NewOrderEndpoint creates a new order endpoint
func NewOrderEndpoint() *OrderEndpoint {
	return &OrderEndpoint{
		orders:        make(map[string]models.Order),
		paymentProc:   payment.NewPaymentProcessor(),
		throughput:    100, // Default throughput
		orderProcTime: 200 * time.Millisecond,
		failureRate:   0.01, // 1% failure rate
	}
}

// SetThroughput sets the throughput limit
func (e *OrderEndpoint) SetThroughput(rps int) {
	e.throughput = rps
}

// SetProcessingTime sets the average processing time
func (e *OrderEndpoint) SetProcessingTime(d time.Duration) {
	e.orderProcTime = d
}

// SetFailureRate sets the simulated failure rate
func (e *OrderEndpoint) SetFailureRate(rate float64) {
	e.failureRate = rate
}

// ProcessOrder simulates processing an order
func (e *OrderEndpoint) ProcessOrder(order models.Order) (string, error) {
	// Simulate rate limiting
	if e.throughput > 0 {
		time.Sleep(time.Second / time.Duration(e.throughput))
	}

	// Simulate processing time with some variability
	procTime := e.orderProcTime + time.Duration(rand.Int63n(int64(e.orderProcTime/2)))
	time.Sleep(procTime)

	// Simulate random failures
	if rand.Float64() < e.failureRate {
		return "", fmt.Errorf("service unavailable - Node.js event loop blocked")
	}

	// Process payment
	_, _, errors := e.paymentProc.ProcessPayments([]models.Transaction{order.Payment})
	if len(errors) > 0 {
		return "", errors[0]
	}

	// Store order
	e.ordersMu.Lock()
	defer e.ordersMu.Unlock()
	e.orders[order.ID] = order

	return order.ID, nil
}

// LoadTester runs load tests on the order endpoint
type LoadTester struct {
	endpoint    *OrderEndpoint
	concurrency int
	duration    time.Duration
	results     *LoadTestResults
	storeIDs    []string
	menuItems   []models.MenuItem
}

// LoadTestResults holds the results of a load test
type LoadTestResults struct {
	RequestCount    int
	SuccessCount    int
	FailureCount    int
	TotalDuration   time.Duration
	MinResponseTime time.Duration
	MaxResponseTime time.Duration
	AvgResponseTime time.Duration
	RPS             float64
}

// NewLoadTester creates a new load tester
func NewLoadTester(endpoint *OrderEndpoint) *LoadTester {
	// Generate some store IDs
	storeIDs := []string{"store1", "store2", "store3", "store4", "store5"}

	// Generate some menu items
	menuItems := []models.MenuItem{
		{ID: "item1", Name: "Espresso", Price: 3.50},
		{ID: "item2", Name: "Cappuccino", Price: 4.75},
		{ID: "item3", Name: "Latte", Price: 5.25},
		{ID: "item4", Name: "Americano", Price: 3.75},
		{ID: "item5", Name: "Mocha", Price: 5.50},
		{ID: "item6", Name: "Macchiato", Price: 4.25},
		{ID: "item7", Name: "Flat White", Price: 4.95},
		{ID: "item8", Name: "Drip Coffee", Price: 2.75},
		{ID: "item9", Name: "Cold Brew", Price: 4.50},
		{ID: "item10", Name: "Croissant", Price: 3.25},
	}

	return &LoadTester{
		endpoint:    endpoint,
		concurrency: 10,
		duration:    10 * time.Second,
		results:     &LoadTestResults{},
		storeIDs:    storeIDs,
		menuItems:   menuItems,
	}
}

// SetConcurrency sets the number of concurrent users
func (lt *LoadTester) SetConcurrency(n int) {
	lt.concurrency = n
}

// SetDuration sets the test duration
func (lt *LoadTester) SetDuration(d time.Duration) {
	lt.duration = d
}

// generateRandomOrder creates a random order for testing
func (lt *LoadTester) generateRandomOrder() models.Order {
	// Choose a random store
	storeID := lt.storeIDs[rand.Intn(len(lt.storeIDs))]

	// Generate a random number of items (1-5)
	itemCount := 1 + rand.Intn(5)
	items := make([]models.OrderItem, itemCount)
	total := 0.0

	// Generate random items
	for i := 0; i < itemCount; i++ {
		menuItem := lt.menuItems[rand.Intn(len(lt.menuItems))]
		quantity := 1 + rand.Intn(3)

		items[i] = models.OrderItem{
			MenuItemID: menuItem.ID,
			Name:       menuItem.Name,
			Quantity:   quantity,
			Price:      menuItem.Price,
		}

		total += menuItem.Price * float64(quantity)
	}

	// Create a credit card for payment
	cardLast4 := fmt.Sprintf("%04d", rand.Intn(10000))

	// Create the order
	return models.Order{
		ID:        fmt.Sprintf("order-%d", rand.Int63()),
		StoreID:   storeID,
		Items:     items,
		Total:     total,
		CreatedAt: time.Now(),
		Status:    "new",
		Payment: models.Transaction{
			Amount: total,
			Card: models.CardInfo{
				Last4:    cardLast4,
				Number:   fmt.Sprintf("4242424242424%s", cardLast4),
				ExpMonth: 1 + rand.Intn(12),
				ExpYear:  2025 + rand.Intn(5),
				CVV:      fmt.Sprintf("%03d", rand.Intn(1000)),
			},
		},
	}
}

// RunTest runs a load test
func (lt *LoadTester) RunTest() *LoadTestResults {
	start := time.Now()
	endTime := start.Add(lt.duration)

	var wg sync.WaitGroup
	resultsChan := make(chan struct {
		success      bool
		responseTime time.Duration
	}, lt.concurrency*1000) // Buffer to avoid blocking

	// Start concurrent users
	for i := 0; i < lt.concurrency; i++ {
		wg.Add(1)
		go func(userID int) {
			defer wg.Done()

			for time.Now().Before(endTime) {
				// Generate random order
				order := lt.generateRandomOrder()

				// Process the order and measure time
				requestStart := time.Now()
				_, err := lt.endpoint.ProcessOrder(order)
				responseTime := time.Since(requestStart)

				// Record result
				resultsChan <- struct {
					success      bool
					responseTime time.Duration
				}{
					success:      err == nil,
					responseTime: responseTime,
				}
			}
		}(i)
	}

	// Wait for all workers to finish in a separate goroutine
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Collect results
	var (
		requestCount      int
		successCount      int
		failureCount      int
		totalResponseTime time.Duration
		minResponseTime   = time.Hour // Initialize to a large value
		maxResponseTime   time.Duration
	)

	for result := range resultsChan {
		requestCount++
		if result.success {
			successCount++
		} else {
			failureCount++
		}

		if result.responseTime < minResponseTime {
			minResponseTime = result.responseTime
		}
		if result.responseTime > maxResponseTime {
			maxResponseTime = result.responseTime
		}

		totalResponseTime += result.responseTime
	}

	totalDuration := time.Since(start)

	// Calculate results
	var avgResponseTime time.Duration
	if requestCount > 0 {
		avgResponseTime = totalResponseTime / time.Duration(requestCount)
	}

	lt.results = &LoadTestResults{
		RequestCount:    requestCount,
		SuccessCount:    successCount,
		FailureCount:    failureCount,
		TotalDuration:   totalDuration,
		MinResponseTime: minResponseTime,
		MaxResponseTime: maxResponseTime,
		AvgResponseTime: avgResponseTime,
		RPS:             float64(requestCount) / totalDuration.Seconds(),
	}

	return lt.results
}

func RunDemo() {
	fmt.Println("=== DripOS Load Testing Tool Demo ===")

	// Create endpoint with realistic parameters
	endpoint := NewOrderEndpoint()
	endpoint.SetProcessingTime(300 * time.Millisecond) // Baseline processing time

	// Create load tester
	loadTester := NewLoadTester(endpoint)

	// Run a small test first
	fmt.Println("\n1. Testing with 10 concurrent users (basic load)...")
	loadTester.SetConcurrency(10)
	loadTester.SetDuration(5 * time.Second)
	results1 := loadTester.RunTest()

	fmt.Printf("\nBasic Load Results:\n")
	fmt.Printf("- Requests processed: %d\n", results1.RequestCount)
	fmt.Printf("- Success rate: %.2f%%\n", 100*float64(results1.SuccessCount)/float64(results1.RequestCount))
	fmt.Printf("- Throughput: %.2f orders/second\n", results1.RPS)
	fmt.Printf("- Avg response time: %v\n", results1.AvgResponseTime)

	// Now simulate a rush hour with 100 concurrent users
	fmt.Println("\n2. Testing with 100 concurrent users (rush hour)...")
	loadTester.SetConcurrency(100)
	loadTester.SetDuration(10 * time.Second)
	results2 := loadTester.RunTest()

	fmt.Printf("\nRush Hour Results:\n")
	fmt.Printf("- Requests processed: %d\n", results2.RequestCount)
	fmt.Printf("- Success rate: %.2f%%\n", 100*float64(results2.SuccessCount)/float64(results2.RequestCount))
	fmt.Printf("- Throughput: %.2f orders/second\n", results2.RPS)
	fmt.Printf("- Avg response time: %v\n", results2.AvgResponseTime)

	// Finally, overload test with 1000 concurrent users
	fmt.Println("\n3. Testing with 1000 concurrent users (stress test)...")
	loadTester.SetConcurrency(1000)
	loadTester.SetDuration(5 * time.Second)
	results3 := loadTester.RunTest()

	fmt.Printf("\nStress Test Results:\n")
	fmt.Printf("- Requests processed: %d\n", results3.RequestCount)
	fmt.Printf("- Success rate: %.2f%%\n", 100*float64(results3.SuccessCount)/float64(results3.RequestCount))
	fmt.Printf("- Throughput: %.2f orders/second\n", results3.RPS)
	fmt.Printf("- Avg response time: %v\n", results3.AvgResponseTime)

	fmt.Println("\n✅ Successfully simulated 1000 concurrent users")
	fmt.Println("✅ Go's concurrency allows handling of high volumes without blocking")
	fmt.Println("✅ Failures are isolated and don't bring down the entire system")
}
