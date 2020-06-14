apk add curl 
sleep 10

# datasource
curl -X POST http://grafana:3000/api/datasources -H "Content-Type: application/json" -d '{"name": "Prometheus", "type": "prometheus", "url": "http://prometheus:9090", "access": "proxy"}'

# dashboard
json=$(curl -s https://grafana.com/api/dashboards/7673/revisions/3/download)
echo "{\"dashboard\":$json,\"overwrite\":true, \"inputs\":[{\"name\":\"DS_PROMETHEUS\",\"type\":\"datasource\", \"pluginId\":\"prometheus\",\"value\":\"Prometheus\"}]}" > req.json
curl -X POST http://grafana:3000/api/dashboards/import -H "Content-Type: application/json"  -d @req.json