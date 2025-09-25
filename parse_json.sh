#!/bin/bash

# Simple JSON parsing functions that don't require jq

# Extract boolean value from JSON
get_json_bool() {
    local json="$1"
    local key="$2"
    echo "$json" | grep -o "\"$key\":[^,}]*" | cut -d: -f2 | tr -d ' "' | head -1
}

# Extract string value from JSON
get_json_string() {
    local json="$1"
    local key="$2"
    echo "$json" | grep -o "\"$key\":\"[^\"]*\"" | cut -d'"' -f4 | head -1
}

# Pretty print JSON (basic formatting)
pretty_json() {
    local json="$1"
    echo "$json" | sed 's/,/,\n  /g' | sed 's/{/{\n  /' | sed 's/}/\n}/'
}

# Check if response indicates success
is_success() {
    local json="$1"
    local success=$(get_json_bool "$json" "success")
    [ "$success" = "true" ]
}

# Get error message from JSON
get_error_message() {
    local json="$1"
    local error=$(get_json_string "$json" "error")
    local message=$(get_json_string "$json" "message")
    
    if [ -n "$error" ]; then
        echo "$error"
    elif [ -n "$message" ]; then
        echo "$message"
    else
        echo "Unknown error"
    fi
}
