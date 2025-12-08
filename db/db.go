package db

import (
	"encoding/json"
	"os"
	"sync"
)

type Database struct {
	mu       sync.RWMutex
	Data     map[string]interface{}
	FilePath string
}

func New(filePath string) *Database {
	return &Database{
		Data:     make(map[string]interface{}),
		FilePath: filePath,
	}
}

func (db *Database) Load() error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.FilePath == "" {
		return nil
	}

	file, err := os.ReadFile(db.FilePath)
	if err != nil {
		if os.IsNotExist(err) {
			// Create empty db if not exists
			db.Data = make(map[string]interface{})
			return nil
		}
		return err
	}

	return json.Unmarshal(file, &db.Data)
}

func (db *Database) Save() error {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if db.FilePath == "" {
		return nil
	}

	data, err := json.MarshalIndent(db.Data, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(db.FilePath, data, 0644)
}

func (db *Database) Get(key string) (interface{}, bool) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	val, ok := db.Data[key]
	return val, ok
}

func (db *Database) Set(key string, value interface{}) {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.Data[key] = value
}

func (db *Database) GetData() map[string]interface{} {
	db.mu.RLock()
	defer db.mu.RUnlock()
	// Return a copy or the map itself? For simplicity, returning the map.
	// In a real app, we might want to deep copy to avoid race conditions if modified outside.
	return db.Data
}

// Helper to check if a resource is a list (plural) or object (singular)
func IsPlural(data interface{}) bool {
	_, ok := data.([]interface{})
	return ok
}
