#!/bin/bash

rm -rf go.mod
rm -rf go.sum

go mod init github.com/chindada/capitan

go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
go install go.uber.org/mock/mockgen@latest

go get -u github.com/chindada/panther@95f54c8fc4ae04dee3f15c82a557fe178b26b0fa
go get -u github.com/chindada/leopard@v1.0.0

go mod tidy

git add go.mod go.sum
