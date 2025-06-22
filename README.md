# CAPITAN

## Command

```sh
# stop old container
docker stop capitan -t 600
docker system prune --volumes -f
# run
docker pull ghcr.io/chindada/capitan:v1.0
docker run -d \
    --restart always \
    --name capitan \
    --network container:titan \
    -v $(pwd)/capitan:/usr/share/app/data \
    -v $(pwd)/db_backup:/usr/share/app/db_backup \
    -v $(pwd)/logs:/usr/share/app/logs \
    ghcr.io/chindada/capitan:v1.0
docker logs -f capitan
```

```sh
docker stop titan
docker stop capitan
docker system prune --volumes -f
```

```sh
docker stop titan
docker stop capitan
docker system prune --volumes -f
docker pull ghcr.io/chindada/titan:v1.0
docker run -d \
    --restart always \
    --name titan \
    -v $(pwd)/data/config.yaml:/usr/share/app/titan/data/config.yaml \
    -p 80:80 \
    -p 443:443 \
    ghcr.io/chindada/titan:v1.0
docker pull ghcr.io/chindada/capitan:v1.0
docker run -d \
    --restart always \
    --name capitan \
    --network container:titan \
    -v $(pwd)/capitan:/usr/share/app/data \
    -v $(pwd)/db_backup:/usr/share/app/db_backup \
    -v $(pwd)/logs:/usr/share/app/logs \
    ghcr.io/chindada/capitan:v1.0
```
