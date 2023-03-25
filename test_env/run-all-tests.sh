set -e

echo "Shutting down any running environments"
docker-compose -f 21.10-single/docker-compose.yml down
docker-compose -f 21.10-cluster/docker-compose.yml down
docker-compose -f 22.10-single/docker-compose.yml down
docker-compose -f 22.10-cluster/docker-compose.yml down

echo "Running tests on ES 21.10 - SINGLE"
EVENTSTORE_RUN_PROJECTIONS=None docker-compose -f 21.10-single/docker-compose.yml up -d 
sleep 5
go test ../... -count=1 -v
docker-compose -f 21.10-single/docker-compose.yml down

echo "Running tests on ES 21.10 - SINGLE - WITH PROJECTIONS"
EVENTSTORE_RUN_PROJECTIONS=All docker-compose -f 21.10-single/docker-compose.yml up -d
sleep 5
go test ../... -count=1 -v
docker-compose -f 21.10-single/docker-compose.yml down

echo "Running tests on ES 21.10 - CLUSTER"
docker-compose -f 21.10-cluster/docker-compose.yml up -d
sleep 15
go test ../... -count=1 -v
docker-compose -f 21.10-cluster/docker-compose.yml down

echo "Running tests on ES 22.10 - SINGLE"
EVENTSTORE_RUN_PROJECTIONS=None docker-compose -f 22.10-single/docker-compose.yml up -d
sleep 5
go test ../... -count=1 -v
docker-compose -f 22.10-single/docker-compose.yml down

echo "Running tests on ES 22.10 - SINGLE - WITH PROJECTIONS"
EVENTSTORE_RUN_PROJECTIONS=All docker-compose -f 22.10-single/docker-compose.yml up -d
sleep 5
go test ../... -count=1 -v
docker-compose -f 22.10-single/docker-compose.yml down

echo "Running tests on ES 22.10 - CLUSTER"
docker-compose -f 22.10-cluster/docker-compose.yml up -d
sleep 15
go test ../... -count=1 -v
docker-compose -f 22.10-cluster/docker-compose.yml down