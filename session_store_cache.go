package limen

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"
)

type cacheSessionStore struct {
	cache  CacheAdapter
	prefix string
	locks  [lockStripeCount]sync.Mutex
}

func newSecondarySessionStore(core *LimenCore) *cacheSessionStore {
	return &cacheSessionStore{
		cache:  core.CacheStore(),
		prefix: core.CacheKeyPrefix(),
	}
}

func (s *cacheSessionStore) sessionKey(token string) string {
	return s.prefix + ":session:t:" + token
}

func (s *cacheSessionStore) userSessionsKey(userID any) string {
	return s.prefix + ":session:u:" + fmt.Sprint(userID)
}

func (s *cacheSessionStore) Get(ctx context.Context, token string) (*Session, error) {
	data, err := s.cache.Get(ctx, s.sessionKey(token))
	if err != nil {
		return nil, ErrSessionNotFound
	}

	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, ErrSessionNotFound
	}

	return &session, nil
}

func (s *cacheSessionStore) Set(ctx context.Context, session *Session) error {
	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	ttl := max(time.Until(session.ExpiresAt), 0)
	if err := s.cache.Set(ctx, s.sessionKey(session.Token), data, ttl); err != nil {
		return err
	}

	return s.addToUserIndex(ctx, session)
}

func (s *cacheSessionStore) Delete(ctx context.Context, token string) error {
	sess, err := s.Get(ctx, token)
	if err != nil {
		return nil
	}

	if err := s.cache.Delete(ctx, s.sessionKey(token)); err != nil {
		return err
	}

	return s.removeFromUserIndex(ctx, sess.UserID, token)
}

func (s *cacheSessionStore) ListByUserID(ctx context.Context, userID any) ([]Session, error) {
	lock := s.lockUser(userID)
	lock.Lock()
	defer lock.Unlock()
	sessions, err := s.getUserSessions(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.pruneUserSessions(ctx, userID, sessions)
}

func (s *cacheSessionStore) DeleteByUserID(ctx context.Context, userID any) error {
	lock := s.lockUser(userID)
	lock.Lock()
	defer lock.Unlock()

	sessions, err := s.getUserSessions(ctx, userID)
	if err != nil {
		return err
	}

	for _, sess := range sessions {
		if err := s.cache.Delete(ctx, s.sessionKey(sess.Token)); err != nil {
			return err
		}
	}

	return s.cache.Delete(ctx, s.userSessionsKey(userID))
}

func (s *cacheSessionStore) getUserSessions(ctx context.Context, userID any) ([]Session, error) {
	data, err := s.cache.Get(ctx, s.userSessionsKey(userID))
	if err != nil {
		if errors.Is(err, ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	var sessions []Session
	if err := json.Unmarshal(data, &sessions); err != nil {
		return nil, err
	}
	return sessions, nil
}

func (s *cacheSessionStore) addToUserIndex(ctx context.Context, session *Session) error {
	lock := s.lockUser(session.UserID)
	lock.Lock()
	defer lock.Unlock()

	sessions, _ := s.getUserSessions(ctx, session.UserID)

	for i, sess := range sessions {
		if sess.Token == session.Token {
			sessions[i] = *session
			return s.saveUserIndex(ctx, session.UserID, sessions)
		}
	}

	sessions = append(sessions, *session)
	return s.saveUserIndex(ctx, session.UserID, sessions)
}

func (s *cacheSessionStore) removeFromUserIndex(ctx context.Context, userID any, token string) error {
	lock := s.lockUser(userID)
	lock.Lock()
	defer lock.Unlock()

	sessions, err := s.getUserSessions(ctx, userID)
	if err != nil {
		return nil
	}

	filtered := sessions[:0]
	for _, sess := range sessions {
		if sess.Token != token {
			filtered = append(filtered, sess)
		}
	}

	if len(filtered) == 0 {
		return s.cache.Delete(ctx, s.userSessionsKey(userID))
	}

	return s.saveUserIndex(ctx, userID, filtered)
}

func (s *cacheSessionStore) saveUserIndex(ctx context.Context, userID any, sessions []Session) error {
	if len(sessions) == 0 {
		return s.cache.Delete(ctx, s.userSessionsKey(userID))
	}
	data, err := json.Marshal(sessions)
	if err != nil {
		return err
	}
	return s.cache.Set(ctx, s.userSessionsKey(userID), data, sessionsTTL(sessions))
}

func (s *cacheSessionStore) pruneUserSessions(ctx context.Context, userID any, sessions []Session) ([]Session, error) {
	if len(sessions) == 0 {
		return nil, nil
	}
	now := time.Now()
	live := sessions[:0]
	for _, sess := range sessions {
		if !sess.ExpiresAt.After(now) {
			continue
		}
		exists, err := s.cache.Has(ctx, s.sessionKey(sess.Token))
		if err != nil {
			return nil, err
		}
		if exists {
			live = append(live, sess)
		}
	}
	if len(live) != len(sessions) {
		if err := s.saveUserIndex(ctx, userID, live); err != nil {
			return nil, err
		}
	}
	return live, nil
}

func sessionsTTL(sessions []Session) time.Duration {
	var maxTTL time.Duration
	now := time.Now()
	for _, sess := range sessions {
		ttl := time.Until(sess.ExpiresAt)
		if ttl <= 0 {
			continue
		}
		if expiresIn := sess.ExpiresAt.Sub(now); expiresIn > maxTTL {
			maxTTL = expiresIn
		}
	}
	return maxTTL
}

func (s *cacheSessionStore) lockUser(userID any) *sync.Mutex {
	key := fmt.Sprint(userID)
	return &s.locks[lockStripeIndex(key)]
}
