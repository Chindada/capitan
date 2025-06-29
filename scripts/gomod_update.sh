#!/bin/bash

rm -rf go.mod
rm -rf go.sum

go mod init github.com/chindada/capitan

go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
go install go.uber.org/mock/mockgen@latest

go get -u github.com/chindada/panther@c01bbb0b7be34b9c68f75ba874639de8d4eca2dd
go get -u github.com/chindada/leopard@v1.0.0

go mod tidy

git add go.mod go.sum
