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
docker rmi -f $(docker images -a -q)
docker system prune --volumes -f
```
