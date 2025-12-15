#!/usr/bin/env bash

declare -a test_results
declare -a test_names
declare -a test_attempts

MAX_ATTEMPTS=3

versions=("23.10" "24.10" "25.1")

for version in "${versions[@]}"
do
    echo "Shutting down any running $version environments"
    docker compose -f "$version"-single/docker-compose.yml down
    docker compose -f "$version"-cluster/docker-compose.yml down
done

test_index=0

run_test_with_retry() {
    local test_name="$1"
    local attempt=1
    local success=false
    
    while [ $attempt -le $MAX_ATTEMPTS ] && [ "$success" = "false" ]; do
        echo "Running $test_name (Attempt $attempt/$MAX_ATTEMPTS)"
        go test ../... -count=1
        test_exit_code=$?
        
        if [ $test_exit_code -eq 0 ]; then
            success=true
            echo "✅ $test_name: SUCCESS (Attempt $attempt)"
        else
            echo "❌ $test_name: FAILED (Attempt $attempt)"
            if [ $attempt -lt $MAX_ATTEMPTS ]; then
                echo "Retrying..."
            fi
            attempt=$((attempt + 1))
        fi
    done
    
    test_results[$test_index]=$test_exit_code
    test_names[$test_index]="$test_name"
    test_attempts[$test_index]=$attempt
    test_index=$((test_index + 1))
    
    return $test_exit_code
}

for version in "${versions[@]}"
do
    echo "Setting up ES $version - SINGLE"
    EVENTSTORE_RUN_PROJECTIONS=None docker compose -f "$version"-single/docker-compose.yml up -d
    sleep 10
    run_test_with_retry "ES $version - SINGLE"
    docker compose -f "$version"-single/docker-compose.yml down

    echo "Setting up ES $version - SINGLE - WITH PROJECTIONS"
    EVENTSTORE_RUN_PROJECTIONS=All docker compose -f "$version"-single/docker-compose.yml up -d
    sleep 10
    run_test_with_retry "ES $version - SINGLE - WITH PROJECTIONS"
    docker compose -f "$version"-single/docker-compose.yml down

    echo "Setting up ES $version - CLUSTER"
    docker compose -f "$version"-cluster/docker-compose.yml up -d
    sleep 15
    run_test_with_retry "ES $version - CLUSTER"
    docker compose -f "$version"-cluster/docker-compose.yml down
done

echo ""
echo "=================== TEST SUMMARY ==================="
overall_success=true
for i in $(seq 0 $((test_index - 1))); do
    if [ ${test_results[$i]} -eq 0 ]; then
        echo "✅ ${test_names[$i]}: SUCCESS (Attempts: ${test_attempts[$i]})"
    else
        echo "❌ ${test_names[$i]}: FAILED after ${test_attempts[$i]} attempts"
        overall_success=false
    fi
done
echo "===================================================="

if $overall_success; then
    echo "All tests passed successfully!"
    exit 0
else
    echo "Some tests failed after multiple attempts. See summary above for details."
    exit 1
fi