#!/bin/bash

rm -rf go.mod
rm -rf go.sum

go mod init github.com/chindada/capitan

go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
go install go.uber.org/mock/mockgen@latest

go get -u github.com/chindada/panther@latest
go get -u github.com/chindada/leopard@latest

go mod tidy

git add go.mod go.sum
