/* Package session provides persistent session supports.

*/
package session

import (
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/aokoli/goutils"
)

// Cache describes session storage.
//
// Cache is capable of persisting sessions across requests.
//
// Cache *must* generate session keys, and *must not* accept
// session keys that it did not generate.
type Cache interface {
	// Create creates a session and sets the session ID.
	Create() *Session
	// Save saves a session. While some backends to not require this for
	// persistence, it is always best to use it.
	Save(*Session) error
	// Get retrieves a session when given its Id.
	Get(string) (*Session, error)
	// Delete deletes a session by Id.
	Delete(string) error
	// Expire removes expired sessions. It is safe to call in
	// a goroutine.
	Expire()
}

// The Cookie name for a Session.
const SessionKey = "SESSION"

var (
	// ErrExpired indicates that a cache entry has expired, but has
	// not been pruned yet.
	ErrExpired = errors.New("session expired")
	// ErrNotFound indicates that a session is not in the cache.
	ErrNotFound = errors.New("session not found")
	// ErrUnrecognized means the session format is not recognized.
	// Typically, this indicates that the session key is not found or
	// in the wrong format.
	ErrUnrecognized = errors.New("unrecognized session")
)

// DefaultDuration controls the default session duration. Note that
// individual sessions can alter their expiration time. There is no way
// to set a session to never expire.
var DefaultDuration time.Duration = time.Minute * 5

// New creates a new MemoryCache session cache.
//
// Typical usage:
//
// 	// Create a cache
// 	cache := session.New()
// 	// Whenever you need a new session:
// 	sess := cache.Create()
// 	println(sess.Id)
// 	sess.Values["myKey"] = "myvalue"
// 	cache.Save(sess)

func New() *MemoryCache {
	return &MemoryCache{
		store: map[string]*Session{},
	}
}

// MemoryCache implements a session cache that stores data in-memory.
type MemoryCache struct {
	mx    sync.RWMutex
	store map[string]*Session
}

// Create generates a session and sets its key.
//
// Sessions are immediately stored. Expiration range is set
// according to the DefaultDuration.
func (m *MemoryCache) Create() *Session {
	id, err := goutils.RandomAlphaNumeric(60)
	if err != nil {
		// All of the errors that RandomAlphaNumeric returns are for
		// cases where the parameters were incorrect. Normally, this is
		// cause for a panic. (In fact, the only way to get it to error
		// is to set a less-than-zero number as the param.)
		panic(err)
	}
	s := &Session{
		Id:      "$ms$" + id,
		Expires: time.Now().Add(DefaultDuration),
		Values:  map[string]interface{}{},
	}
	m.store[s.Id] = s

	return s
}

// Save saves a session.
//
// Since MemoryCache manages pointers, strictly speaking this is not
// often necessary. But you should use it anyway, since other backends
// may need an explict Save call to persist the data.
//
// You can replace the Session with a different Session object by using
// this method. However, the Session Id must match an existing session.
func (m *MemoryCache) Save(s *Session) error {
	// Basically, this provides to option to replace an old session
	// with a new one (change the pointer), provided that the old
	// one existed and is still valid.
	m.mx.Lock()
	defer m.mx.Unlock()

	if old, ok := m.store[s.Id]; !ok || !old.Valid() {
		return ErrUnrecognized
	}

	m.store[s.Id] = s
	return nil
}

// Get retrieves the Session for the given Id.
//
// This will error out if the session is not found or if the
// session is found, but expired.
func (m *MemoryCache) Get(id string) (*Session, error) {
	m.mx.RLock()
	s, ok := m.store[id]
	if !ok {
		m.mx.RUnlock()
		return nil, ErrNotFound
	}
	if !s.Valid() {
		// We have to upgrade the lock. There's no harm in a yield between.
		m.mx.RUnlock()
		m.mx.Lock()
		delete(m.store, id)
		m.mx.Unlock()
		return nil, ErrExpired
	}
	m.mx.RUnlock()
	return s, nil
}

// Delete removes a session from the cache.
//
// This will not return an error if the session Id is already
// gone or expired. It only returns an error if the session is found, yet
// cannot be deleted.
func (m *MemoryCache) Delete(id string) error {
	m.mx.Lock()
	defer m.mx.Unlock()
	delete(m.store, id)
	return nil
}

// Expire removes any expired sessions.
func (m *MemoryCache) Expire() {
	m.mx.Lock()
	defer m.mx.Unlock()

	mark := []string{}
	for k, v := range m.store {
		if !v.Valid() {
			mark = append(mark, k)
		}
	}
	for _, d := range mark {
		delete(m.store, d)
	}
}

// Session describes a session object. A session may have arbitrary data
// attached to it.
type Session struct {
	Id      string
	Values  map[string]interface{}
	Expires time.Time
}

// Valid indicates whether this session is still valid.
//
// It returns false if the session has expired.
//
// An invalid session may be garbage collected.
func (s *Session) Valid() bool {
	return !time.Now().After(s.Expires)
}

// Cookie generates an HTTP cookie representing this session.
//
// This does not set Path or Domain, nor does it turn on Secure or HttpOnly.
//
// MaxAge is automatically set to 0 (delete) if Valid returns false. Otherwise
// it sets the MaxAge at the default duration.
func (s *Session) Cookie() *http.Cookie {
	ma := 0
	if s.Valid() {
		ma = int(DefaultDuration.Seconds())
	}
	return &http.Cookie{
		Name:    SessionKey,
		Value:   s.Id,
		Expires: s.Expires,
		MaxAge:  ma,
	}
}
