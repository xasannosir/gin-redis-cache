# Gin Redis Cache

A lightweight, production-ready Redis caching middleware for Gin web framework. This package provides intelligent HTTP response caching with automatic cache invalidation support.

## Features

- **Easy Integration**: Simple Gin middleware for HTTP caching
- **Redis Backend**: Uses Redis for distributed caching
- **Smart Cache Invalidation**: Automatically invalidates related cache entries on mutations
- **Query Parameter Support**: Generates unique cache keys based on paths and query parameters
- **Configurable TTL**: Per-endpoint or global time-to-live settings
- **Flexible Exclusion**: Skip caching for specific endpoints
- **Resource Grouping**: Define relationships between resources for cascading invalidation
- **Custom Logging**: Optional logger function for debugging

## Installation

To install the cache package, use the following command:

```bash
go get github.com/xasannosir/gin-redis-cache
```

## Usage

Import the cache package in your Go code:

```bash
import "github.com/xasannosir/gin-redis-cache"
```

## Usage Examples

```go
package main

import (
    "fmt"
    "time"
	
    "github.com/gin-gonic/gin"
    "github.com/xasannosir/gin-redis-cache"
)

func main() {
    // Initialize cache
    cacheInstance, err := cache.NewRedisCache(cache.RedisConfig{
        Host:     "localhost",
        Port:     6379,
        Password: "",
        Database: 0,
    })
    if err != nil {
        fmt.Printf("Failed to initialize cache: %v\n", err)
        return
    }

    // Setup Gin router
    router := gin.Default()

    // Configure cache behavior
    config := cache.CacheConfig{
        TTL: 10 * time.Minute,
        Outdoors: []string{"auth", "health", "login"},
        Groups: map[string][]string{
            "product": {"inventory", "category"},
            "user":    {"profile", "settings"},
        },
        Logger: func(format string, args ...interface{}) {
            fmt.Printf("[Cache] "+format+"\n", args...)
        },
    }

    // Apply caching middleware
    router.Use(cache.SetOrGetCache(cacheInstance, config))

    // Define your routes here ...

    router.Run(":8080")
}
```

## Dependencies

- `github.com/gin-gonic/gin` - HTTP web framework
- `github.com/redis/go-redis/v9` - Redis client

## Contributing

Contributions are welcome! Please feel free to submit pull requests or open issues.
