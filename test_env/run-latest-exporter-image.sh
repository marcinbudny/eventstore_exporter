docker pull marcinbudny/eventstore_exporter:latest

docker run -it --rm -p 9448:9448 \
    --add-host=host.docker.internal:host-gateway \
    -e EVENTSTORE_URL=http://host.docker.internal:2113 \
    -e EVENTSTORE_USER=admin \
    -e EVENTSTORE_PASSWORD=changeit \
    -e ENABLE_PARKED_MESSAGES_STATS=True \
    marcinbudny/eventstore_exporter