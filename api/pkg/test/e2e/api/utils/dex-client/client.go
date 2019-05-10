package dexclient

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"

	"github.com/dexidp/dex/api"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type DexClient interface {
	CreateUser(user DexUser) error
	ListUsers() ([]DexUser, error)
	DeleteUser(user DexUser) error
}

type dexClient struct {
	client api.DexClient
}

func (d *dexClient) CreateUser(user DexUser) error {
	p := api.Password{
		Email:    user.GetEmail(),
		Hash:     user.GetPassword(),
		Username: user.GetUsername(),
		UserId:   user.GetID(),
	}

	createReq := &api.CreatePasswordReq{
		Password: &p,
	}

	if resp, err := d.client.CreatePassword(context.TODO(), createReq); err != nil || resp.AlreadyExists {
		if resp != nil && resp.AlreadyExists {
			return fmt.Errorf("user %s already exists", createReq.Password.Email)
		}
		return fmt.Errorf("failed to create user: %v", err)
	}

	return nil
}

func (d *dexClient) ListUsers() ([]DexUser, error) {
	users := make([]DexUser, 0)

	resp, err := d.client.ListPasswords(context.TODO(), &api.ListPasswordReq{})
	if err != nil {
		return users, fmt.Errorf("failed to list password: %v", err)
	}

	for _, password := range resp.Passwords {
		user := NewSimpleDexUser(password.UserId, password.Username, password.Email)
		user.SetRawPassword(password.Hash)
		users = append(users, user)
	}

	return users, nil
}

func (d *dexClient) DeleteUser(user DexUser) error {
	deleteReq := &api.DeletePasswordReq{
		Email: user.GetEmail(),
	}

	if resp, err := d.client.DeletePassword(context.TODO(), deleteReq); err != nil || resp.NotFound {
		if resp.NotFound {
			return fmt.Errorf("user %s not found", deleteReq.Email)
		}
		return fmt.Errorf("failed to delete user: %v", err)
	}

	return nil
}

func NewDexClient(hostAndPort string, clientCrt, clientKey, caCrt []byte) (DexClient, error) {
	cPool := x509.NewCertPool()
	if !cPool.AppendCertsFromPEM(caCrt) {
		return nil, fmt.Errorf("failed to parse CA crt")
	}

	clientCert, err := tls.X509KeyPair(clientCrt, clientKey)
	if err != nil {
		return nil, err
	}

	clientTLSConfig := &tls.Config{
		RootCAs:      cPool,
		Certificates: []tls.Certificate{clientCert},
	}
	creds := credentials.NewTLS(clientTLSConfig)

	conn, err := grpc.Dial(hostAndPort, grpc.WithTransportCredentials(creds))
	if err != nil {
		return nil, fmt.Errorf("dail: %v", err)
	}

	return &dexClient{client: api.NewDexClient(conn)}, nil
}
