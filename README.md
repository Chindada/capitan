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
    ghcr.io/chindada/capitan:v1.0
docker logs -f capitan
```

```sh
docker stop titan
docker stop capitan
docker rmi -f $(docker images -a -q)
docker system prune --volumes -f
```
