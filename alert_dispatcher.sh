#!/bin/bash

#############################################
# Prometheus Alert Logger Script
# 
# This script queries Prometheus alerts API
# and logs alerts locally to simulate 
# alert dispatch functionality
#############################################

# Configuration
PROMETHEUS_URL="${PROMETHEUS_URL:-http://localhost:9090}"
LOG_DIR="./prometheus-alerts"
LOG_FILE="${LOG_DIR}/alerts-$(date +%Y%m%d).log"
ALERT_LOG="${LOG_DIR}/alert-dispatch.log"
CHECK_INTERVAL="${CHECK_INTERVAL:-30}"  # seconds

# Colors for output
RED='\033[0;31m'
YELLOW='\033[1;33m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Create log directory if it doesn't exist
mkdir -p "$LOG_DIR"

# Function to log messages
log() {
    local level=$1
    shift
    local message="$@"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    echo "[$timestamp] [$level] $message" | tee -a "$LOG_FILE"
}

# Function to log alert dispatch
dispatch_alert() {
    local alert_json="$1"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    
    # Parse alert details
    local alert_name=$(echo "$alert_json" | jq -r '.labels.alertname // "Unknown"')
    local severity=$(echo "$alert_json" | jq -r '.labels.severity // "none"')
    local state=$(echo "$alert_json" | jq -r '.state')
    local instance=$(echo "$alert_json" | jq -r '.labels.instance // "N/A"')
    local job=$(echo "$alert_json" | jq -r '.labels.job // "N/A"')
    local description=$(echo "$alert_json" | jq -r '.annotations.description // .annotations.summary // "No description"')
    
    # Create alert dispatch entry
    cat >> "$ALERT_LOG" << EOF
================================================================================
ALERT DISPATCHED: $timestamp
================================================================================
Alert Name:    $alert_name
Severity:      $severity
State:         $state
Instance:      $instance
Job:           $job
Description:   $description
--------------------------------------------------------------------------------
Full Alert Data:
$alert_json
================================================================================

EOF

    # Print to console with color
    case "$severity" in
        critical)
            echo -e "${RED}🚨 CRITICAL ALERT: $alert_name${NC}"
            ;;
        warning)
            echo -e "${YELLOW}⚠️  WARNING ALERT: $alert_name${NC}"
            ;;
        *)
            echo -e "${BLUE}ℹ️  INFO ALERT: $alert_name${NC}"
            ;;
    esac
    
    echo -e "   State: $state | Instance: $instance"
    echo -e "   $description"
    echo ""
}

# Function to fetch and process alerts
fetch_alerts() {
    log "INFO" "Fetching alerts from Prometheus..."
    
    # Query Prometheus alerts API
    local response=$(curl -s "${PROMETHEUS_URL}/api/v1/alerts" 2>/dev/null)
    
    if [ $? -ne 0 ]; then
        log "ERROR" "Failed to connect to Prometheus at $PROMETHEUS_URL"
        return 1
    fi
    
    # Check if response is valid JSON
    if ! echo "$response" | jq empty 2>/dev/null; then
        log "ERROR" "Invalid JSON response from Prometheus"
        return 1
    fi
    
    # Extract alerts
    local alerts=$(echo "$response" | jq -c '.data.alerts[]?' 2>/dev/null)
    
    if [ -z "$alerts" ]; then
        log "INFO" "No alerts found"
        return 0
    fi
    
    # Process each alert
    local alert_count=0
    local firing_count=0
    local pending_count=0
    
    while IFS= read -r alert; do
        if [ -n "$alert" ]; then
            ((alert_count++))
            
            local state=$(echo "$alert" | jq -r '.state')
            
            if [ "$state" = "firing" ]; then
                ((firing_count++))
                dispatch_alert "$alert"
            elif [ "$state" = "pending" ]; then
                ((pending_count++))
            fi
        fi
    done <<< "$alerts"
    
    # Summary
    log "INFO" "Alert Summary - Total: $alert_count, Firing: $firing_count, Pending: $pending_count"
}

# Function to fetch alert rules
fetch_alert_rules() {
    log "INFO" "Fetching alert rules from Prometheus..."
    
    local response=$(curl -s "${PROMETHEUS_URL}/api/v1/rules" 2>/dev/null)
    
    if [ $? -ne 0 ]; then
        log "ERROR" "Failed to fetch alert rules"
        return 1
    fi
    
    # Save rules to file
    local rules_file="${LOG_DIR}/alert-rules-$(date +%Y%m%d-%H%M%S).json"
    echo "$response" | jq '.' > "$rules_file"
    log "INFO" "Alert rules saved to: $rules_file"
}

# Function to check Prometheus health
check_prometheus() {
    local health=$(curl -s "${PROMETHEUS_URL}/-/healthy" 2>/dev/null)
    
    if [ "$health" = "Prometheus Server is Healthy." ] || [ "$health" = "Prometheus is Healthy." ]; then
        echo -e "${GREEN}✓ Prometheus is healthy${NC}"
        return 0
    else
        echo -e "${RED}✗ Prometheus is not responding${NC}"
        return 1
    fi
}

# Function to monitor continuously
monitor_continuously() {
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}Prometheus Alert Monitor Started${NC}"
    echo -e "${GREEN}========================================${NC}"
    echo "Prometheus URL: $PROMETHEUS_URL"
    echo "Log Directory:  $LOG_DIR"
    echo "Check Interval: ${CHECK_INTERVAL}s"
    echo "Press Ctrl+C to stop"
    echo ""
    
    # Check Prometheus health
    check_prometheus || exit 1
    
    log "INFO" "Alert monitoring started"
    
    local iteration=0
    while true; do
        ((iteration++))
        echo -e "${BLUE}[Check #$iteration at $(date '+%H:%M:%S')]${NC}"
        
        fetch_alerts
        
        echo "Next check in ${CHECK_INTERVAL} seconds..."
        echo ""
        
        sleep "$CHECK_INTERVAL"
    done
}

# Function to show help
show_help() {
    cat << EOF
Prometheus Alert Logger Script

Usage: $0 [COMMAND] [OPTIONS]

Commands:
    check           Fetch alerts once and exit
    monitor         Continuously monitor alerts (default)
    rules           Fetch and save alert rules
    health          Check Prometheus health
    tail            Tail the alert dispatch log
    stats           Show alert statistics
    help            Show this help message

Options:
    -u URL          Prometheus URL (default: http://localhost:9090)
    -i INTERVAL     Check interval in seconds (default: 30)
    -l LOG_DIR      Log directory (default: ./prometheus-alerts)

Environment Variables:
    PROMETHEUS_URL      Prometheus server URL
    CHECK_INTERVAL      Interval between checks in seconds

Examples:
    $0 check                                    # Check alerts once
    $0 monitor                                  # Monitor continuously
    $0 monitor -i 60                            # Monitor with 60s interval
    $0 -u http://192.168.1.100:9090 monitor    # Use custom Prometheus URL
    $0 rules                                    # Fetch alert rules
    $0 tail                                     # View alert dispatch log

Log Files:
    Daily Log:      ${LOG_DIR}/alerts-YYYYMMDD.log
    Alert Dispatch: ${LOG_DIR}/alert-dispatch.log
    Alert Rules:    ${LOG_DIR}/alert-rules-*.json

EOF
}

# Function to show statistics
show_stats() {
    echo "📊 Alert Statistics"
    echo "=================="
    echo ""
    
    if [ -f "$ALERT_LOG" ]; then
        local total_dispatched=$(grep -c "ALERT DISPATCHED" "$ALERT_LOG")
        local critical=$(grep -c "Severity:.*critical" "$ALERT_LOG")
        local warning=$(grep -c "Severity:.*warning" "$ALERT_LOG")
        
        echo "Total Alerts Dispatched: $total_dispatched"
        echo "Critical Alerts:         $critical"
        echo "Warning Alerts:          $warning"
        echo ""
        echo "Recent Alerts (last 5):"
        grep "Alert Name:" "$ALERT_LOG" | tail -5
    else
        echo "No alert dispatch log found yet."
    fi
}

# Function to tail the alert log
tail_log() {
    if [ -f "$ALERT_LOG" ]; then
        tail -f "$ALERT_LOG"
    else
        echo "No alert dispatch log found yet."
        echo "Creating log file: $ALERT_LOG"
        touch "$ALERT_LOG"
        echo "Waiting for alerts..."
        tail -f "$ALERT_LOG"
    fi
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -u|--url)
            PROMETHEUS_URL="$2"
            shift 2
            ;;
        -i|--interval)
            CHECK_INTERVAL="$2"
            shift 2
            ;;
        -l|--log-dir)
            LOG_DIR="$2"
            LOG_FILE="${LOG_DIR}/alerts-$(date +%Y%m%d).log"
            ALERT_LOG="${LOG_DIR}/alert-dispatch.log"
            shift 2
            ;;
        check)
            COMMAND="check"
            shift
            ;;
        monitor)
            COMMAND="monitor"
            shift
            ;;
        rules)
            COMMAND="rules"
            shift
            ;;
        health)
            COMMAND="health"
            shift
            ;;
        tail)
            COMMAND="tail"
            shift
            ;;
        stats)
            COMMAND="stats"
            shift
            ;;
        help|--help|-h)
            show_help
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            show_help
            exit 1
            ;;
    esac
done

# Default command
COMMAND="${COMMAND:-monitor}"

# Execute command
case "$COMMAND" in
    check)
        check_prometheus && fetch_alerts
        ;;
    monitor)
        monitor_continuously
        ;;
    rules)
        fetch_alert_rules
        ;;
    health)
        check_prometheus
        ;;
    tail)
        tail_log
        ;;
    stats)
        show_stats
        ;;
    *)
        echo "Unknown command: $COMMAND"
        show_help
        exit 1
        ;;
esac
