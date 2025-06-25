#!/bin/bash

rm -rf go.mod
rm -rf go.sum

go mod init github.com/chindada/capitan

go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
go install go.uber.org/mock/mockgen@latest

go get -u github.com/chindada/panther@40bac16f93aa288796d65cc3546f21e336fe541d
go get -u github.com/chindada/leopard@v1.0.0

go mod tidy

git add go.mod go.sum
