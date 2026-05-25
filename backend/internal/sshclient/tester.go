package sshclient

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
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
	_, err := RunCommand(ctx, req, "true")
	return err
}

func RunCommand(ctx context.Context, req TestRequest, command string) (string, error) {
	return RunCommandWithOutput(ctx, req, command, nil)
}

func RunCommandWithOutput(ctx context.Context, req TestRequest, command string, onOutput func(string)) (string, error) {
	if req.Timeout == 0 {
		req.Timeout = 30 * time.Minute
	}
	runCtx, cancel := context.WithTimeout(ctx, req.Timeout)
	defer cancel()

	authMethod, err := buildAuthMethod(req)
	if err != nil {
		return "", err
	}

	config := &ssh.ClientConfig{
		User:            req.Username,
		Auth:            []ssh.AuthMethod{authMethod},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         req.Timeout,
	}

	address := fmt.Sprintf("%s:%d", req.Host, req.Port)
	dialer := net.Dialer{Timeout: req.Timeout}
	conn, err := dialer.DialContext(runCtx, "tcp", address)
	if err != nil {
		return "", fmt.Errorf("connect ssh address: %w", err)
	}
	defer conn.Close()

	clientConn, chans, reqs, err := ssh.NewClientConn(conn, address, config)
	if err != nil {
		return "", fmt.Errorf("authenticate ssh: %w", err)
	}
	client := ssh.NewClient(clientConn, chans, reqs)
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("create ssh session: %w", err)
	}
	defer session.Close()

	var output safeBuffer
	writer := io.Writer(&output)
	var lineWriter *lineCallbackWriter
	if onOutput != nil {
		lineWriter = newLineCallbackWriter(onOutput)
		writer = io.MultiWriter(&output, lineWriter)
		defer lineWriter.Flush()
	}
	session.Stdout = writer
	session.Stderr = writer

	done := make(chan error, 1)
	go func() {
		done <- session.Run(command)
	}()

	select {
	case err := <-done:
		if err != nil {
			return output.String(), fmt.Errorf("run ssh command: %w", err)
		}
	case <-runCtx.Done():
		_ = session.Close()
		_ = client.Close()
		_ = conn.Close()
		return output.String(), fmt.Errorf("run ssh command timeout after %s: %w", req.Timeout, runCtx.Err())
	}

	return output.String(), nil
}

type lineCallbackWriter struct {
	mu       sync.Mutex
	buffer   strings.Builder
	onOutput func(string)
}

func newLineCallbackWriter(onOutput func(string)) *lineCallbackWriter {
	return &lineCallbackWriter{onOutput: onOutput}
}

func (w *lineCallbackWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	text := string(p)
	for {
		index := strings.IndexByte(text, '\n')
		if index < 0 {
			w.buffer.WriteString(text)
			return len(p), nil
		}

		w.buffer.WriteString(text[:index])
		w.emitLocked()
		text = text[index+1:]
	}
}

func (w *lineCallbackWriter) Flush() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.buffer.Len() == 0 {
		return
	}
	w.emitLocked()
}

func (w *lineCallbackWriter) emitLocked() {
	line := strings.TrimRight(w.buffer.String(), "\r")
	w.buffer.Reset()
	if strings.TrimSpace(line) == "" {
		return
	}
	w.onOutput(line)
}

type safeBuffer struct {
	mu     sync.Mutex
	buffer bytes.Buffer
}

func (b *safeBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buffer.Write(p)
}

func (b *safeBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buffer.String()
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
