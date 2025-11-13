<?php
/**
 * PHP Client for Telemetry Service
 * 
 * Usage:
 * require_once 'TelemetryClient.php';
 * $telemetry = new TelemetryClient('my-php-service');
 * $telemetry->info('Application started', ['version' => '1.0.0']);
 */

class TelemetryClient {
    private $serviceName;
    private $baseUrl;
    private $timeout;

    /**
     * Create a new TelemetryClient instance
     * 
     * @param string $serviceName Name of your service
     * @param string $baseUrl Base URL of telemetry service
     * @param int $timeout Request timeout in seconds
     */
    public function __construct($serviceName, $baseUrl = null, $timeout = 5) {
        $this->serviceName = $serviceName;
        $this->baseUrl = $baseUrl ?: ($_ENV['telemetry-service_URL'] ?? 'http://localhost:8080');
        $this->timeout = $timeout;
    }

    /**
     * Send a log entry to the telemetry service
     * 
     * @param string $level Log level (debug, info, warn, error, fatal)
     * @param string $message Log message
     * @param array $fields Additional fields
     * @param string|null $traceId Optional trace ID
     * @param string|null $spanId Optional span ID
     * @return bool Success status
     */
    public function log($level, $message, $fields = [], $traceId = null, $spanId = null) {
        try {
            $logEntry = [
                'service_name' => $this->serviceName,
                'level' => $level,
                'message' => $message,
                'timestamp' => date('c'),
                'fields' => $fields,
            ];

            if ($traceId !== null) {
                $logEntry['trace_id'] = $traceId;
            }

            if ($spanId !== null) {
                $logEntry['span_id'] = $spanId;
            }

            return $this->sendRequest('/api/v1/logs', $logEntry);
        } catch (Exception $e) {
            error_log('Failed to send log to telemetry service: ' . $e->getMessage());
            return false;
        }
    }

    /**
     * Send multiple log entries in batch
     * 
     * @param array $logs Array of log entries
     * @return bool Success status
     */
    public function logBatch($logs) {
        try {
            $entries = array_map(function($log) {
                return [
                    'service_name' => $this->serviceName,
                    'level' => $log['level'] ?? 'info',
                    'message' => $log['message'],
                    'timestamp' => $log['timestamp'] ?? date('c'),
                    'fields' => $log['fields'] ?? [],
                    'trace_id' => $log['trace_id'] ?? null,
                    'span_id' => $log['span_id'] ?? null,
                ];
            }, $logs);

            return $this->sendRequest('/api/v1/logs/batch', $entries);
        } catch (Exception $e) {
            error_log('Failed to send log batch to telemetry service: ' . $e->getMessage());
            return false;
        }
    }

    /**
     * Send a metric to the telemetry service
     * 
     * @param string $metricName Name of the metric
     * @param string $metricType Type of metric (counter, gauge, histogram)
     * @param float $value Metric value
     * @param array $labels Additional labels
     * @return bool Success status
     */
    public function recordMetric($metricName, $metricType, $value, $labels = []) {
        try {
            $metricEntry = [
                'service_name' => $this->serviceName,
                'metric_name' => $metricName,
                'metric_type' => $metricType,
                'value' => (float) $value,
                'labels' => $labels,
            ];

            return $this->sendRequest('/api/v1/metrics', $metricEntry);
        } catch (Exception $e) {
            error_log('Failed to send metric to telemetry service: ' . $e->getMessage());
            return false;
        }
    }

    // Convenience methods for logging
    public function debug($message, $fields = []) {
        return $this->log('debug', $message, $fields);
    }

    public function info($message, $fields = []) {
        return $this->log('info', $message, $fields);
    }

    public function warn($message, $fields = []) {
        return $this->log('warn', $message, $fields);
    }

    public function error($message, $fields = []) {
        return $this->log('error', $message, $fields);
    }

    public function fatal($message, $fields = []) {
        return $this->log('fatal', $message, $fields);
    }

    // Convenience methods for metrics
    public function incrementCounter($metricName, $value = 1, $labels = []) {
        return $this->recordMetric($metricName, 'counter', $value, $labels);
    }

    public function setGauge($metricName, $value, $labels = []) {
        return $this->recordMetric($metricName, 'gauge', $value, $labels);
    }

    public function recordHistogram($metricName, $value, $labels = []) {
        return $this->recordMetric($metricName, 'histogram', $value, $labels);
    }

    /**
     * Send HTTP request to telemetry service
     * 
     * @param string $endpoint API endpoint
     * @param array $data Request data
     * @return bool Success status
     */
    private function sendRequest($endpoint, $data) {
        $url = rtrim($this->baseUrl, '/') . $endpoint;
        $payload = json_encode($data);

        $ch = curl_init($url);
        curl_setopt_array($ch, [
            CURLOPT_POST => true,
            CURLOPT_POSTFIELDS => $payload,
            CURLOPT_RETURNTRANSFER => true,
            CURLOPT_TIMEOUT => $this->timeout,
            CURLOPT_HTTPHEADER => [
                'Content-Type: application/json',
                'Content-Length: ' . strlen($payload),
            ],
        ]);

        $response = curl_exec($ch);
        $httpCode = curl_getinfo($ch, CURLINFO_HTTP_CODE);
        $error = curl_error($ch);
        curl_close($ch);

        if ($error) {
            throw new Exception('cURL error: ' . $error);
        }

        if ($httpCode < 200 || $httpCode >= 300) {
            throw new Exception('HTTP error: ' . $httpCode . ' - ' . $response);
        }

        return true;
    }
}

// Usage Example
if (php_sapi_name() === 'cli') {
    $telemetry = new TelemetryClient('my-php-service');

    // Send logs
    $telemetry->info('Application started', ['version' => '1.0.0', 'php_version' => PHP_VERSION]);
    $telemetry->warn('High memory usage detected', ['memory_mb' => 256]);
    $telemetry->error('Database connection failed', ['error' => 'timeout', 'retry_count' => 3]);

    // Send logs in batch
    $telemetry->logBatch([
        ['level' => 'info', 'message' => 'User logged in', 'fields' => ['user_id' => 123]],
        ['level' => 'info', 'message' => 'Order created', 'fields' => ['order_id' => 456]],
    ]);

    // Send metrics
    $telemetry->incrementCounter('http_requests_total', 1, ['method' => 'GET', 'path' => '/api/users']);
    $telemetry->setGauge('active_sessions', 42);
    $telemetry->recordHistogram('request_duration_seconds', 0.234, ['endpoint' => '/api/orders']);

    echo "Telemetry data sent successfully!\n";
}
?>
