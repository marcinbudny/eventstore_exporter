# EventStore Prometheus exporter

[![Go Report Card](https://goreportcard.com/badge/github.com/marcinbudny/eventstore_exporter)](https://goreportcard.com/report/github.com/marcinbudny/eventstore_exporter)
[![CI](https://github.com/marcinbudny/eventstore_exporter/actions/workflows/main.yml/badge.svg)](https://github.com/marcinbudny/eventstore_exporter/actions/workflows/main.yml)

EventStoreDB (https://eventstore.com/eventstoredb/) metrics Prometheus exporter.

## Installation

### From source

You need to have a Go 1.17+ environment configured.

```bash
go get github.com/marcinbudny/eventstore_exporter

eventstore_exporter \
    --eventstore-url=https://localhost:2113 \
    --eventstore-user=admin \
    --eventstore-password=changeit \
    --cluster-mode=single \
    --insecure-skip-verify \
    --enable-parked-messages-stats
```

### Using Docker

```bash
docker run -d -p 9448:9448 \
    -e EVENTSTORE_URL=https://my-eventstore:2113 \
    -e CLUSTER_MODE=single \
    -e EVENTSTORE_USER=admin \
    -e EVENTSTORE_PASSWORD=changeit \
    marcinbudny/eventstore_exporter
```

### Supported versions

- 5.0
- 20.10 LTS
- 21.6

## Configuration

The exporter can be configured with command line arguments, environment variables and a configuration file. For the details on how to format the configuration file, visit [namsral/flag](https://github.com/namsral/flag) repo.

| Flag                           | ENV variable                 | Default               | Meaning                                                                                                                                                                                                                                                                                                                                                                   |
| ------------------------------ | ---------------------------- | --------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| --config                       |                              |                       | Path to config file (optional)                                                                                                                                                                                                                                                                                                                                            |
| --eventstore-url               | EVENTSTORE_URL               | http://localhost:2113 | Eventstore HTTP endpoint                                                                                                                                                                                                                                                                                                                                                  |
| --eventstore-user              | EVENTSTORE_USER              | (empty)               | Eventstore user (if not specified, basic auth is not used)                                                                                                                                                                                                                                                                                                                |
| --eventstore-password          | EVENTSTORE_PASSWORD          | (empty)               | Eventstore password (if not specified, basic auth is not used)                                                                                                                                                                                                                                                                                                            |
| --cluster-mode                 | CLUSTER_MODE                 | cluster               | Set to 'single' when monitoring a single node instance, set to 'cluster' when monitoring a cluster node. This settings decides whether gossip stats endpoint is queried. **Note:** Starting with ES 21.2, the gossip endpoint responds to queries even for single node installation. So if using this version, you can ignore this setting and leave it at default value. |
| --port                         | PORT                         | 9448                  | Port to expose scrape endpoint on                                                                                                                                                                                                                                                                                                                                         |
| --timeout                      | TIMEOUT                      | 10s                   | Timeout when calling EventStore                                                                                                                                                                                                                                                                                                                                           |
| --verbose                      | VERBOSE                      | false                 | Enable verbose logging                                                                                                                                                                                                                                                                                                                                                    |
| --insecure-skip-verify         | INSECURE_SKIP_VERIFY         | false                 | Skip TLS certificate verification for EventStore HTTP client                                                                                                                                                                                                                                                                                                              |
| --enable-parked-messages-stats | ENABLE_PARKED_MESSAGES_STATS | false                 | Enable parked messages stats scraping. **Note:** for ES 20.10+ you need to enable Atom Pub over HTTP in EventStoreDB to get subscriptions stats. For ES 21.2+, number of parked messages can be extracted without enabling AtomPub, but not the age of the oldest message.                                                                                                |

Sample configuration file

```
eventstore-url=http://localhost:2113
port=8888
verbose
```

To run with configuration file:

```bash
./eventstore_exporter --config my_config_file
```

## Grafana dashboard

Can be found [here](https://grafana.com/dashboards/7673)

![EventStore Grafana dashboard](dashboard.png)

## Exported metrics

Let me know if there is a metric you would like to be added.

```
# HELP eventstore_cluster_member_alive If 1, cluster member is alive, as seen from current cluster member
# TYPE eventstore_cluster_member_alive gauge
eventstore_cluster_member_alive{member="172.16.1.11:2113"} 1
# HELP eventstore_cluster_member_is_clone If 1, current cluster member is a clone
# TYPE eventstore_cluster_member_is_clone gauge
eventstore_cluster_member_is_clone 1
# HELP eventstore_cluster_member_is_follower If 1, current cluster member is a follower (only versions >= 20.6)
# TYPE eventstore_cluster_member_is_follower gauge
eventstore_cluster_member_is_follower 0
# HELP eventstore_cluster_member_is_leader If 1, current cluster member is the leader (only versions >= 20.6)
# TYPE eventstore_cluster_member_is_leader gauge
eventstore_cluster_member_is_leader 0
# HELP eventstore_cluster_member_is_master If 1, current cluster member is the master (only versions < 20.6)
# TYPE eventstore_cluster_member_is_master gauge
eventstore_cluster_member_is_master 0
# HELP eventstore_cluster_member_is_readonly_replica If 1, current cluster member is a readonly replica (only versions >= 20.6)
# TYPE eventstore_cluster_member_is_readonly_replica gauge
eventstore_cluster_member_is_readonly_replica 0
# HELP eventstore_cluster_member_is_slave If 1, current cluster member is a slave (only versions < 20.6)
# TYPE eventstore_cluster_member_is_slave gauge
eventstore_cluster_member_is_slave 0
# HELP eventstore_disk_io_read_bytes Total number of disk IO read bytes
# TYPE eventstore_disk_io_read_bytes gauge
eventstore_disk_io_read_bytes 20480
# HELP eventstore_disk_io_read_ops Total number of disk IO read operations
# TYPE eventstore_disk_io_read_ops gauge
eventstore_disk_io_read_ops 2814
# HELP eventstore_disk_io_write_ops Total number of disk IO write operations
# TYPE eventstore_disk_io_write_ops gauge
eventstore_disk_io_write_ops 4421
# HELP eventstore_disk_io_written_bytes Total number of disk IO written bytes
# TYPE eventstore_disk_io_written_bytes gauge
eventstore_disk_io_written_bytes 2.6918912e+08
# HELP eventstore_drive_available_bytes Drive available bytes
# TYPE eventstore_drive_available_bytes gauge
eventstore_drive_available_bytes{drive="/var/lib/eventstore"} 5.6815230976e+10
# HELP eventstore_drive_total_bytes Drive total size in bytes
# TYPE eventstore_drive_total_bytes gauge
eventstore_drive_total_bytes{drive="/var/lib/eventstore"} 6.2725787648e+10
# HELP eventstore_process_cpu Process CPU usage, 0 - number of cores
# TYPE eventstore_process_cpu gauge
eventstore_process_cpu 0.08
# HELP eventstore_process_cpu_scaled Process CPU usage scaled to number of cores, 0 - 1, 1 = full load on all cores (available only on versions < 20.6)
# TYPE eventstore_process_cpu_scaled gauge
eventstore_process_cpu_scaled 0
# HELP eventstore_process_memory_bytes Process memory usage, as reported by EventStore
# TYPE eventstore_process_memory_bytes gauge
eventstore_process_memory_bytes 1.19267328e+08
# HELP eventstore_projection_events_processed_after_restart_total Projection event processed count after restart
# TYPE eventstore_projection_events_processed_after_restart_total counter
eventstore_projection_events_processed_after_restart_total{projection="$by_event_type"} 0
# HELP eventstore_projection_progress Projection progress 0 - 1, where 1 = projection progress at 100%
# TYPE eventstore_projection_progress gauge
eventstore_projection_progress{projection="$by_event_type"} 1
# HELP eventstore_projection_running If 1, projection is in 'Running' state
# TYPE eventstore_projection_running gauge
eventstore_projection_running{projection="$by_event_type"} 1
# HELP eventstore_queue_items_processed_total Total number items processed by queue
# TYPE eventstore_queue_items_processed_total counter
eventstore_queue_items_processed_total{queue="index Committer"} 54
# HELP eventstore_queue_length Queue length
# TYPE eventstore_queue_length gauge
eventstore_queue_length{queue="index Committer"} 0
# HELP eventstore_subscription_connections Number of connections to subscription
# TYPE eventstore_subscription_connections gauge
eventstore_subscription_connections{event_stream_id="test-stream",group_name="group1"} 0
# HELP eventstore_subscription_items_processed_total Total items processed by subscription
# TYPE eventstore_subscription_items_processed_total counter
eventstore_subscription_items_processed_total{event_stream_id="test-stream",group_name="group1"} 24
# HELP eventstore_subscription_last_known_event_number Last known event number in subscription
# TYPE eventstore_subscription_last_known_event_number gauge
eventstore_subscription_last_known_event_number{event_stream_id="test-stream",group_name="group1"} 23
# HELP eventstore_subscription_last_processed_event_number Last event number processed by subscription
# TYPE eventstore_subscription_last_processed_event_number gauge
eventstore_subscription_last_processed_event_number{event_stream_id="test-stream",group_name="group1"} 19
# HELP eventstore_subscription_messages_in_flight Number of messages in flight for subscription
# TYPE eventstore_subscription_messages_in_flight gauge
eventstore_subscription_messages_in_flight{event_stream_id="test-stream",group_name="group1"} 0
# HELP eventstore_subscription_oldest_parked_message_age_seconds Oldest parked message age for subscription in seconds
# TYPE eventstore_subscription_oldest_parked_message_age_seconds gauge
eventstore_subscription_oldest_parked_message_age_seconds{event_stream_id="test-stream",group_name="group1"} 33
# HELP eventstore_subscription_parked_messages Number of parked messages for subscription
# TYPE eventstore_subscription_parked_messages gauge
eventstore_subscription_parked_messages{event_stream_id="test-stream",group_name="group1"} 1
# HELP eventstore_tcp_connections Current number of TCP connections
# TYPE eventstore_tcp_connections gauge
eventstore_tcp_connections 1
# HELP eventstore_tcp_received_bytes TCP received bytes
# TYPE eventstore_tcp_received_bytes gauge
eventstore_tcp_received_bytes 17237
# HELP eventstore_tcp_sent_bytes TCP sent bytes
# TYPE eventstore_tcp_sent_bytes gauge
eventstore_tcp_sent_bytes 3423
# HELP eventstore_up Whether the EventStore scrape was successful
# TYPE eventstore_up gauge
eventstore_up 1
```

## Changelog

### 0.10.3

- FIX: fixed parked message count based on group info (when atom pub is disabled)

### 0.10.2

- FIX: ability to properly load config files

### 0.10.1

- FIX: fixed memory leak occurring when calls to ESDB fail

### 0.10.0

- BREAKING: The `is_master` and `is_slave` metrics are now only exported for ES version 5, while `is_leader`, `is_follower`, `is_readonly_replica` for ES versions 20.6+
- BREAKING: The `cpu_scaled` metric is only available for ES version 5
- FEATURE: It is now possible to get subscription parked message count even if Atom Pub over HTTP is disabled (requires ES 21.2+)
- FEATURE: Updated Grafana dashboard to include parked message count and oldest parked message age and also adjusted member status presentation to the breaking change - update to dashboard revision 7
- OTHER: Docker image is now based on Go 1.16.3

### 0.9.0

- FEATURE: parked message metrics (note: enable them with `--enable-parked-messages-stats` flag)

### 0.8.1

- FIX: in some cases /stats endpoint scrape will fail due to missing Accept header - see #13

### 0.8.0

- FEATURE: support for ES 20.6 - see #12
- FEATURE: option to ignore invalid certificates on HTTPS connection - see #11
- FIX: moved to stateless metrics, that should fix the problem (without a workaround) of zombie metrics after projection / member / subscription has been removed

### 0.7.0

- FIX: for items of variable count (queues, drives, projections, subscriptions, members) the exporter should not return items that have been removed (see #7)

### 0.6.0

- FEATURE: add HTTP Basic auth to support EventStore 5.0.2+ (see #6)
- FIX: when status code of http call does not indicate success, the exporter will now log a message and it won't report metrics

### 0.5.0

- FEATURE: new metrics for detecting cluster node status: `eventstore_cluster_member_is_slave` and `eventstore_cluster_member_is_clone`

### 0.4.0

- FEATURE: persistent subscription metrics

### 0.3.0

- FEATURE: added drive metrics

### 0.2.0

- FIX: missing `eventstore_process_memory_bytes` metric
- BREAKING: `eventstore_projection_events_processed_after_restart` metric renamed to `eventstore_projection_events_processed_after_restart_total` to comply with Prometheus metric naming rules

### 0.1.1

- experimenting with dockerhub tags

### 0.1.0

- Initial version
