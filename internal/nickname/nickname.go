package nickname

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
)

// Store holds a thread-safe mapping of IP -> nickname, backed by a JSON file.
type Store struct {
	mu   sync.RWMutex
	path string
	data map[string]string
}

// Load reads the JSON nickname file at path and returns a Store.
// If the file does not exist, an empty Store is returned without error.
func Load(path string) (*Store, error) {
	st := &Store{
		path: path,
		data: make(map[string]string),
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return st, nil
		}
		return nil, fmt.Errorf("nickname: read %s: %w", path, err)
	}
	if err := json.Unmarshal(data, &st.data); err != nil {
		return nil, fmt.Errorf("nickname: parse %s: %w", path, err)
	}
	return st, nil
}

// Get returns the nickname for ip, or "" if none is set.
func (st *Store) Get(ip string) string {
	st.mu.RLock()
	defer st.mu.RUnlock()
	return st.data[ip]
}

// Set assigns name to ip and persists the store to disk.
func (st *Store) Set(ip, name string) error {
	st.mu.Lock()
	defer st.mu.Unlock()
	if name == "" {
		delete(st.data, ip)
	} else {
		st.data[ip] = name
	}
	return st.save()
}

// All returns a copy of the full nickname map.
func (st *Store) All() map[string]string {
	st.mu.RLock()
	defer st.mu.RUnlock()
	out := make(map[string]string, len(st.data))
	for k, v := range st.data {
		out[k] = v
	}
	return out
}

// save writes the current data map to disk (must be called with mu held).
func (st *Store) save() error {
	b, err := json.MarshalIndent(st.data, "", "  ")
	if err != nil {
		return fmt.Errorf("nickname: marshal: %w", err)
	}
	if err := os.WriteFile(st.path, b, 0644); err != nil {
		return fmt.Errorf("nickname: write %s: %w", st.path, err)
	}
	return nil
}
