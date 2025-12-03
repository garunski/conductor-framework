package api

import "time"

// DefaultRequestTimeout is the default timeout for API requests
const DefaultRequestTimeout = 5 * time.Second

// DefaultHealthCheckTimeout is the default timeout for individual health check requests
const DefaultHealthCheckTimeout = 2 * time.Second

