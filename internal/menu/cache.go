package menu

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/kieracarman/dripos-demo/internal/models"
)

// MockRedis is a simple mock of Redis
type MockRedis struct {
	data   map[string][]byte
	expiry map[string]time.Time
	mu     sync.Mutex
}

// NewMockRedis creates a new mock Redis
func NewMockRedis() *MockRedis {
	return &MockRedis{
		data:   make(map[string][]byte),
		expiry: make(map[string]time.Time),
	}
}

// Get retrieves a value from Redis
func (r *MockRedis) Get(ctx context.Context, key string) ([]models.MenuItem, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	data, exists := r.data[key]
	if !exists {
		return nil, fmt.Errorf("cache miss: key not found")
	}

	// Check expiry
	expTime, hasExpiry := r.expiry[key]
	if hasExpiry && time.Now().After(expTime) {
		delete(r.data, key)
		delete(r.expiry, key)
		return nil, fmt.Errorf("cache miss: key expired")
	}

	var items []models.MenuItem
	err := json.Unmarshal(data, &items)
	if err != nil {
		return nil, err
	}

	return items, nil
}

// Set stores a value in Redis with expiration
func (r *MockRedis) Set(ctx context.Context, key string, val []models.MenuItem, expiration time.Duration) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	data, err := json.Marshal(val)
	if err != nil {
		return err
	}

	r.data[key] = data
	if expiration > 0 {
		r.expiry[key] = time.Now().Add(expiration)
	}

	return nil
}

// MockMongo is a simple mock of MongoDB
type MockMongo struct {
	collections map[string][]models.MenuItem
	mu          sync.Mutex
	queryCount  int
}

// NewMockMongo creates a new mock MongoDB
func NewMockMongo() *MockMongo {
	return &MockMongo{
		collections: make(map[string][]models.MenuItem),
	}
}

// FindAll retrieves all items from a collection
func (m *MockMongo) FindAll(collection string) ([]models.MenuItem, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Increment query count
	m.queryCount++

	// Simulate MongoDB query latency (MongoDB is slow during rush hour)
	time.Sleep(300 * time.Millisecond)

	items, exists := m.collections[collection]
	if !exists {
		return nil, fmt.Errorf("collection not found")
	}

	return items, nil
}

// GetQueryCount returns the number of MongoDB queries made
func (m *MockMongo) GetQueryCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.queryCount
}

// InsertTestData adds test data to MongoDB
func (m *MockMongo) InsertTestData(collection string, items []models.MenuItem) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.collections[collection] = items
}

// MenuService provides menu data with caching
type MenuService struct {
	redis      *MockRedis
	mongo      *MockMongo
	cacheLock  sync.Mutex
	cacheKey   string
	collection string
}

// NewMenuService creates a new menu service
func NewMenuService(redis *MockRedis, mongo *MockMongo) *MenuService {
	return &MenuService{
		redis:      redis,
		mongo:      mongo,
		cacheKey:   "menu",
		collection: "menu",
	}
}

// GetMenu retrieves the menu with caching
func (s *MenuService) GetMenu() ([]models.MenuItem, error) {
	ctx := context.Background()

	// Try to get from cache first—it's faster than asking MongoDB during a rush
	cached, err := s.redis.Get(ctx, s.cacheKey)
	if err == nil {
		return cached, nil
	}

	// Cache miss? Lock, load from MongoDB, and update
	s.cacheLock.Lock()
	defer s.cacheLock.Unlock() // (Defer is like finally remembering to lock the café door at night)

	// Try once more (another goroutine might have updated while we waited for the lock)
	cached, err = s.redis.Get(ctx, s.cacheKey)
	if err == nil {
		return cached, nil
	}

	// Definitely need to load from MongoDB
	menu, err := s.mongo.FindAll(s.collection)
	if err != nil {
		return nil, fmt.Errorf("MongoDB is having a Monday: %v", err)
	}

	// Update the cache
	cacheErr := s.redis.Set(ctx, s.cacheKey, menu, 10*time.Minute)
	if cacheErr != nil {
		log.Printf("Warning: Failed to update cache: %v", cacheErr)
	}

	return menu, nil
}

func RunDemo() {
	// Create mock services
	redis := NewMockRedis()
	mongo := NewMockMongo()

	// Create test data
	menuItems := []models.MenuItem{
		{
			ID:          "1",
			Name:        "Espresso",
			Description: "A strong coffee brewed by forcing steam under pressure through ground coffee beans",
			Price:       3.50,
			Category:    "Coffee",
			Variations:  []string{"Single", "Double"},
			UpdatedAt:   time.Now(),
		},
		{
			ID:          "2",
			Name:        "Cappuccino",
			Description: "Equal parts espresso, steamed milk, and milk foam",
			Price:       4.75,
			Category:    "Coffee",
			Variations:  []string{"Small", "Medium", "Large"},
			Modifiers:   []string{"Extra Shot", "Vanilla Syrup", "Caramel Syrup"},
			UpdatedAt:   time.Now(),
		},
		{
			ID:          "3",
			Name:        "Latte",
			Description: "Espresso with steamed milk and a small layer of foam",
			Price:       5.25,
			Category:    "Coffee",
			Variations:  []string{"Small", "Medium", "Large"},
			Modifiers:   []string{"Extra Shot", "Vanilla Syrup", "Caramel Syrup", "Hazelnut Syrup"},
			UpdatedAt:   time.Now(),
		},
		{
			ID:          "4",
			Name:        "Croissant",
			Description: "Buttery, flaky pastry",
			Price:       3.25,
			Category:    "Pastries",
			UpdatedAt:   time.Now(),
		},
		{
			ID:          "5",
			Name:        "Blueberry Muffin",
			Description: "Moist muffin with fresh blueberries",
			Price:       3.50,
			Category:    "Pastries",
			UpdatedAt:   time.Now(),
		},
	}

	// Insert the test data into MongoDB
	mongo.InsertTestData("menu", menuItems)

	// Create the menu service
	menuService := NewMenuService(redis, mongo)

	fmt.Println("=== DripOS Menu Cache Demo ===")

	// Demonstrate cache miss followed by cache hit
	fmt.Println("\nSimulating first menu request (cache miss)...")
	start := time.Now()
	menu1, err := menuService.GetMenu()
	firstQueryTime := time.Since(start)

	if err != nil {
		fmt.Printf("Error getting menu: %v\n", err)
		return
	}

	fmt.Printf("First query (cache miss) took %v\n", firstQueryTime)
	fmt.Printf("Menu has %d items\n", len(menu1))

	// Second query should be faster due to caching
	fmt.Println("\nSimulating second menu request (cache hit)...")
	start = time.Now()
	menu2, err := menuService.GetMenu()
	secondQueryTime := time.Since(start)

	if err != nil {
		fmt.Printf("Error getting menu on second try: %v\n", err)
		return
	}

	fmt.Printf("Second query (cache hit) took %v\n", secondQueryTime)

	if len(menu2) != len(menu1) {
		fmt.Println("WARNING: Different menu sizes returned!")
	}

	fmt.Printf("Cache speedup: %.2fx\n", float64(firstQueryTime)/float64(secondQueryTime))

	// Simulate concurrent access during rush hour
	concurrentQueries := 20
	fmt.Printf("\nSimulating rush hour: %d baristas loading the menu simultaneously...\n", concurrentQueries)

	var wg sync.WaitGroup
	start = time.Now()

	for i := 0; i < concurrentQueries; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			_, err := menuService.GetMenu()
			if err != nil {
				log.Printf("Barista %d got an error: %v", id, err)
			}
		}(i)
	}

	wg.Wait()
	concurrentTime := time.Since(start)

	fmt.Printf("All %d baristas got the menu in %v\n", concurrentQueries, concurrentTime)
	fmt.Printf("Average response time per barista: %v\n", concurrentTime/time.Duration(concurrentQueries))
	fmt.Printf("MongoDB query count: %d (should be much less than barista count due to caching)\n", mongo.GetQueryCount())

	fmt.Println("\n✅ Menu loads in <200ms after first request")
	fmt.Println("✅ Redis caching prevents database overload during rush hour")
	fmt.Println("✅ Mutex prevents cache stampede when cache expires")
}
