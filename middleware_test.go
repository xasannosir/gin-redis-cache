package cache

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// Mock response structure
type TestResponse struct {
	Message string `json:"message"`
	ID      string `json:"id"`
}

// setupTestRouter creates a test router with cache middleware
func setupTestRouter(cache Cache, config CacheConfig) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(SetOrGetCache(cache, config))
	return router
}

// TestMiddleware_GetRequest_CacheHit tests that GET requests are cached and served from cache
func TestMiddleware_GetRequest_CacheHit(t *testing.T) {
	// Setup cache
	cfg := RedisConfig{
		Host:     "localhost",
		Port:     6379,
		Password: "",
		Database: 0,
	}
	cache, err := NewRedisCache(cfg)
	assert.NoError(t, err)

	// Setup middleware config
	config := CacheConfig{
		TTL:      10 * time.Second,
		Groups:   map[string][]string{},
		Outdoors: []string{},
		Logger:   func(format string, args ...interface{}) {},
	}

	router := setupTestRouter(cache, config)

	// Counter to track handler calls
	callCount := 0

	// Setup test endpoint
	router.GET("/v1/product/:id", func(c *gin.Context) {
		callCount++
		response := TestResponse{
			Message: "Product details",
			ID:      c.Param("id"),
		}
		c.JSON(http.StatusOK, response)
	})

	// First request - should hit handler
	req1 := httptest.NewRequest("GET", "/v1/product/123", nil)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	assert.Equal(t, http.StatusOK, w1.Code)
	assert.Equal(t, 1, callCount, "handler should be called once")

	var response1 TestResponse
	err = json.Unmarshal(w1.Body.Bytes(), &response1)
	assert.NoError(t, err)
	assert.Equal(t, "Product details", response1.Message)
	assert.Equal(t, "123", response1.ID)

	// Second request - should serve from cache
	req2 := httptest.NewRequest("GET", "/v1/product/123", nil)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusOK, w2.Code)
	assert.Equal(t, 1, callCount, "handler should NOT be called again (served from cache)")

	var response2 TestResponse
	err = json.Unmarshal(w2.Body.Bytes(), &response2)
	assert.NoError(t, err)
	assert.Equal(t, response1, response2, "cached response should match original")

	// Cleanup
	err = cache.Del(context.Background(), "/v1/product/123")
	assert.NoError(t, err)
}

// TestMiddleware_GetRequest_WithQueryParams tests cache key generation with query parameters
func TestMiddleware_GetRequest_WithQueryParams(t *testing.T) {
	cfg := RedisConfig{
		Host:     "localhost",
		Port:     6379,
		Password: "",
		Database: 0,
	}
	cache, err := NewRedisCache(cfg)
	assert.NoError(t, err)

	config := CacheConfig{
		TTL:      10 * time.Second,
		Groups:   map[string][]string{},
		Outdoors: []string{},
		Logger:   func(format string, args ...interface{}) {},
	}

	router := setupTestRouter(cache, config)

	callCount := 0
	router.GET("/v1/product", func(c *gin.Context) {
		callCount++
		category := c.Query("category")
		sort := c.Query("sort")
		response := TestResponse{
			Message: "Products list",
			ID:      category + "-" + sort,
		}
		c.JSON(http.StatusOK, response)
	})

	// First request with query params
	req1 := httptest.NewRequest("GET", "/v1/product?category=electronics&sort=price", nil)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	assert.Equal(t, http.StatusOK, w1.Code)
	assert.Equal(t, 1, callCount)

	// Same request - should serve from cache
	req2 := httptest.NewRequest("GET", "/v1/product?category=electronics&sort=price", nil)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusOK, w2.Code)
	assert.Equal(t, 1, callCount, "should serve from cache")

	// Different query params order - should still serve from cache (params are sorted)
	req3 := httptest.NewRequest("GET", "/v1/product?sort=price&category=electronics", nil)
	w3 := httptest.NewRecorder()
	router.ServeHTTP(w3, req3)

	assert.Equal(t, http.StatusOK, w3.Code)
	assert.Equal(t, 1, callCount, "should serve from cache even with different param order")

	// Different query params - should NOT serve from cache
	req4 := httptest.NewRequest("GET", "/v1/product?category=books&sort=name", nil)
	w4 := httptest.NewRecorder()
	router.ServeHTTP(w4, req4)

	assert.Equal(t, http.StatusOK, w4.Code)
	assert.Equal(t, 2, callCount, "different params should hit handler")

	// Cleanup
	err = cache.DelWildCard(context.Background(), "/v1/product*")
	assert.NoError(t, err)
}

// TestMiddleware_PostRequest_InvalidatesCache tests that POST requests invalidate cache
func TestMiddleware_PostRequest_InvalidatesCache(t *testing.T) {
	cfg := RedisConfig{
		Host:     "localhost",
		Port:     6379,
		Password: "",
		Database: 0,
	}
	cache, err := NewRedisCache(cfg)
	assert.NoError(t, err)

	config := CacheConfig{
		TTL:      10 * time.Second,
		Groups:   map[string][]string{},
		Outdoors: []string{},
		Logger:   func(format string, args ...interface{}) {},
	}

	router := setupTestRouter(cache, config)

	getCallCount := 0
	postCallCount := 0

	router.GET("/v1/product/:id", func(c *gin.Context) {
		getCallCount++
		response := TestResponse{
			Message: "Product details",
			ID:      c.Param("id"),
		}
		c.JSON(http.StatusOK, response)
	})

	router.POST("/v1/product", func(c *gin.Context) {
		postCallCount++
		c.JSON(http.StatusCreated, gin.H{"message": "created"})
	})

	// First GET request - caches response
	req1 := httptest.NewRequest("GET", "/v1/product/123", nil)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)
	assert.Equal(t, 1, getCallCount)

	// Second GET request - serves from cache
	req2 := httptest.NewRequest("GET", "/v1/product/123", nil)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	assert.Equal(t, 1, getCallCount, "should serve from cache")

	// POST request - invalidates cache
	reqPost := httptest.NewRequest("POST", "/v1/product", strings.NewReader(`{"name":"new product"}`))
	reqPost.Header.Set("Content-Type", "application/json")
	wPost := httptest.NewRecorder()
	router.ServeHTTP(wPost, reqPost)
	assert.Equal(t, http.StatusCreated, wPost.Code)
	assert.Equal(t, 1, postCallCount)

	// Third GET request - cache invalidated, should hit handler again
	req3 := httptest.NewRequest("GET", "/v1/product/123", nil)
	w3 := httptest.NewRecorder()
	router.ServeHTTP(w3, req3)
	assert.Equal(t, 2, getCallCount, "cache should be invalidated after POST")

	// Cleanup
	err = cache.DelWildCard(context.Background(), "/v1/product*")
	assert.NoError(t, err)
}

// TestMiddleware_PutRequest_InvalidatesCache tests that PUT requests invalidate cache
func TestMiddleware_PutRequest_InvalidatesCache(t *testing.T) {
	cfg := RedisConfig{
		Host:     "localhost",
		Port:     6379,
		Password: "",
		Database: 0,
	}
	cache, err := NewRedisCache(cfg)
	assert.NoError(t, err)

	config := CacheConfig{
		TTL:      10 * time.Second,
		Groups:   map[string][]string{},
		Outdoors: []string{},
		Logger:   func(format string, args ...interface{}) {},
	}

	router := setupTestRouter(cache, config)

	getCallCount := 0
	putCallCount := 0

	router.GET("/v1/product/:id", func(c *gin.Context) {
		getCallCount++
		c.JSON(http.StatusOK, gin.H{"message": "product"})
	})

	router.PUT("/v1/product/:id", func(c *gin.Context) {
		putCallCount++
		c.JSON(http.StatusOK, gin.H{"message": "updated"})
	})

	// Cache the response
	req1 := httptest.NewRequest("GET", "/v1/product/123", nil)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)
	assert.Equal(t, 1, getCallCount)

	// PUT request - invalidates cache
	reqPut := httptest.NewRequest("PUT", "/v1/product/123", strings.NewReader(`{"name":"updated"}`))
	wPut := httptest.NewRecorder()
	router.ServeHTTP(wPut, reqPut)
	assert.Equal(t, 1, putCallCount)

	// Should hit handler again
	req2 := httptest.NewRequest("GET", "/v1/product/123", nil)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	assert.Equal(t, 2, getCallCount, "cache should be invalidated after PUT")

	// Cleanup
	err = cache.DelWildCard(context.Background(), "/v1/product*")
	assert.NoError(t, err)
}

// TestMiddleware_PatchRequest_InvalidatesCache tests that PATCH requests invalidate cache
func TestMiddleware_PatchRequest_InvalidatesCache(t *testing.T) {
	cfg := RedisConfig{
		Host:     "localhost",
		Port:     6379,
		Password: "",
		Database: 0,
	}
	cache, err := NewRedisCache(cfg)
	assert.NoError(t, err)

	config := CacheConfig{
		TTL:      10 * time.Second,
		Groups:   map[string][]string{},
		Outdoors: []string{},
		Logger:   func(format string, args ...interface{}) {},
	}

	router := setupTestRouter(cache, config)

	getCallCount := 0
	putCallCount := 0

	router.GET("/v1/product/:id", func(c *gin.Context) {
		getCallCount++
		c.JSON(http.StatusOK, gin.H{"message": "product"})
	})

	router.PATCH("/v1/product/:id", func(c *gin.Context) {
		putCallCount++
		c.JSON(http.StatusOK, gin.H{"message": "updated"})
	})

	// Cache the response
	req1 := httptest.NewRequest("GET", "/v1/product/123", nil)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)
	assert.Equal(t, 1, getCallCount)

	// PUT request - invalidates cache
	reqPut := httptest.NewRequest("PATCH", "/v1/product/123", strings.NewReader(`{"name":"updated"}`))
	wPut := httptest.NewRecorder()
	router.ServeHTTP(wPut, reqPut)
	assert.Equal(t, 1, putCallCount)

	// Should hit handler again
	req2 := httptest.NewRequest("GET", "/v1/product/123", nil)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	assert.Equal(t, 2, getCallCount, "cache should be invalidated after PUT")

	// Cleanup
	err = cache.DelWildCard(context.Background(), "/v1/product*")
	assert.NoError(t, err)
}

// TestMiddleware_DeleteRequest_InvalidatesCache tests that DELETE requests invalidate cache
func TestMiddleware_DeleteRequest_InvalidatesCache(t *testing.T) {
	cfg := RedisConfig{
		Host:     "localhost",
		Port:     6379,
		Password: "",
		Database: 0,
	}
	cache, err := NewRedisCache(cfg)
	assert.NoError(t, err)

	config := CacheConfig{
		TTL:      10 * time.Second,
		Groups:   map[string][]string{},
		Outdoors: []string{},
		Logger:   func(format string, args ...interface{}) {},
	}

	router := setupTestRouter(cache, config)

	getCallCount := 0
	deleteCallCount := 0

	router.GET("/v1/product/:id", func(c *gin.Context) {
		getCallCount++
		c.JSON(http.StatusOK, gin.H{"message": "product"})
	})

	router.DELETE("/v1/product/:id", func(c *gin.Context) {
		deleteCallCount++
		c.JSON(http.StatusOK, gin.H{"message": "deleted"})
	})

	// Cache the response
	req1 := httptest.NewRequest("GET", "/v1/product/123", nil)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)
	assert.Equal(t, 1, getCallCount)

	// DELETE request - invalidates cache
	reqDel := httptest.NewRequest("DELETE", "/v1/product/123", nil)
	wDel := httptest.NewRecorder()
	router.ServeHTTP(wDel, reqDel)
	assert.Equal(t, 1, deleteCallCount)

	// Should hit handler again
	req2 := httptest.NewRequest("GET", "/v1/product/123", nil)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	assert.Equal(t, 2, getCallCount, "cache should be invalidated after DELETE")

	// Cleanup
	err = cache.DelWildCard(context.Background(), "/v1/product*")
	assert.NoError(t, err)
}

// TestMiddleware_Groups_InvalidatesRelatedCaches tests that Groups invalidate related resources
func TestMiddleware_Groups_InvalidatesRelatedCaches(t *testing.T) {
	cfg := RedisConfig{
		Host:     "localhost",
		Port:     6379,
		Password: "",
		Database: 0,
	}
	cache, err := NewRedisCache(cfg)
	assert.NoError(t, err)

	// Configure groups: when product changes, category cache should be invalidated
	config := CacheConfig{
		TTL: 10 * time.Second,
		Groups: map[string][]string{
			"product": {"category", "brand"},
		},
		Outdoors: []string{},
		Logger:   func(format string, args ...interface{}) {},
	}

	router := setupTestRouter(cache, config)

	productCallCount := 0
	categoryCallCount := 0
	brandCallCount := 0

	router.GET("/v1/product/:id", func(c *gin.Context) {
		productCallCount++
		c.JSON(http.StatusOK, gin.H{"message": "product"})
	})

	router.GET("/v1/category/:id", func(c *gin.Context) {
		categoryCallCount++
		c.JSON(http.StatusOK, gin.H{"message": "category"})
	})

	router.GET("/v1/brand/:id", func(c *gin.Context) {
		brandCallCount++
		c.JSON(http.StatusOK, gin.H{"message": "brand"})
	})

	router.POST("/v1/product", func(c *gin.Context) {
		c.JSON(http.StatusCreated, gin.H{"message": "created"})
	})

	// Cache all resources
	reqProduct := httptest.NewRequest("GET", "/v1/product/123", nil)
	wProduct := httptest.NewRecorder()
	router.ServeHTTP(wProduct, reqProduct)
	assert.Equal(t, 1, productCallCount)

	reqCategory := httptest.NewRequest("GET", "/v1/category/456", nil)
	wCategory := httptest.NewRecorder()
	router.ServeHTTP(wCategory, reqCategory)
	assert.Equal(t, 1, categoryCallCount)

	reqBrand := httptest.NewRequest("GET", "/v1/brand/789", nil)
	wBrand := httptest.NewRecorder()
	router.ServeHTTP(wBrand, reqBrand)
	assert.Equal(t, 1, brandCallCount)

	// Verify caches are working
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/v1/product/123", nil))
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/v1/category/456", nil))
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/v1/brand/789", nil))
	assert.Equal(t, 1, productCallCount, "product should be cached")
	assert.Equal(t, 1, categoryCallCount, "category should be cached")
	assert.Equal(t, 1, brandCallCount, "brand should be cached")

	// POST to product - should invalidate product, category, and brand
	reqPost := httptest.NewRequest("POST", "/v1/product", strings.NewReader(`{}`))
	wPost := httptest.NewRecorder()
	router.ServeHTTP(wPost, reqPost)

	// All related caches should be invalidated
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/v1/product/123", nil))
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/v1/category/456", nil))
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/v1/brand/789", nil))

	assert.Equal(t, 2, productCallCount, "product cache should be invalidated")
	assert.Equal(t, 2, categoryCallCount, "category cache should be invalidated (related)")
	assert.Equal(t, 2, brandCallCount, "brand cache should be invalidated (related)")

	// Cleanup
	err = cache.DelWildCard(context.Background(), "/v1/*")
	assert.NoError(t, err)
}

// TestMiddleware_Outdoors_SkipsCaching tests that Outdoors paths are not cached
func TestMiddleware_Outdoors_SkipsCaching(t *testing.T) {
	cfg := RedisConfig{
		Host:     "localhost",
		Port:     6379,
		Password: "",
		Database: 0,
	}
	cache, err := NewRedisCache(cfg)
	assert.NoError(t, err)

	// Configure auth endpoint to not be cached
	config := CacheConfig{
		TTL:      10 * time.Second,
		Groups:   map[string][]string{},
		Outdoors: []string{"auth", "health"},
		Logger:   func(format string, args ...interface{}) {},
	}

	router := setupTestRouter(cache, config)

	authCallCount := 0
	productCallCount := 0

	router.GET("/v1/auth/me", func(c *gin.Context) {
		authCallCount++
		c.JSON(http.StatusOK, gin.H{"user": "current user"})
	})

	router.GET("/v1/product/:id", func(c *gin.Context) {
		productCallCount++
		c.JSON(http.StatusOK, gin.H{"message": "product"})
	})

	// Auth endpoint - should NOT be cached
	req1 := httptest.NewRequest("GET", "/v1/auth/me", nil)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)
	assert.Equal(t, 1, authCallCount)

	req2 := httptest.NewRequest("GET", "/v1/auth/me", nil)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	assert.Equal(t, 2, authCallCount, "auth should NOT be cached")

	// Product endpoint - should be cached
	req3 := httptest.NewRequest("GET", "/v1/product/123", nil)
	w3 := httptest.NewRecorder()
	router.ServeHTTP(w3, req3)
	assert.Equal(t, 1, productCallCount)

	req4 := httptest.NewRequest("GET", "/v1/product/123", nil)
	w4 := httptest.NewRecorder()
	router.ServeHTTP(w4, req4)
	assert.Equal(t, 1, productCallCount, "product should be cached")

	// Cleanup
	err = cache.DelWildCard(context.Background(), "/v1/*")
	assert.NoError(t, err)
}

// TestMiddleware_NonOKStatus_DoesNotCache tests that non-200 responses are not cached
func TestMiddleware_NonOKStatus_DoesNotCache(t *testing.T) {
	cfg := RedisConfig{
		Host:     "localhost",
		Port:     6379,
		Password: "",
		Database: 0,
	}
	cache, err := NewRedisCache(cfg)
	assert.NoError(t, err)

	config := CacheConfig{
		TTL:      10 * time.Second,
		Groups:   map[string][]string{},
		Outdoors: []string{},
		Logger:   func(format string, args ...interface{}) {},
	}

	router := setupTestRouter(cache, config)

	callCount := 0
	router.GET("/v1/product/:id", func(c *gin.Context) {
		callCount++
		if callCount == 1 {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		} else {
			c.JSON(http.StatusOK, gin.H{"message": "product"})
		}
	})

	// First request - returns 404, should NOT be cached
	req1 := httptest.NewRequest("GET", "/v1/product/999", nil)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusNotFound, w1.Code)
	assert.Equal(t, 1, callCount)

	// Second request - should hit handler again (404 was not cached)
	req2 := httptest.NewRequest("GET", "/v1/product/999", nil)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)
	assert.Equal(t, 2, callCount, "404 response should NOT be cached")

	// Third request - should serve from cache (200 response)
	req3 := httptest.NewRequest("GET", "/v1/product/999", nil)
	w3 := httptest.NewRecorder()
	router.ServeHTTP(w3, req3)
	assert.Equal(t, http.StatusOK, w3.Code)
	assert.Equal(t, 2, callCount, "200 response should be cached")

	// Cleanup
	err = cache.DelWildCard(context.Background(), "/v1/*")
	assert.NoError(t, err)
}
