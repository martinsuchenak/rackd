package storage

import (
	"context"
	"encoding/json"
	"time"

	"github.com/martinsuchenak/rackd/internal/auth"
	"github.com/redis/go-redis/v9"
)

// ValkeySessionStore provides a Valkey-backed implementation of auth.SessionStore
type ValkeySessionStore struct {
	client *redis.Client
}

// NewValkeySessionStore creates a new ValkeySessionStore
func NewValkeySessionStore(client *redis.Client) *ValkeySessionStore {
	return &ValkeySessionStore{client: client}
}

// sessionKey returns the full Valkey key for a given token
func sessionKey(token string) string {
	return "rackd:session:" + token
}

// userIndexKey returns the key for the set of sessions owned by this user
func userIndexKey(userID string) string {
	return "rackd:user_sessions:" + userID
}

func (s *ValkeySessionStore) Save(ctx context.Context, session *auth.Session) error {
	data, err := json.Marshal(session)
	if err != nil {
		return err
	}

	ttl := time.Until(session.ExpiresAt)
	if ttl < 0 {
		return auth.ErrSessionExpired
	}

	pipe := s.client.Pipeline()
	pipe.Set(ctx, sessionKey(session.Token), data, ttl)

	if session.UserID != "" { // legacy keys might lack UserID, but we assume true sessions have it
		pipe.SAdd(ctx, userIndexKey(session.UserID), session.Token)
		pipe.Expire(ctx, userIndexKey(session.UserID), ttl)
	}

	_, err = pipe.Exec(ctx)
	return err
}

func (s *ValkeySessionStore) Get(ctx context.Context, token string) (*auth.Session, error) {
	data, err := s.client.Get(ctx, sessionKey(token)).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, auth.ErrSessionNotFound
		}
		return nil, err
	}

	var session auth.Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, err
	}

	if time.Now().UTC().After(session.ExpiresAt) {
		_ = s.Delete(ctx, token)
		return nil, auth.ErrSessionExpired
	}

	return &session, nil
}

func (s *ValkeySessionStore) Delete(ctx context.Context, token string) error {
	// Need to check which user it belongs to if we strictly want to keep SREM accurate,
	// but it's simpler to just delete the key. The set index will accumulate obsolete tokens,
	// but those will just fail when looked up to be deleted via user.
	// We can try fetching first:
	session, err := s.Get(ctx, token)
	if err == nil && session.UserID != "" {
		_ = s.client.SRem(ctx, userIndexKey(session.UserID), token).Err()
	}

	err = s.client.Del(ctx, sessionKey(token)).Err()
	return err
}

func (s *ValkeySessionStore) DeleteByUser(ctx context.Context, userID string) error {
	idxKey := userIndexKey(userID)
	tokens, err := s.client.SMembers(ctx, idxKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil
		}
		return err
	}

	if len(tokens) == 0 {
		return nil
	}

	pipe := s.client.Pipeline()
	for _, token := range tokens {
		pipe.Del(ctx, sessionKey(token))
	}
	pipe.Del(ctx, idxKey)
	_, err = pipe.Exec(ctx)
	return err
}

func (s *ValkeySessionStore) Cleanup(ctx context.Context) error {
	// Valkey/Redis automatically expires TTL-based keys, so Cleanup is a no-op here.
	return nil
}
