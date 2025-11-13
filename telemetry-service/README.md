# Telemetry Service

A centralized observability service for microservices built with Go, Zap logger, Prometheus, and Grafana. This service provides logging and metrics collection endpoints that can be used by any microservice regardless of the programming language.

## Features

- 🚀 **Centralized Logging** with Zap logger
- 📊 **Metrics Collection** with Prometheus
- 📈 **Visualization** with Grafana dashboards
- 🌐 **Language Agnostic** - Works with any programming language (Go, Node.js, PHP, Python, etc.)
- 🔍 **Distributed Tracing** support with trace IDs
- 📦 **Batch Processing** for efficient log ingestion
- 🐳 **Docker Support** for easy deployment

## Architecture

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│  Node.js/PHP/   │────▶│  Telemetry      │────▶│  Prometheus     │
│  Any Service    │     │  Service        │     │                 │
└─────────────────┘     │  (Port 8080)    │     │  (Port 9090)    │
                        └─────────────────┘     └─────────────────┘
                                │                         │
                                │                         │
                                ▼                         ▼
                        ┌─────────────────┐     ┌─────────────────┐
                        │  Zap Logger     │     │  Grafana        │
                        │  (Logs)         │     │  (Port 3000)    │
                        └─────────────────┘     └─────────────────┘
```

## Quick Start

### Prerequisites

- Docker & Docker Compose
- Go 1.23+ (for development)

### Run with Docker Compose

```bash
cd telemetry-service
docker-compose up -d
```

This will start:
- **Telemetry Service** on `http://localhost:8080`
- **Prometheus** on `http://localhost:9090`
- **Grafana** on `http://localhost:3000` (admin/admin)

### Build and Run Locally

```bash
cd telemetry-service

# Download dependencies
go mod download

# Build
go build -o telemetry-service ./cmd/main.go

# Run
./telemetry-service
```

## API Endpoints

### Health Check
```
GET /api/v1/health
GET /
```

**Response:**
```json
{
  "status": "healthy",
  "timestamp": "2025-11-13T10:00:00Z",
  "service": "telemetry-service"
}
```

### Send Log Entry
```
POST /api/v1/logs
```

**Request Body:**
```json
{
  "service_name": "my-service",
  "level": "info",
  "message": "User logged in successfully",
  "timestamp": "2025-11-13T10:00:00Z",
  "fields": {
    "user_id": 123,
    "ip_address": "192.168.1.1"
  },
  "trace_id": "abc123",
  "span_id": "xyz789"
}
```

**Supported Log Levels:** `debug`, `info`, `warn`, `error`, `fatal`, `panic`

### Send Batch Logs
```
POST /api/v1/logs/batch
```

**Request Body:**
```json
[
  {
    "service_name": "my-service",
    "level": "info",
    "message": "First log entry",
    "fields": {"key": "value"}
  },
  {
    "service_name": "my-service",
    "level": "error",
    "message": "Second log entry",
    "fields": {"error": "something went wrong"}
  }
]
```

### Send Metrics
```
POST /api/v1/metrics
```

**Request Body:**
```json
{
  "service_name": "my-service",
  "metric_name": "http_requests_total",
  "metric_type": "counter",
  "value": 1,
  "labels": {
    "method": "GET",
    "path": "/api/users",
    "status": "200"
  }
}
```

**Supported Metric Types:**
- `counter` - Monotonically increasing value
- `gauge` - Value that can go up or down
- `histogram` - Distribution of values

### Prometheus Metrics Endpoint
```
GET /metrics
```

Returns metrics in Prometheus format for scraping.

## Client Libraries

### Node.js/JavaScript

```bash
npm install axios
```

```javascript
const TelemetryClient = require('./examples/nodejs-client');
const telemetry = new TelemetryClient('my-nodejs-service');

// Logs
await telemetry.info('Application started', { version: '1.0.0' });
await telemetry.error('Database error', { error: 'connection timeout' });

// Metrics
await telemetry.incrementCounter('http_requests', 1, { method: 'GET' });
await telemetry.setGauge('active_users', 42);
```

See full example: `examples/nodejs-client.js`

### PHP

```php
<?php
require_once 'examples/php-client.php';

$telemetry = new TelemetryClient('my-php-service');

// Logs
$telemetry->info('Application started', ['version' => '1.0.0']);
$telemetry->error('Database error', ['error' => 'connection timeout']);

// Metrics
$telemetry->incrementCounter('http_requests', 1, ['method' => 'GET']);
$telemetry->setGauge('active_users', 42);
?>
```

See full example: `examples/php-client.php`

### Other Languages

You can use any HTTP client library in your language of choice. Just send POST requests to the API endpoints with the proper JSON structure.

## Configuration

Edit `config/config.yaml`:

```yaml
server:
  port: 8080
  host: "0.0.0.0"

logger:
  level: "info"  # debug, info, warn, error
  encoding: "json"  # json or console
  output_paths:
    - "stdout"
    - "logs/telemetry.log"

prometheus:
  enabled: true
  port: 9090

grafana:
  enabled: true
  port: 3000
```

## Grafana Dashboard

1. Access Grafana at `http://localhost:3000`
2. Login with `admin/admin`
3. Navigate to **Dashboards** → **Telemetry Service Dashboard**

The dashboard includes:
- HTTP request metrics
- Request duration percentiles
- Logs received by service and level
- Error rates
- Batch sizes
- Real-time request monitoring

## Prometheus Configuration

To add your microservices to Prometheus scraping, edit `docker/prometheus/prometheus.yml`:

```yaml
scrape_configs:
  - job_name: 'my-service'
    scrape_interval: 10s
    static_configs:
      - targets: ['my-service:port']
        labels:
          service: 'my-service'
          environment: 'production'
```

## Environment Variables

- `PORT` - Server port (default: 8080)
- `telemetry-service_URL` - Base URL for client libraries

## Development

### Project Structure

```
telemetry-service/
├── cmd/
│   └── main.go                    # Application entry point
├── internal/
│   ├── logger/
│   │   └── logger.go              # Zap logger implementation
│   ├── metrics/
│   │   └── metrics.go             # Prometheus metrics
│   └── handler/
│       └── handler.go             # HTTP handlers
├── config/
│   └── config.yaml                # Configuration file
├── docker/
│   ├── prometheus/
│   │   └── prometheus.yml         # Prometheus config
│   └── grafana/
│       ├── provisioning/          # Grafana provisioning
│       └── dashboards/            # Grafana dashboards
├── examples/
│   ├── nodejs-client.js           # Node.js client example
│   └── php-client.php             # PHP client example
├── Dockerfile
├── docker-compose.yml
└── go.mod
```

### Run Tests

```bash
go test ./... -v
```

### Build Docker Image

```bash
docker build -t telemetry-service:latest .
```

## Use Cases

### Microservices Logging
Collect logs from all your microservices in one centralized location, making debugging and monitoring easier.

### Distributed Tracing
Use `trace_id` and `span_id` to track requests across multiple services.

### Performance Monitoring
Track request durations, error rates, and throughput across all services.

### Custom Metrics
Record business metrics like user signups, orders placed, revenue, etc.

## Best Practices

1. **Use Structured Logging**: Always include relevant context in the `fields` parameter
2. **Add Trace IDs**: For distributed systems, always include trace and span IDs
3. **Batch Logs**: Use batch endpoints for high-volume logging to reduce network overhead
4. **Label Metrics**: Use meaningful labels for better metric organization
5. **Monitor the Telemetry Service**: The service exposes its own metrics for self-monitoring

## Troubleshooting

### Logs not appearing
- Check if the telemetry service is running: `docker-compose ps`
- Verify the service URL is correct in your client
- Check telemetry service logs: `docker-compose logs telemetry-service`

### Metrics not showing in Grafana
- Verify Prometheus is scraping: Visit `http://localhost:9090/targets`
- Check Prometheus configuration in `docker/prometheus/prometheus.yml`
- Ensure your service is sending metrics to the correct endpoint

### Connection refused errors
- Ensure all services are on the same Docker network
- Check firewall settings
- Verify port mappings in docker-compose.yml

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

MIT License

## Support

For issues and questions, please open an issue on the repository.
