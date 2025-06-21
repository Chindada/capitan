# ROOT_PATH defined in panther
FROM ghcr.io/chindada/panther:v1.0
ENV TZ=Asia/Taipei \
    SRV_PORT=23456 \
    HTTP_PORT=80 \
    HTTPS_PORT=443
RUN mkdir -p \
    ${ROOT_PATH}/bin \
    ${ROOT_PATH}/scripts
WORKDIR ${ROOT_PATH}
COPY bin/capitan bin/capitan
COPY bin/dbtool bin/dbtool
COPY dist dist
COPY certs certs
COPY docker/scripts/docker-entrypoint.sh scripts/docker-entrypoint.sh
ENTRYPOINT ["/usr/bin/dumb-init", "--"]
CMD ["bash", "-c" ,"./scripts/docker-entrypoint.sh"]
