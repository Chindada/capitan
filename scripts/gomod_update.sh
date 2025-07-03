#!/bin/bash

rm -rf go.mod
rm -rf go.sum

go mod init github.com/chindada/capitan

go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
go install go.uber.org/mock/mockgen@latest

go get -u github.com/chindada/panther@97e69b43b7e3aafb697ceb976289e314398ebc3c
go get -u github.com/chindada/leopard@v1.0.0

go mod tidy

git add go.mod go.sum
