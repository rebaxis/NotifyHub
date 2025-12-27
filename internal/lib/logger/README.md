# Logger Package

Universal logging library based on Uber Zap with custom abstractions.

## Features

- **Custom Logger Wrapper**: Provides a clean API over zap.Logger
- **Context Integration**: Store and retrieve logger from context
- **HTTP Middleware**: Automatic request/response logging
- **Component Scoping**: Create child loggers with component names
- **Formatted Logging**: Support for both structured and formatted logging
- **Error Chaining**: Convenience methods for error logging

## Usage

### Basic Setup

```go
import "github.com/notifyhub/notifyhub/internal/lib/logger"

// Create logger
log, err := logger.New(logger.Config{
    Level:       "info",      // debug, info, warn, error
    Development: false,
})
if err != nil {
    panic(err)
}
defer log.Sync()
```

### Structured Logging

```go
log.Info("server started",
    zap.String("addr", ":8080"),
    zap.Int("port", 8080),
)

log.Error("request failed",
    zap.String("method", "GET"),
    zap.Error(err),
)
```

### Formatted Logging

```go
log.Infof("starting server on port %d", 8080)
log.Errorf("failed to connect: %v", err)
```

### Component Scoping

```go
// Create scoped logger
dbLogger := log.WithComponent("database")
dbLogger.Info("connection established")  // Output: [database] connection established

repoLogger := log.WithComponent("repository")
repoLogger.Info("query executed")        // Output: [repository] query executed
```

### Context Integration

```go
// Add logger to context
ctx := logger.ToContext(ctx, log)

// Retrieve logger from context
log := logger.FromContext(ctx)
log.Info("processing request")

// With default fallback
log := logger.FromContextOrDefault(ctx, defaultLogger)
```

### HTTP Middleware

```go
import (
    "github.com/go-chi/chi/v5"
    "github.com/notifyhub/notifyhub/internal/lib/logger"
)

r := chi.NewRouter()

// Add logging middleware
r.Use(logger.HTTPMiddleware(log))

// All requests will be logged automatically
r.Get("/api/users", handler)
```

Output example:
```json
{
  "level": "info",
  "msg": "http request",
  "method": "GET",
  "path": "/api/users",
  "remote_addr": "127.0.0.1:52341",
  "status": 200,
  "bytes": 1523,
  "duration": 0.023,
  "user_agent": "curl/7.64.1"
}
```

### With Fields

```go
// Add fields to logger
userLogger := log.WithFields(
    zap.String("user_id", "123"),
    zap.String("tenant", "acme"),
)

userLogger.Info("action performed")
// Output includes user_id and tenant fields
```

### With Error

```go
// Convenience method for error logging
log.WithError(err).Error("operation failed")

// Equivalent to:
log.Error("operation failed", zap.Error(err))
```

## Integration Example

```go
package main

import (
    "github.com/notifyhub/notifyhub/internal/lib/logger"
    "go.uber.org/fx"
)

func NewLogger(cfg *config.Config) (*logger.Logger, error) {
    return logger.New(logger.Config{
        Level:       cfg.LogLevel,
        Development: cfg.Env == "development",
    })
}

func main() {
    fx.New(
        fx.Provide(NewLogger),
        // ... other providers
    ).Run()
}
```

## Log Levels

- **debug**: Detailed information for diagnosing problems
- **info**: General informational messages
- **warn**: Warning messages for potentially harmful situations
- **error**: Error messages for failures
- **fatal**: Critical errors that cause program termination

## Best Practices

1. **Use Structured Logging**: Prefer structured fields over formatted strings for better parsing
2. **Create Component Loggers**: Use `WithComponent()` to scope loggers by module
3. **Add Context**: Use `ToContext()` to pass logger through the call chain
4. **Flush on Exit**: Always call `Sync()` before program termination
5. **Avoid Sensitive Data**: Never log passwords, tokens, or PII

## Example in Handlers

```go
type UserHandler struct {
    logger *logger.Logger
}

func NewUserHandler(log *logger.Logger) *UserHandler {
    return &UserHandler{
        logger: log.WithComponent("user_handler"),
    }
}

func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
    userID := chi.URLParam(r, "user_id")
    
    h.logger.Info("fetching user",
        zap.String("user_id", userID),
    )
    
    user, err := h.service.GetUser(r.Context(), userID)
    if err != nil {
        h.logger.WithError(err).Error("failed to fetch user")
        http.Error(w, "internal error", 500)
        return
    }
    
    h.logger.Infof("user %s fetched successfully", userID)
    json.NewEncoder(w).Encode(user)
}
```

## Performance

The logger uses Uber Zap which provides:
- Zero allocation for structured logging
- Fast serialization
- Minimal overhead compared to standard library
- Production-ready performance

