package session

import (
	"testing"
	"time"
)

func TestMemoryCache(t *testing.T) {
	DefaultDuration = 20 * time.Millisecond
	cache := New()

	s := cache.Create()
	s.Values["foo"] = "bar"

	if err := cache.Save(s); err != nil {
		t.Error(err.Error())
	}

	s2, err := cache.Get(s.Id)
	if err != nil {
		t.Errorf("Failed to get session: %s", err)
	} else if s2 == nil {
		t.Error("Expected S2 to be a session.")
		return
	}

	if s2.Values["foo"] != "bar" {
		t.Errorf("Expected 'foo' to be 'bar', got '%s'", s.Values["foo"])
	}

	if err := cache.Delete(s2.Id); err != nil {
		t.Errorf("Unexpected error during delete: %s", err)
	}

	if _, err := cache.Get(s2.Id); err == nil {
		t.Errorf("Expected an error when getting a deleted item.")
	} else if err != ErrNotFound {
		t.Errorf("Expected '%s' error, got '%s'", ErrNotFound, err)
	}

	s3 := &Session{
		Id:      "hello",
		Expires: time.Now().Add(10 * time.Millisecond),
	}
	if err := cache.Save(s3); err == nil {
		t.Errorf("Cache should not set unknown key.")
	}

	s4 := cache.Create()
	if _, err := cache.Get(s4.Id); err != nil {
		t.Error("Cache should be set for s4")
	}

	s4.Expires = time.Now()
	time.Sleep(5)
	if _, err := cache.Get(s4.Id); err == nil {
		t.Errorf("Cached session s4 should have expired. Exp time: %s", s4.Expires)
	}

}
