set -e

echo "Shutting down any running environments"
docker-compose -f 20.10-single/docker-compose.yml down
docker-compose -f 20.10-cluster/docker-compose.yml down
docker-compose -f 21.10-single/docker-compose.yml down
docker-compose -f 21.10-cluster/docker-compose.yml down
docker-compose -f 22.10-single/docker-compose.yml down
docker-compose -f 22.10-cluster/docker-compose.yml down

echo "Running tests on ES 20.10 - SINGLE"
EVENTSTORE_RUN_PROJECTIONS=None docker-compose -f 20.10-single/docker-compose.yml up -d 
sleep 5
TEST_EVENTSTORE_URL=http://localhost:2113 go test ../... -count=1 -v
docker-compose -f 20.10-single/docker-compose.yml down

echo "Running tests on ES 20.10 - SINGLE - WITH PROJECTIONS"
EVENTSTORE_RUN_PROJECTIONS=All docker-compose -f 20.10-single/docker-compose.yml up -d
sleep 5
TEST_EVENTSTORE_URL=http://localhost:2113 TEST_PROJECTION_METRICS=1 go test ../... -count=1 -v
docker-compose -f 20.10-single/docker-compose.yml down

echo "Running tests on ES 20.10 - CLUSTER"
docker-compose -f 20.10-cluster/docker-compose.yml up -d
sleep 15
TEST_EVENTSTORE_URL=https://localhost:2113 TEST_CLUSTER_MODE=cluster go test ../... -count=1 -v
docker-compose -f 20.10-cluster/docker-compose.yml down

echo "Running tests on ES 21.10 - SINGLE"
EVENTSTORE_RUN_PROJECTIONS=None docker-compose -f 21.10-single/docker-compose.yml up -d 
sleep 5
TEST_EVENTSTORE_URL=http://localhost:2113 go test ../... -count=1 -v
docker-compose -f 21.10-single/docker-compose.yml down

echo "Running tests on ES 21.10 - SINGLE - WITH PROJECTIONS"
EVENTSTORE_RUN_PROJECTIONS=All docker-compose -f 21.10-single/docker-compose.yml up -d
sleep 5
TEST_EVENTSTORE_URL=http://localhost:2113 TEST_PROJECTION_METRICS=1 go test ../... -count=1 -v
docker-compose -f 21.10-single/docker-compose.yml down

echo "Running tests on ES 21.10 - CLUSTER"
docker-compose -f 21.10-cluster/docker-compose.yml up -d
sleep 15
TEST_EVENTSTORE_URL=https://localhost:2113 TEST_CLUSTER_MODE=cluster go test ../... -count=1 -v
docker-compose -f 21.10-cluster/docker-compose.yml down

echo "Running tests on ES 22.10 - SINGLE"
EVENTSTORE_RUN_PROJECTIONS=None docker-compose -f 22.10-single/docker-compose.yml up -d
sleep 5
TEST_EVENTSTORE_URL=http://localhost:2113 go test ../... -count=1 -v
docker-compose -f 22.10-single/docker-compose.yml down

echo "Running tests on ES 22.10 - SINGLE - WITH PROJECTIONS"
EVENTSTORE_RUN_PROJECTIONS=All docker-compose -f 22.10-single/docker-compose.yml up -d
sleep 5
TEST_EVENTSTORE_URL=http://localhost:2113 TEST_PROJECTION_METRICS=1 go test ../... -count=1 -v
docker-compose -f 22.10-single/docker-compose.yml down

echo "Running tests on ES 22.10 - CLUSTER"
docker-compose -f 22.10-cluster/docker-compose.yml up -d
sleep 15
TEST_EVENTSTORE_URL=https://localhost:2113 TEST_CLUSTER_MODE=cluster go test ../... -count=1 -v
docker-compose -f 22.10-cluster/docker-compose.yml down