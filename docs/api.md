# k8s-conductor-framework API Reference

Complete API reference for the k8s-conductor-framework REST API.

## Base URL

All API endpoints are relative to the base URL: `http://localhost:8081` (configurable via `PORT`).

## Authentication

Currently, the API does not require authentication. In production, you should add authentication middleware.

## Response Format

### Success Response

```json
{
  "data": {...}
}
```

### Error Response

```json
{
  "error": "error_code",
  "message": "Human-readable error message",
  "details": {
    "key": "value"
  }
}
```

## Endpoints

### Health Endpoints

#### GET /healthz

Health check endpoint.

**Response:**
```json
{
  "status": "healthy",
  "version": "1.0.0",
  "timestamp": "2024-01-01T00:00:00Z"
}
```

#### GET /readyz

Readiness check endpoint. Returns component status.

**Response:**
```json
{
  "status": "healthy",
  "version": "1.0.0",
  "timestamp": "2024-01-01T00:00:00Z",
  "components": {
    "database": {
      "status": "healthy"
    },
    "manager": {
      "status": "ready"
    },
    "eventStore": {
      "status": "available"
    }
  }
}
```

### Manifest Endpoints

#### GET /manifests

List all manifests.

**Response:**
```json
{
  "default/Deployment/redis": "...",
  "default/Service/redis": "..."
}
```

#### GET /manifests/*

Get a specific manifest by key.

**Path Parameters:**
- `*` - Manifest key (e.g., `default/Deployment/redis`)

**Response:** YAML content

**Errors:**
- `404` - Manifest not found

#### POST /manifests

Create a manifest override.

**Request Body:**
```json
{
  "key": "default/Deployment/redis",
  "value": "apiVersion: apps/v1\nkind: Deployment\n..."
}
```

**Response:** `201 Created`

**Errors:**
- `400` - Invalid request (invalid YAML, missing fields)

#### PUT /manifests/*

Update a manifest override.

**Path Parameters:**
- `*` - Manifest key

**Request Body:**
```json
{
  "value": "apiVersion: apps/v1\nkind: Deployment\n..."
}
```

**Response:** `200 OK`

**Errors:**
- `400` - Invalid request
- `404` - Manifest not found

#### DELETE /manifests/*

Delete a manifest override.

**Path Parameters:**
- `*` - Manifest key

**Response:** `204 No Content`

**Errors:**
- `404` - Manifest not found

### Deployment Endpoints

#### POST /api/up

Deploy all or selected services.

**Request Body (optional):**
```json
{
  "services": ["redis", "postgresql"]
}
```

If `services` is omitted, all services are deployed.

**Response:**
```json
{
  "message": "Successfully deployed all manifests"
}
```

**Errors:**
- `500` - Deployment failed

#### POST /api/down

Remove all or selected services.

**Request Body (optional):**
```json
{
  "services": ["redis"]
}
```

**Response:**
```json
{
  "message": "Successfully deleted all managed resources"
}
```

#### POST /api/update

Update all or selected services.

**Request Body (optional):**
```json
{
  "services": ["redis", "postgresql"]
}
```

**Response:**
```json
{
  "message": "Successfully updated all manifests"
}
```

### Event Endpoints

#### GET /api/events

List events with optional filtering.

**Query Parameters:**
- `resource` - Filter by resource key
- `type` - Filter by event type (`error`, `success`, `info`, `warning`)
- `since` - Filter events after timestamp (RFC3339)
- `until` - Filter events before timestamp (RFC3339)
- `limit` - Maximum number of events (default: 100, max: 1000)
- `offset` - Pagination offset

**Example:**
```
GET /api/events?type=error&limit=10&since=2024-01-01T00:00:00Z
```

**Response:**
```json
{
  "events": [
    {
      "id": "...",
      "timestamp": "2024-01-01T00:00:00Z",
      "type": "error",
      "resourceKey": "default/Deployment/redis",
      "message": "Deployment failed",
      "details": {}
    }
  ],
  "total": 100,
  "limit": 10,
  "offset": 0
}
```

#### GET /api/events/*

Get events for a specific resource.

**Path Parameters:**
- `*` - Resource key

**Query Parameters:**
- `limit` - Maximum number of events (default: 100)

**Response:** Same as `GET /api/events`

#### GET /api/events/errors

Get recent error events.

**Query Parameters:**
- `limit` - Maximum number of events (default: 50)

**Response:** Same as `GET /api/events`

#### DELETE /api/events

Cleanup old events.

**Query Parameters:**
- `before` - Delete events before timestamp (RFC3339, required)

**Response:**
```json
{
  "message": "Events cleaned up successfully"
}
```

### Service Endpoints

#### GET /api/services

List all services.

**Response:**
```json
{
  "services": [
    {
      "name": "redis",
      "namespace": "default",
      "port": 6379,
      "installed": true
    }
  ]
}
```

#### GET /api/services/health

Get health status for all services.

**Response:**
```json
{
  "services": [
    {
      "name": "redis",
      "namespace": "default",
      "port": 6379,
      "status": "healthy",
      "endpoint": "/health",
      "lastChecked": "2024-01-01T00:00:00Z"
    }
  ]
}
```

#### GET /api/service/{namespace}/{name}

Get detailed information about a service.

**Path Parameters:**
- `namespace` - Service namespace
- `name` - Service name

**Response:**
```json
{
  "name": "redis",
  "env": [
    {
      "name": "REDIS_PASSWORD",
      "value": "secret",
      "source": "direct"
    }
  ]
}
```

### Parameter Endpoints

#### GET /api/parameters

Get deployment parameters.

**Response:**
```json
{
  "global": {
    "namespace": "default",
    "replicas": 1
  },
  "services": {
    "redis": {
      "replicas": 3
    }
  }
}
```

#### POST /api/parameters

Update deployment parameters.

**Request Body:**
```json
{
  "global": {
    "namespace": "default",
    "replicas": 1
  },
  "services": {
    "redis": {
      "replicas": 3,
      "imageTag": "7.0"
    }
  }
}
```

**Response:**
```json
{
  "message": "Parameters updated successfully"
}
```

#### GET /api/parameters/{service}

Get merged parameters for a specific service.

**Path Parameters:**
- `service` - Service name

**Response:**
```json
{
  "namespace": "default",
  "replicas": 3,
  "imageTag": "7.0"
}
```

#### GET /api/parameters/values

Get all parameter values (merged and deployed).

**Response:**
```json
{
  "redis": {
    "merged": {
      "namespace": "default",
      "replicas": 3
    },
    "deployed": {
      "namespace": "default",
      "replicas": 3
    }
  }
}
```

### Cluster Endpoints

#### GET /api/cluster/requirements

Check cluster requirements.

**Response:**
```json
{
  "requirements": [
    {
      "name": "Kubernetes Version",
      "description": "Kubernetes cluster version must be 1.24 or higher",
      "status": "pass",
      "message": "Version 1.28.0 meets requirement",
      "required": true
    }
  ],
  "overall": "pass"
}
```

## Error Codes

- `not_found` - Resource not found
- `validation_error` - Validation failed
- `invalid_yaml` - Invalid YAML format
- `invalid_request` - Invalid request format
- `storage_error` - Storage operation failed
- `kubernetes_error` - Kubernetes API error
- `reconciliation_error` - Reconciliation failed
- `event_store_unavailable` - Event store unavailable

## Rate Limiting

Currently, there is no rate limiting. In production, you should add rate limiting middleware.

## CORS

CORS is enabled by default for all origins. In production, you should restrict CORS to specific origins.

