#!/bin/bash

rm -rf go.mod
rm -rf go.sum

go mod init github.com/chindada/capitan

go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
go install go.uber.org/mock/mockgen@latest

go get -u github.com/chindada/panther@fc817486f4588fccc6e4d69f3931682f585107d7
go get -u github.com/chindada/leopard@v1.0.0

go mod tidy

git add go.mod go.sum
