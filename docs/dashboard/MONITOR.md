# MONITOR

- If this is not first setup, go to [Reset](#reset)
- Install loki plugin

```bash
docker plugin install grafana/loki-docker-driver:latest --alias loki --grant-all-permissions
```

- <https://hub.docker.com/r/bitnami/node-exporter/tags>
- <https://grafana.com/docs/loki/latest/release-notes/>
- <https://hub.docker.com/r/prom/prometheus/tags>
- <https://github.com/grafana/grafana/blob/main/CHANGELOG.md>

```bash
IMAGE_EXPORTER="bitnami/node-exporter:1.9.1"
IMAGE_LOKI="grafana/loki:3.5"
IMAGE_PROMETHEUS="prom/prometheus:v3.4.2"
IMAGE_GRAFANA="grafana/grafana:12.0.2"

echo '
{
    "log-driver": "loki",
    "log-opts": {
        "loki-url": "http://127.0.0.1:3100/loki/api/v1/push",
        "max-size": "10m",
        "max-file": "10"
    }
}
' >/etc/docker/daemon.json
systemctl restart docker.service

docker run -d \
    --restart always \
    --name exporter \
    -v /:/host:ro,rslave \
    -p 80:80 \
    -p 443:443 \
    -p 3000:3000 \
    -p 3100:3100 \
    $IMAGE_EXPORTER --path.rootfs=/host

echo '
auth_enabled: false
server:
    http_listen_port: 3100
common:
    ring:
        instance_addr: 127.0.0.1
        kvstore:
            store: inmemory
    replication_factor: 1
    path_prefix: /tmp/loki
schema_config:
    configs:
        - from: 2020-05-15
          store: tsdb
          object_store: filesystem
          schema: v13
          index:
              prefix: index_
              period: 24h
storage_config:
    filesystem:
        directory: /tmp/loki/chunks
' >loki.yaml
docker run -d \
    --restart always \
    --network container:exporter \
    --name loki \
    -v $(pwd)/loki.yaml:/etc/loki/local-config.yaml:ro \
    $IMAGE_LOKI

echo '
global:
    scrape_interval: 15s
    evaluation_interval: 15s
scrape_configs:
    - job_name: "node-exporter"
      static_configs:
          - targets: ["127.0.0.1:9100"]
    - job_name: "postgres-exporter"
      static_configs:
          - targets: ["127.0.0.1:9187"]
    - job_name: "capitan"
      static_configs:
          - targets: ["127.0.0.1:23456"]
    - job_name: "titan"
      static_configs:
          - targets: ["127.0.0.1:6666"]
' >prometheus.yml
docker run -d \
    --restart always \
    --name prometheus \
    --network container:exporter \
    -v $(pwd)/prometheus.yml:/etc/prometheus/prometheus.yml:ro \
    $IMAGE_PROMETHEUS

docker run -d \
    --restart always \
    --name grafana \
    --network container:exporter \
    $IMAGE_GRAFANA
docker exec -it grafana grafana cli plugins install grafana-clock-panel
docker container restart grafana

echo "
Prometheus: http://127.0.0.1:9090
Loki: http://127.0.0.1:3100
"
```

- Run containers

## Reset

- Remove loki plugin

```sh
rm -rf /etc/docker/daemon.json
systemctl restart docker.service
docker plugin disable loki --force
docker plugin rm loki --force
```

- Stop containers

```bash
docker kill prometheus
docker kill exporter
docker kill loki
docker kill grafana
docker system prune --volumes -f
```

- There should be no containers running
