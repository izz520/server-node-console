package sshclient

import (
	"context"
	"fmt"
	"net"
	"time"

	"golang.org/x/crypto/ssh"
)

type AuthMethod string

const (
	AuthMethodPassword   AuthMethod = "password"
	AuthMethodPrivateKey AuthMethod = "private_key"
)

type TestRequest struct {
	Host       string
	Port       int
	Username   string
	AuthMethod AuthMethod
	Password   string
	PrivateKey string
	Timeout    time.Duration
}

func TestConnection(ctx context.Context, req TestRequest) error {
	if req.Timeout == 0 {
		req.Timeout = 10 * time.Second
	}

	authMethod, err := buildAuthMethod(req)
	if err != nil {
		return err
	}

	config := &ssh.ClientConfig{
		User:            req.Username,
		Auth:            []ssh.AuthMethod{authMethod},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         req.Timeout,
	}

	address := fmt.Sprintf("%s:%d", req.Host, req.Port)
	dialer := net.Dialer{Timeout: req.Timeout}
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return fmt.Errorf("connect ssh address: %w", err)
	}
	defer conn.Close()

	clientConn, chans, reqs, err := ssh.NewClientConn(conn, address, config)
	if err != nil {
		return fmt.Errorf("authenticate ssh: %w", err)
	}
	client := ssh.NewClient(clientConn, chans, reqs)
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("create ssh session: %w", err)
	}
	defer session.Close()

	if err := session.Run("true"); err != nil {
		return fmt.Errorf("run ssh probe: %w", err)
	}

	return nil
}

func buildAuthMethod(req TestRequest) (ssh.AuthMethod, error) {
	switch req.AuthMethod {
	case AuthMethodPassword:
		if req.Password == "" {
			return nil, fmt.Errorf("ssh password is required")
		}
		return ssh.Password(req.Password), nil
	case AuthMethodPrivateKey:
		if req.PrivateKey == "" {
			return nil, fmt.Errorf("ssh private key is required")
		}
		signer, err := ssh.ParsePrivateKey([]byte(req.PrivateKey))
		if err != nil {
			return nil, fmt.Errorf("parse ssh private key: %w", err)
		}
		return ssh.PublicKeys(signer), nil
	default:
		return nil, fmt.Errorf("unsupported ssh auth method: %s", req.AuthMethod)
	}
}
