package cache

import (
	"context"
	"maps"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Define a test object struct
type TestObject struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Age  int    `json:"age"`
}

// TestRedisCache is a struct to hold the redis cache instance for testing
type TestRedisCache struct {
	Cache
}

// NewTestCache creates a new RedisCache instance for testing
func NewTestCache(t *testing.T, cfg RedisConfig) *TestRedisCache {
	client, err := NewRedisCache(cfg)

	if err != nil {
		t.Fatalf("NewRedisCache() failed: %v", err)
	}

	return &TestRedisCache{
		Cache: client,
	}
}

// TestSetGetDel_String tests the Set, Get, and Del methods with string values
func (c *TestRedisCache) TestSetGetDel_String(t *testing.T) {
	ctx := context.Background()

	key := "test_key_string"
	value := "test_value"

	// Set value
	err := c.Set(ctx, key, value, 10*time.Second)
	assert.NoError(t, err, "error while setting value")

	// Get Value
	var wanted string
	err = c.Get(ctx, key, &wanted)
	assert.NoError(t, err, "error while getting value")
	assert.Equal(t, value, wanted, "value not equal")

	// Wait for value to expire
	time.Sleep(10 * time.Second)

	// Verify value is expired
	err = c.Get(ctx, key, &wanted)
	assert.Error(t, err, "value should have been expired from cache")

	err = c.Set(ctx, key, value, 10*time.Second)
	assert.NoError(t, err, "error while setting value")

	// Delete value
	err = c.Del(ctx, key)
	assert.NoError(t, err, "error while deleting value")

	// Verify value is deleted
	err = c.Get(ctx, key, &wanted)
	assert.Error(t, err, "value should have been deleted from cache")

	items := map[string]string{
		"test_key1": "test_value1",
		"test_key2": "test_value2",
		"test_key3": "test_value3",
		"test_key4": "test_value4",
	}

	for k, v := range items {
		err = c.Set(ctx, k, v, 10*time.Second)
		assert.NoError(t, err, "error while setting value")
	}

	// Delete keys once
	err = c.Del(ctx, slices.Collect(maps.Keys(items))...)
	assert.NoError(t, err, "error while deleting keys")

	for k, v := range items {
		err = c.Get(ctx, k, &wanted)
		assert.Error(t, err, "value should have been deleted from cache")
		assert.NotEqual(t, v, wanted, "value not equal")
	}
}

// TestSetGetDel_Struct tests the Set, Get, and Del methods with struct values
func (c *TestRedisCache) TestSetGetDel_Struct(t *testing.T) {
	ctx := context.Background()

	key := "test_key_struct"
	value := TestObject{
		ID:   "123",
		Name: "John Doe",
		Age:  30,
	}

	// Set struct value
	err := c.Set(ctx, key, value, 10*time.Second)
	assert.NoError(t, err, "error while setting struct value")

	// Get struct value
	var wanted TestObject
	err = c.Get(ctx, key, &wanted)
	assert.NoError(t, err, "error while getting struct value")
	assert.Equal(t, value.ID, wanted.ID, "ID not equal")
	assert.Equal(t, value.Name, wanted.Name, "Name not equal")
	assert.Equal(t, value.Age, wanted.Age, "Age not equal")

	// Wait for value to expire
	time.Sleep(10 * time.Second)

	// Verify struct is expired
	err = c.Get(ctx, key, &wanted)
	assert.Error(t, err, "struct should have been expired from cache")

	// Set struct value again
	err = c.Set(ctx, key, value, 10*time.Second)
	assert.NoError(t, err, "error while setting struct value")

	// Delete struct value
	err = c.Del(ctx, key)
	assert.NoError(t, err, "error while deleting struct value")

	// Verify struct is deleted
	err = c.Get(ctx, key, &wanted)
	assert.Error(t, err, "struct should have been deleted from cache")

	// Test with multiple structs
	items := map[string]TestObject{
		"user:1": {ID: "1", Name: "Alice", Age: 25},
		"user:2": {ID: "2", Name: "Bob", Age: 35},
		"user:3": {ID: "3", Name: "Charlie", Age: 40},
	}

	for k, v := range items {
		err = c.Set(ctx, k, v, 10*time.Second)
		assert.NoError(t, err, "error while setting struct value")
	}

	// Verify all structs are stored correctly
	for k, expected := range items {
		var actual TestObject
		err = c.Get(ctx, k, &actual)
		assert.NoError(t, err, "error while getting struct value")
		assert.Equal(t, expected, actual, "struct values not equal")
	}

	// Delete all structs
	err = c.Del(ctx, slices.Collect(maps.Keys(items))...)
	assert.NoError(t, err, "error while deleting structs")

	// Verify all structs are deleted
	for k := range items {
		var actual TestObject
		err = c.Get(ctx, k, &actual)
		assert.Error(t, err, "struct should have been deleted from cache")
	}
}

// TestSetGetDel_Bytes tests the Set, Get, and Del methods with byte slice values
func (c *TestRedisCache) TestSetGetDel_Bytes(t *testing.T) {
	ctx := context.Background()

	key := "test_key_bytes"
	value := []byte("content data")

	// Set byte slice value
	err := c.Set(ctx, key, value, 10*time.Second)
	assert.NoError(t, err, "error while setting byte slice value")

	// Get byte slice value
	var wanted []byte
	err = c.Get(ctx, key, &wanted)
	assert.NoError(t, err, "error while getting byte slice value")
	assert.Equal(t, value, wanted, "byte slices not equal")

	// Wait for value to expire
	time.Sleep(10 * time.Second)

	// Verify struct is expired
	err = c.Get(ctx, key, &wanted)
	assert.Error(t, err, "bytes should have been expired from cache")

	// Set byte slice value
	err = c.Set(ctx, key, value, 10*time.Second)
	assert.NoError(t, err, "error while setting byte slice value")

	// Delete byte slice value
	err = c.Del(ctx, key)
	assert.NoError(t, err, "error while deleting byte slice value")

	// Verify byte slice is deleted
	err = c.Get(ctx, key, &wanted)
	assert.Error(t, err, "byte slice should have been deleted from cache")

	// Test with multiple byte slices
	items := map[string][]byte{
		"file:1": []byte("content of file 1"),
		"file:2": []byte("content of file 2"),
		"file:3": []byte("content of file 3"),
	}

	for k, v := range items {
		err = c.Set(ctx, k, v, 10*time.Second)
		assert.NoError(t, err, "error while setting byte slice value")
	}

	// Verify all byte slices are stored correctly
	for k, expected := range items {
		var actual []byte
		err = c.Get(ctx, k, &actual)
		assert.NoError(t, err, "error while getting byte slice value")
		assert.Equal(t, expected, actual, "byte slices not equal")
	}

	// Delete all byte slices
	err = c.Del(ctx, slices.Collect(maps.Keys(items))...)
	assert.NoError(t, err, "error while deleting byte slices")
}

// TestDelWildCard tests the DelWildCard method with different data types
func (c *TestRedisCache) TestDelWildCard(t *testing.T) {
	ctx := context.Background()

	// String items
	stringItems := map[string]string{
		"test_key1": "test_value1",
		"test_key2": "test_value2",
		"test_key3": "test_value3",
		"test_key4": "test_value4",
	}

	for k, v := range stringItems {
		err := c.Set(ctx, k, v, 10*time.Second)
		assert.NoError(t, err, "error while setting string value")
	}

	// Struct items
	structItems := map[string]TestObject{
		"user:1": {ID: "1", Name: "Alice", Age: 25},
		"user:2": {ID: "2", Name: "Bob", Age: 35},
	}

	for k, v := range structItems {
		err := c.Set(ctx, k, v, 10*time.Second)
		assert.NoError(t, err, "error while setting struct value")
	}

	// Delete only string items with wildcard
	err := c.DelWildCard(ctx, "test_*")
	assert.NoError(t, err, "error while deleting with wildcard")

	// Verify string items are deleted
	for k := range stringItems {
		var wanted string
		err = c.Get(ctx, k, &wanted)
		assert.Error(t, err, "string value should have been deleted from cache")
	}

	// Verify struct items are still present
	for k, expected := range structItems {
		var actual TestObject
		err = c.Get(ctx, k, &actual)
		assert.NoError(t, err, "struct value should still be present")
		assert.Equal(t, expected, actual, "struct values not equal")
	}

	// Delete struct items with wildcard
	err = c.DelWildCard(ctx, "user:*")
	assert.NoError(t, err, "error while deleting structs with wildcard")

	// Verify struct items are deleted
	for k := range structItems {
		var actual TestObject
		err = c.Get(ctx, k, &actual)
		assert.Error(t, err, "struct should have been deleted from cache")
	}
}

// TestCache runs all cache tests
func TestCache(t *testing.T) {
	// Setup test configuration
	cfg := RedisConfig{
		Host:     "localhost",
		Port:     6379,
		Password: "",
		Database: 0,
	}

	// Create RedisCache instance for testing
	instance := NewTestCache(t, cfg)

	// Run all tests
	t.Run("String_Values", instance.TestSetGetDel_String)
	t.Run("Struct_Values", instance.TestSetGetDel_Struct)
	t.Run("Bytes_Values", instance.TestSetGetDel_Bytes)
	t.Run("Wildcard_Delete", instance.TestDelWildCard)
}
