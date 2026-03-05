package discovery

import (
	"context"

	"github.com/martinsuchenak/rackd/internal/storage"
	"golang.org/x/crypto/ssh"
)

// DBHostKeyStore implements HostKeyStore using storage.ExtendedStorage
type DBHostKeyStore struct {
	store storage.ExtendedStorage
}

func NewDBHostKeyStore(store storage.ExtendedStorage) *DBHostKeyStore {
	return &DBHostKeyStore{store: store}
}

func (s *DBHostKeyStore) Get(host string) (ssh.PublicKey, error) {
	keyBytes, err := s.store.GetSSHHostKey(context.Background(), host)
	if err != nil {
		return nil, err
	}
	if len(keyBytes) == 0 {
		return nil, nil
	}

	key, err := ssh.ParsePublicKey(keyBytes)
	if err != nil {
		return nil, err
	}
	return key, nil
}

func (s *DBHostKeyStore) Store(host string, key ssh.PublicKey) error {
	return s.store.SaveSSHHostKey(context.Background(), host, key.Marshal())
}
