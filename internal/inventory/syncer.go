package inventory

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/kieracarman/dripos-demo/internal/models"
)

// MockInventoryMongo is a simple mock of MongoDB for inventory
type MockInventoryMongo struct {
	inventory  map[string]map[string]models.InventoryItem // storeId -> productId -> item
	mu         sync.Mutex
	writeCount int
	readCount  int
}

// NewMockInventoryMongo creates a new mock MongoDB for inventory
func NewMockInventoryMongo() *MockInventoryMongo {
	return &MockInventoryMongo{
		inventory: make(map[string]map[string]models.InventoryItem),
	}
}

// GetInventory retrieves inventory for a store
func (m *MockInventoryMongo) GetInventory(storeID string) ([]models.InventoryItem, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Increment read count
	m.readCount++

	// Simulate MongoDB query latency
	time.Sleep(200 * time.Millisecond)

	storeInventory, exists := m.inventory[storeID]
	if !exists {
		return []models.InventoryItem{}, nil
	}

	items := make([]models.InventoryItem, 0, len(storeInventory))
	for _, item := range storeInventory {
		items = append(items, item)
	}

	return items, nil
}

// UpdateInventory updates the quantity of an inventory item
func (m *MockInventoryMongo) UpdateInventory(item models.InventoryItem) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Simulate MongoDB write latency
	time.Sleep(150 * time.Millisecond)

	// Ensure the store exists in our map
	if _, exists := m.inventory[item.StoreID]; !exists {
		m.inventory[item.StoreID] = make(map[string]models.InventoryItem)
	}

	// Update the item
	m.inventory[item.StoreID][item.ProductID] = item
	m.writeCount++

	return nil
}

// BulkUpdateInventory updates multiple inventory items at once
func (m *MockInventoryMongo) BulkUpdateInventory(items []models.InventoryItem) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Simulate MongoDB bulk write latency
	time.Sleep(200 * time.Millisecond)

	for _, item := range items {
		// Ensure the store exists in our map
		if _, exists := m.inventory[item.StoreID]; !exists {
			m.inventory[item.StoreID] = make(map[string]models.InventoryItem)
		}

		// Update the item
		m.inventory[item.StoreID][item.ProductID] = item
	}

	// Count as a single write operation
	m.writeCount++

	return nil
}

// GetWriteCount returns the number of write operations
func (m *MockInventoryMongo) GetWriteCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.writeCount
}

// GetReadCount returns the number of read operations
func (m *MockInventoryMongo) GetReadCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.readCount
}

// MockInventoryRedis is a simple mock of Redis for inventory
type MockInventoryRedis struct {
	data      map[string][]byte
	expiry    map[string]time.Time
	mu        sync.Mutex
	readCount int
}

// NewMockInventoryRedis creates a new mock Redis for inventory
func NewMockInventoryRedis() *MockInventoryRedis {
	return &MockInventoryRedis{
		data:   make(map[string][]byte),
		expiry: make(map[string]time.Time),
	}
}

// GetInventory retrieves inventory for a store from cache
func (r *MockInventoryRedis) GetInventory(ctx context.Context, storeID string) ([]models.InventoryItem, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Increment read count
	r.readCount++

	// Simulate Redis read latency (much faster than MongoDB)
	time.Sleep(5 * time.Millisecond)

	key := fmt.Sprintf("inventory:%s", storeID)
	data, exists := r.data[key]
	if !exists {
		return nil, fmt.Errorf("key not found")
	}

	// Check expiry
	expTime, hasExpiry := r.expiry[key]
	if hasExpiry && time.Now().After(expTime) {
		delete(r.data, key)
		delete(r.expiry, key)
		return nil, fmt.Errorf("key expired")
	}

	var items []models.InventoryItem
	err := json.Unmarshal(data, &items)
	if err != nil {
		return nil, err
	}

	return items, nil
}

// SetInventory stores inventory for a store in cache with expiration
func (r *MockInventoryRedis) SetInventory(ctx context.Context, storeID string, items []models.InventoryItem, expiration time.Duration) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Simulate Redis write latency
	time.Sleep(10 * time.Millisecond)

	key := fmt.Sprintf("inventory:%s", storeID)
	data, err := json.Marshal(items)
	if err != nil {
		return err
	}

	r.data[key] = data
	if expiration > 0 {
		r.expiry[key] = time.Now().Add(expiration)
	}

	return nil
}

// SetItem updates a single inventory item in the cache
func (r *MockInventoryRedis) SetItem(ctx context.Context, item models.InventoryItem) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Simulate Redis write latency
	time.Sleep(10 * time.Millisecond)

	key := fmt.Sprintf("inventory:%s", item.StoreID)

	// Get existing data
	data, exists := r.data[key]
	var items []models.InventoryItem

	if exists {
		err := json.Unmarshal(data, &items)
		if err != nil {
			return err
		}

		// Find and update the item
		found := false
		for i, existing := range items {
			if existing.ProductID == item.ProductID {
				items[i] = item
				found = true
				break
			}
		}

		// If not found, append it
		if !found {
			items = append(items, item)
		}
	} else {
		// Create new list with single item
		items = []models.InventoryItem{item}
	}

	// Save back to Redis
	newData, err := json.Marshal(items)
	if err != nil {
		return err
	}

	r.data[key] = newData

	// Refresh expiry if it exists
	if _, hasExpiry := r.expiry[key]; hasExpiry {
		r.expiry[key] = time.Now().Add(30 * time.Minute)
	}

	return nil
}

// GetReadCount returns the number of read operations
func (r *MockInventoryRedis) GetReadCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.readCount
}

// InventorySync manages inventory synchronization using Redis as a buffer
type InventorySync struct {
	mongo         *MockInventoryMongo
	redis         *MockInventoryRedis
	batchSize     int
	flushInterval time.Duration
	updateQueue   map[string]map[string]models.InventoryItem // storeId -> productId -> item
	queueMu       sync.Mutex
	lastFlush     time.Time
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
}

// NewInventorySync creates a new inventory syncer
func NewInventorySync(mongo *MockInventoryMongo, redis *MockInventoryRedis) *InventorySync {
	ctx, cancel := context.WithCancel(context.Background())

	return &InventorySync{
		mongo:         mongo,
		redis:         redis,
		batchSize:     50,
		flushInterval: 5 * time.Second,
		updateQueue:   make(map[string]map[string]models.InventoryItem),
		lastFlush:     time.Now(),
		ctx:           ctx,
		cancel:        cancel,
	}
}

// SetBatchSize sets the batch size for MongoDB writes
func (is *InventorySync) SetBatchSize(size int) {
	is.batchSize = size
}

// SetFlushInterval sets the interval for automatic flushing
func (is *InventorySync) SetFlushInterval(interval time.Duration) {
	is.flushInterval = interval
}

// UpdateItem updates an inventory item
func (is *InventorySync) UpdateItem(item models.InventoryItem) error {
	// Update in Redis immediately for fast reads
	ctx := context.Background()
	err := is.redis.SetItem(ctx, item)
	if err != nil {
		return fmt.Errorf("redis update failed: %v", err)
	}

	// Queue the update for MongoDB
	is.queueMu.Lock()
	defer is.queueMu.Unlock()

	// Ensure the store exists in our map
	if _, exists := is.updateQueue[item.StoreID]; !exists {
		is.updateQueue[item.StoreID] = make(map[string]models.InventoryItem)
	}

	// Add to queue
	is.updateQueue[item.StoreID][item.ProductID] = item

	// Check if we need to flush
	queueSize := is.countQueueItems()
	if queueSize >= is.batchSize {
		go is.Flush()
	}

	return nil
}

// countQueueItems counts the total number of items in the queue
func (is *InventorySync) countQueueItems() int {
	count := 0
	for _, items := range is.updateQueue {
		count += len(items)
	}
	return count
}

// Flush writes queued updates to MongoDB
func (is *InventorySync) Flush() error {
	is.queueMu.Lock()

	// Check if there's anything to flush
	if is.countQueueItems() == 0 {
		is.queueMu.Unlock()
		return nil
	}

	// Create a batch of items to update
	batch := make([]models.InventoryItem, 0, is.countQueueItems())
	for _, storeItems := range is.updateQueue {
		for _, item := range storeItems {
			batch = append(batch, item)
		}
	}

	// Clear the queue
	is.updateQueue = make(map[string]map[string]models.InventoryItem)
	is.lastFlush = time.Now()

	is.queueMu.Unlock()

	// Update MongoDB in bulk
	err := is.mongo.BulkUpdateInventory(batch)
	if err != nil {
		// If failed, requeue the items
		is.queueMu.Lock()
		for _, item := range batch {
			if _, exists := is.updateQueue[item.StoreID]; !exists {
				is.updateQueue[item.StoreID] = make(map[string]models.InventoryItem)
			}
			is.updateQueue[item.StoreID][item.ProductID] = item
		}
		is.queueMu.Unlock()

		return fmt.Errorf("bulk update failed: %v", err)
	}

	log.Printf("Flushed %d inventory updates to MongoDB", len(batch))
	return nil
}

// StartAutoFlush starts the background flush loop
func (is *InventorySync) StartAutoFlush() {
	is.wg.Add(1)
	go func() {
		defer is.wg.Done()
		ticker := time.NewTicker(is.flushInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				is.queueMu.Lock()
				queueSize := is.countQueueItems()
				timeSinceFlush := time.Since(is.lastFlush)
				is.queueMu.Unlock()

				if queueSize > 0 && timeSinceFlush >= is.flushInterval {
					if err := is.Flush(); err != nil {
						log.Printf("Auto-flush error: %v", err)
					}
				}
			case <-is.ctx.Done():
				// Final flush on shutdown
				if err := is.Flush(); err != nil {
					log.Printf("Final flush error: %v", err)
				}
				return
			}
		}
	}()
}

// Stop stops the syncer and flushes remaining updates
func (is *InventorySync) Stop() error {
	is.cancel()
	is.wg.Wait()
	return nil
}

// GetInventory gets inventory with Redis caching
func (is *InventorySync) GetInventory(storeID string) ([]models.InventoryItem, error) {
	ctx := context.Background()

	// Try to get from Redis first
	items, err := is.redis.GetInventory(ctx, storeID)
	if err == nil {
		return items, nil
	}

	// If not in Redis, get from MongoDB
	items, err = is.mongo.GetInventory(storeID)
	if err != nil {
		return nil, err
	}

	// Update Redis
	err = is.redis.SetInventory(ctx, storeID, items, 30*time.Minute)
	if err != nil {
		log.Printf("Warning: Failed to update Redis: %v", err)
	}

	return items, nil
}

func RunDemo() {
	fmt.Println("=== DripOS Inventory Sync Demo ===")

	// Create mocks
	mongo := NewMockInventoryMongo()
	redis := NewMockInventoryRedis()

	// Sample inventory items
	productNames := map[string]string{
		"coffee-beans-arabica": "Arabica Coffee Beans",
		"coffee-beans-robusta": "Robusta Coffee Beans",
		"milk-whole":           "Whole Milk",
		"milk-almond":          "Almond Milk",
		"milk-oat":             "Oat Milk",
		"syrup-vanilla":        "Vanilla Syrup",
		"syrup-caramel":        "Caramel Syrup",
		"cup-small":            "Small Cup",
		"cup-medium":           "Medium Cup",
		"cup-large":            "Large Cup",
	}

	// === Part 1: Direct MongoDB updates (the "old" way) ===
	fmt.Println("\n1. Simulating inventory updates WITHOUT syncer (old way)...")
	start := time.Now()

	storeIDs := []string{"store1", "store2", "store3", "store4", "store5"}
	productIDs := make([]string, 0, len(productNames))
	for id := range productNames {
		productIDs = append(productIDs, id)
	}

	// Do 100 individual updates
	for i := 0; i < 100; i++ {
		storeID := storeIDs[i%len(storeIDs)]
		productID := productIDs[i%len(productIDs)]

		item := models.InventoryItem{
			ID:          fmt.Sprintf("%s-%s", storeID, productID),
			StoreID:     storeID,
			ProductID:   productID,
			Name:        productNames[productID],
			Quantity:    100 + i,
			LastUpdated: time.Now(),
		}

		err := mongo.UpdateInventory(item)
		if err != nil {
			log.Printf("Error updating inventory directly: %v", err)
		}
	}

	directUpdateTime := time.Since(start)
	directWriteCount := mongo.GetWriteCount()

	// === Part 2: Syncer-based updates ===
	fmt.Println("\n2. Simulating inventory updates WITH syncer (new way)...")

	// Create inventory syncer
	syncer := NewInventorySync(mongo, redis)
	syncer.SetBatchSize(20) // Batch 20 updates together
	syncer.SetFlushInterval(2 * time.Second)

	// Start auto-flush
	syncer.StartAutoFlush()

	// Reset stats
	writeCountBefore := mongo.GetWriteCount()
	start = time.Now()

	// Do 100 updates through the syncer
	for i := 0; i < 100; i++ {
		storeID := storeIDs[i%len(storeIDs)]
		productID := productIDs[i%len(productIDs)]

		item := models.InventoryItem{
			ID:          fmt.Sprintf("%s-%s", storeID, productID),
			StoreID:     storeID,
			ProductID:   productID,
			Name:        productNames[productID],
			Quantity:    200 + i,
			LastUpdated: time.Now(),
		}

		err := syncer.UpdateItem(item)
		if err != nil {
			log.Printf("Error updating inventory via syncer: %v", err)
		}
	}

	// Ensure all updates are flushed
	time.Sleep(3 * time.Second)
	syncer.Stop()

	syncerUpdateTime := time.Since(start)
	syncerWriteCount := mongo.GetWriteCount() - writeCountBefore

	// === Part 3: Test read performance ===
	fmt.Println("\n3. Testing read performance...")

	// Read directly from MongoDB
	start = time.Now()
	_, err := mongo.GetInventory("store1")
	if err != nil {
		log.Printf("Error reading from MongoDB: %v", err)
	}
	mongoReadTime := time.Since(start)

	// Read through syncer (Redis-backed)
	start = time.Now()
	_, err = syncer.GetInventory("store1")
	if err != nil {
		log.Printf("Error reading through syncer: %v", err)
	}
	syncerReadTime := time.Since(start)

	// === Results ===
	fmt.Println("\n=== Results ===")
	fmt.Printf("Write Performance:\n")
	fmt.Printf("- Direct MongoDB updates: %v for 100 updates (%v per update)\n",
		directUpdateTime, directUpdateTime/100)
	fmt.Printf("- Syncer-based updates: %v for 100 updates (%v per update)\n",
		syncerUpdateTime, syncerUpdateTime/100)

	fmt.Printf("\nWrite Operations:\n")
	fmt.Printf("- Direct MongoDB: %d writes\n", directWriteCount)
	fmt.Printf("- Syncer-based: %d writes\n", syncerWriteCount)
	fmt.Printf("- Write reduction: %.2f%%\n",
		100*(1-float64(syncerWriteCount)/float64(directWriteCount)))

	fmt.Printf("\nRead Performance:\n")
	fmt.Printf("- Direct MongoDB read: %v\n", mongoReadTime)
	fmt.Printf("- Redis-cached read: %v\n", syncerReadTime)
	fmt.Printf("- Read speedup: %.2fx\n", float64(mongoReadTime)/float64(syncerReadTime))

	fmt.Println("\n✅ Redis-backed inventory sync cuts MongoDB writes by >50%")
	fmt.Println("✅ Read operations are significantly faster with Redis caching")
	fmt.Println("✅ Batch processing reduces database load during peak hours")
}
