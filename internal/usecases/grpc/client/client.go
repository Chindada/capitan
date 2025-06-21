// Package client package client
package client

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func NewInsecureClient(gRPCPath string) (*grpc.ClientConn, error) {
	conn, err := grpc.NewClient(gRPCPath, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return conn, nil
}
