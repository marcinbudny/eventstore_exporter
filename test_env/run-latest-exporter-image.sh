docker pull marcinbudny/eventstore_exporter:latest

docker run -it --rm -p 9448:9448 \
    -e EVENTSTORE_URL=http://host.docker.internal:2113 \
    -e CLUSTER_MODE=single \
    -e EVENTSTORE_USER=admin \
    -e EVENTSTORE_PASSWORD=changeit \
    -e ENABLE_PARKED_MESSAGES_STATS=True \
    marcinbudny/eventstore_exporter