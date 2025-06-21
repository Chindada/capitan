#!/bin/bash
set -e

go install github.com/swaggo/swag/cmd/swag@latest

echo 'package main' >./srv.go
swag fmt -g internal/controller/http/router/router.go
swag init -q --pdl 3 --outputTypes yaml,go -g internal/controller/http/router/router.go
rm -rf ./srv.go

redocly build-docs ./docs/swagger.yaml -o docs/index.html
git add ./docs
