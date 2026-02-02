package discovery

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/martinsuchenak/rackd/internal/credentials"
	"github.com/martinsuchenak/rackd/internal/model"
	"golang.org/x/crypto/ssh"
)

// HostKeyStore stores known host keys for trust-on-first-use verification
type HostKeyStore interface {
	Get(host string) (ssh.PublicKey, error)
	Store(host string, key ssh.PublicKey) error
}

// MemoryHostKeyStore is an in-memory implementation of HostKeyStore
type MemoryHostKeyStore struct {
	keys map[string]ssh.PublicKey
	mu   sync.RWMutex
}

func NewMemoryHostKeyStore() *MemoryHostKeyStore {
	return &MemoryHostKeyStore{keys: make(map[string]ssh.PublicKey)}
}

func (s *MemoryHostKeyStore) Get(host string) (ssh.PublicKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if key, ok := s.keys[host]; ok {
		return key, nil
	}
	return nil, nil
}

func (s *MemoryHostKeyStore) Store(host string, key ssh.PublicKey) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.keys[host] = key
	return nil
}

type SSHScanner struct {
	credStore    credentials.Storage
	timeout      time.Duration
	hostKeyStore HostKeyStore
}

func NewSSHScanner(credStore credentials.Storage, timeout time.Duration) *SSHScanner {
	return &SSHScanner{credStore: credStore, timeout: timeout, hostKeyStore: NewMemoryHostKeyStore()}
}

func NewSSHScannerWithHostKeys(credStore credentials.Storage, timeout time.Duration, hostKeyStore HostKeyStore) *SSHScanner {
	return &SSHScanner{credStore: credStore, timeout: timeout, hostKeyStore: hostKeyStore}
}

type SSHResult struct {
	OS              string
	OSVersion       string
	Kernel          string
	Hostname        string
	Packages        []string
	RunningServices []string
}

func (s *SSHScanner) Scan(ctx context.Context, ip string, credentialID string) (*SSHResult, error) {
	cred, err := s.credStore.Get(credentialID)
	if err != nil {
		return nil, fmt.Errorf("credential lookup failed: %w", err)
	}

	config := &ssh.ClientConfig{
		User:            cred.SSHUsername,
		HostKeyCallback: s.trustOnFirstUseCallback(ip),
		Timeout:         s.timeout,
	}

	switch cred.Type {
	case "ssh_password":
		config.Auth = []ssh.AuthMethod{ssh.Password(cred.SSHKeyID)}
	case "ssh_key":
		signer, err := ssh.ParsePrivateKey([]byte(cred.SSHKeyID))
		if err != nil {
			return nil, fmt.Errorf("failed to parse SSH key: %w", err)
		}
		config.Auth = []ssh.AuthMethod{ssh.PublicKeys(signer)}
	default:
		return nil, fmt.Errorf("unsupported credential type for SSH: %s", cred.Type)
	}

	client, err := ssh.Dial("tcp", net.JoinHostPort(ip, "22"), config)
	if err != nil {
		return nil, fmt.Errorf("SSH connect failed: %w", err)
	}
	defer client.Close()

	result := &SSHResult{}
	s.getOSInfo(client, result)
	s.getPackages(client, result)
	s.getServices(client, result)

	return result, nil
}

func (s *SSHScanner) runCommand(client *ssh.Client, cmd string) (string, error) {
	session, err := client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()
	out, err := session.CombinedOutput(cmd)
	return strings.TrimSpace(string(out)), err
}

func (s *SSHScanner) getOSInfo(client *ssh.Client, result *SSHResult) {
	if out, err := s.runCommand(client, "cat /etc/os-release 2>/dev/null"); err == nil {
		for _, line := range strings.Split(out, "\n") {
			if strings.HasPrefix(line, "PRETTY_NAME=") {
				result.OS = strings.Trim(strings.TrimPrefix(line, "PRETTY_NAME="), "\"")
			}
			if strings.HasPrefix(line, "VERSION_ID=") {
				result.OSVersion = strings.Trim(strings.TrimPrefix(line, "VERSION_ID="), "\"")
			}
		}
	}
	if out, err := s.runCommand(client, "uname -r"); err == nil {
		result.Kernel = out
	}
	if out, err := s.runCommand(client, "hostname"); err == nil {
		result.Hostname = out
	}
}

func (s *SSHScanner) getPackages(client *ssh.Client, result *SSHResult) {
	if out, err := s.runCommand(client, "dpkg -l 2>/dev/null | tail -n +6 | awk '{print $2}' | head -100"); err == nil && out != "" {
		result.Packages = strings.Split(out, "\n")
		return
	}
	if out, err := s.runCommand(client, "rpm -qa 2>/dev/null | head -100"); err == nil && out != "" {
		result.Packages = strings.Split(out, "\n")
	}
}

func (s *SSHScanner) getServices(client *ssh.Client, result *SSHResult) {
	if out, err := s.runCommand(client, "systemctl list-units --type=service --state=running --no-pager --no-legend 2>/dev/null | awk '{print $1}' | head -50"); err == nil && out != "" {
		result.RunningServices = strings.Split(out, "\n")
		return
	}
	if out, err := s.runCommand(client, "ps aux 2>/dev/null | awk 'NR>1 {print $11}' | sort -u | head -50"); err == nil && out != "" {
		result.RunningServices = strings.Split(out, "\n")
	}
}

func (s *SSHScanner) IsAvailable(ip string, cred *model.Credential) bool {
	config := &ssh.ClientConfig{
		User:            cred.SSHUsername,
		HostKeyCallback: s.trustOnFirstUseCallback(ip),
		Timeout:         2 * time.Second,
	}
	if cred.Type == "ssh_password" {
		config.Auth = []ssh.AuthMethod{ssh.Password(cred.SSHKeyID)}
	} else if cred.Type == "ssh_key" {
		signer, err := ssh.ParsePrivateKey([]byte(cred.SSHKeyID))
		if err != nil {
			return false
		}
		config.Auth = []ssh.AuthMethod{ssh.PublicKeys(signer)}
	} else {
		return false
	}
	client, err := ssh.Dial("tcp", net.JoinHostPort(ip, "22"), config)
	if err != nil {
		return false
	}
	client.Close()
	return true
}

// trustOnFirstUseCallback implements TOFU host key verification
func (s *SSHScanner) trustOnFirstUseCallback(host string) ssh.HostKeyCallback {
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		knownKey, err := s.hostKeyStore.Get(host)
		if err != nil {
			return fmt.Errorf("host key lookup failed: %w", err)
		}
		if knownKey == nil {
			return s.hostKeyStore.Store(host, key)
		}
		if string(knownKey.Marshal()) != string(key.Marshal()) {
			return fmt.Errorf("host key mismatch for %s: possible MITM attack", host)
		}
		return nil
	}
}
