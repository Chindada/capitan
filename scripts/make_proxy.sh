#!/bin/bash
set -e

rm -rf proxy
mkdir -p proxy

NGINX_VERSION=1.28.0
ROOT_PATH=$(pwd)
curl -fSL https://nginx.org/download/nginx-$NGINX_VERSION.tar.gz --output nginx-$NGINX_VERSION.tar.gz
tar zxvf nginx-$NGINX_VERSION.tar.gz
cd nginx-$NGINX_VERSION
./configure --prefix=${ROOT_PATH}/proxy --with-http_ssl_module --sbin-path=sbin/proxy
make install
cd -
rm -rf nginx-$NGINX_VERSION.tar.gz nginx-$NGINX_VERSION
