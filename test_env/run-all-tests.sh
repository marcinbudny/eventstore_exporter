set -e

versions=("22.10" "23.10" "24.2")

for version in "${versions[@]}"
do
    echo "Shutting down any running $version environments"
    docker-compose -f "$version"-single/docker-compose.yml down
    docker-compose -f "$version"-cluster/docker-compose.yml down
done

for version in "${versions[@]}"
do
    echo "Running tests on ES $version - SINGLE"
    EVENTSTORE_RUN_PROJECTIONS=None docker-compose -f "$version"-single/docker-compose.yml up -d
    sleep 5
    go test ../... -count=1 -v
    docker-compose -f "$version"-single/docker-compose.yml down

    echo "Running tests on ES $version - SINGLE - WITH PROJECTIONS"
    EVENTSTORE_RUN_PROJECTIONS=All docker-compose -f "$version"-single/docker-compose.yml up -d
    sleep 5
    go test ../... -count=1 -v
    docker-compose -f "$version"-single/docker-compose.yml down

    echo "Running tests on ES $version - CLUSTER"
    docker-compose -f "$version"-cluster/docker-compose.yml up -d
    sleep 15
    go test ../... -count=1 -v
    docker-compose -f "$version"-cluster/docker-compose.yml down
done

echo "All tests passed"