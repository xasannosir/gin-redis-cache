package cache

import (
	"bytes"
	"net/http"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type CacheConfig struct {
	// TTL is the default time-to-live for cached responses
	TTL time.Duration

	// Groups defines cache invalidation relationships between resources
	// When a resource is modified, all related resources in its group are invalidated
	Groups map[string][]string

	// Outdoors (ExcludedPaths) lists API endpoints that should not be cached
	Outdoors []string

	// Logger is an optional custom logger function
	Logger func(format string, args ...interface{})
}

// responseWriter wraps gin.ResponseWriter to capture response body for caching
type responseWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

// Write captures the response body while writing to the original writer
func (w *responseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// getBaseURL extracts the resource type from the URL path
// For example, "/v1/product/123" returns "product"
func getBaseURL(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
}

// getCacheKey generates a unique cache key from the request path and query parameters
// Query parameters are sorted alphabetically to ensure consistent keys
func getCacheKey(c *gin.Context) string {
	path := c.Request.URL.Path
	params := c.Request.URL.Query()

	if len(params) > 0 {
		keys := make([]string, 0, len(params))
		for k := range params {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		var queryParts []string
		for _, k := range keys {
			for _, v := range params[k] {
				queryParts = append(queryParts, k+"="+v)
			}
		}
		return path + "?" + strings.Join(queryParts, "&")
	}

	return path
}

// SetOrGetCache returns a Gin middleware that handles HTTP caching
// GET requests: serve from cache if available, otherwise cache the response
// POST/PUT/PATCH/DELETE requests: invalidate related caches
func SetOrGetCache(cache Cache, config CacheConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method
		path := c.Request.URL.Path

		baseURL := getBaseURL(path)

		// Skip caching for excluded endpoints
		if slices.Contains(config.Outdoors, baseURL) {
			c.Next()
			return
		}

		// Handle cache invalidation for mutating operations
		if method == "POST" || method == "PUT" || method == "PATCH" || method == "DELETE" {

			// Invalidate all caches for this resource type
			err := cache.DelWildCard(c.Request.Context(), "/v1/"+baseURL+"*")
			if err != nil {
				config.Logger("setOrGetCache.delWildCard baseUrl", err)
			}

			// Invalidate caches for related resource types
			if relatedPaths, ok := config.Groups[baseURL]; ok {
				for _, relatedPath := range relatedPaths {
					err = cache.DelWildCard(c.Request.Context(), "/v1/"+relatedPath+"*")
					if err != nil {
						config.Logger("setOrGetCache.delWildCard relatedPath", err)
					}
				}
			}

			c.Next()
			return
		}

		// Handle cache retrieval and storage for GET requests
		if method == "GET" {
			cacheKey := getCacheKey(c)

			// Try to get cached response
			var cachedBytes []byte
			err := cache.Get(c.Request.Context(), cacheKey, &cachedBytes)

			if err != nil {
				config.Logger("setOrGetCache.get cacheKey", err)
			}

			// Serve from cache if available
			if err == nil && len(cachedBytes) > 0 {
				c.Data(http.StatusOK, "application/json; charset=utf-8", cachedBytes)
				c.Abort()
				return
			}

			// Cache miss: capture response for caching
			writer := &responseWriter{
				ResponseWriter: c.Writer,
				body:           bytes.NewBufferString(""),
			}
			c.Writer = writer

			c.Next()

			// Cache successful responses only
			if writer.Status() == http.StatusOK && writer.body.Len() > 0 {
				err = cache.Set(c.Request.Context(), cacheKey, writer.body.Bytes(), config.TTL)
				if err != nil {
					config.Logger("setOrGetCache.set cacheKey", err)
				}
			}
			return
		}

		// Pass through for other HTTP methods
		c.Next()
	}
}
