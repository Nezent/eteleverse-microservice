// Node.js/JavaScript Client for Telemetry Service
// Installation: npm install axios

const axios = require('axios');

const telemetry-service_URL = process.env.telemetry-service_URL || 'http://localhost:8080';

class TelemetryClient {
  constructor(serviceName) {
    this.serviceName = serviceName;
    this.client = axios.create({
      baseURL: telemetry-service_URL,
      timeout: 5000,
      headers: {
        'Content-Type': 'application/json',
      },
    });
  }

  /**
   * Send a log entry to the telemetry service
   * @param {string} level - Log level (debug, info, warn, error, fatal)
   * @param {string} message - Log message
   * @param {object} fields - Additional fields
   * @param {string} traceId - Optional trace ID
   * @param {string} spanId - Optional span ID
   */
  async log(level, message, fields = {}, traceId = null, spanId = null) {
    try {
      const logEntry = {
        service_name: this.serviceName,
        level,
        message,
        timestamp: new Date().toISOString(),
        fields,
        trace_id: traceId,
        span_id: spanId,
      };

      await this.client.post('/api/v1/logs', logEntry);
    } catch (error) {
      console.error('Failed to send log to telemetry service:', error.message);
    }
  }

  /**
   * Send multiple log entries in batch
   * @param {array} logs - Array of log entries
   */
  async logBatch(logs) {
    try {
      const entries = logs.map(log => ({
        service_name: this.serviceName,
        level: log.level || 'info',
        message: log.message,
        timestamp: log.timestamp || new Date().toISOString(),
        fields: log.fields || {},
        trace_id: log.traceId,
        span_id: log.spanId,
      }));

      await this.client.post('/api/v1/logs/batch', entries);
    } catch (error) {
      console.error('Failed to send log batch to telemetry service:', error.message);
    }
  }

  /**
   * Send a metric to the telemetry service
   * @param {string} metricName - Name of the metric
   * @param {string} metricType - Type of metric (counter, gauge, histogram)
   * @param {number} value - Metric value
   * @param {object} labels - Additional labels
   */
  async recordMetric(metricName, metricType, value, labels = {}) {
    try {
      const metricEntry = {
        service_name: this.serviceName,
        metric_name: metricName,
        metric_type: metricType,
        value,
        labels,
      };

      await this.client.post('/api/v1/metrics', metricEntry);
    } catch (error) {
      console.error('Failed to send metric to telemetry service:', error.message);
    }
  }

  // Convenience methods
  debug(message, fields = {}) {
    return this.log('debug', message, fields);
  }

  info(message, fields = {}) {
    return this.log('info', message, fields);
  }

  warn(message, fields = {}) {
    return this.log('warn', message, fields);
  }

  error(message, fields = {}) {
    return this.log('error', message, fields);
  }

  fatal(message, fields = {}) {
    return this.log('fatal', message, fields);
  }

  // Metric convenience methods
  incrementCounter(metricName, value = 1, labels = {}) {
    return this.recordMetric(metricName, 'counter', value, labels);
  }

  setGauge(metricName, value, labels = {}) {
    return this.recordMetric(metricName, 'gauge', value, labels);
  }

  recordHistogram(metricName, value, labels = {}) {
    return this.recordMetric(metricName, 'histogram', value, labels);
  }
}

// Usage Example
async function example() {
  const telemetry = new TelemetryClient('my-nodejs-service');

  // Send logs
  await telemetry.info('Application started', { version: '1.0.0', port: 3000 });
  await telemetry.warn('High memory usage detected', { memory_mb: 512 });
  await telemetry.error('Database connection failed', { error: 'timeout', retry_count: 3 });

  // Send logs in batch
  await telemetry.logBatch([
    { level: 'info', message: 'User logged in', fields: { user_id: 123 } },
    { level: 'info', message: 'Order created', fields: { order_id: 456 } },
  ]);

  // Send metrics
  await telemetry.incrementCounter('http_requests_total', 1, { method: 'GET', path: '/api/users' });
  await telemetry.setGauge('active_connections', 42);
  await telemetry.recordHistogram('request_duration_seconds', 0.234, { endpoint: '/api/orders' });
}

module.exports = TelemetryClient;

// If running this file directly
if (require.main === module) {
  example().catch(console.error);
}
