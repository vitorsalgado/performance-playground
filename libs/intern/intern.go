// Package intern provides string interning to reduce memory for repeated strings.
// See https://en.wikipedia.org/wiki/String_interning .
package intern

import (
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

// Config holds optional settings. Modify before first use of InternString/InternBytes.
var (
	// MaxLen is the maximum length for strings to intern. Longer strings are not cached.
	MaxLen = 500
	// DisableCache disables the cache when true (every call returns a copy).
	DisableCache = false
	// CacheExpireDuration is how long entries stay in the cache before cleanup.
	CacheExpireDuration = 6 * time.Minute
)

type internStringMap struct {
	mutableLock  sync.Mutex
	mutable      map[string]string
	mutableReads uint64

	readonly atomic.Pointer[map[string]internStringMapEntry]
}

type internStringMapEntry struct {
	deadline int64 // unix seconds, for cleanup only
	s        string
}

func newInternStringMap() *internStringMap {
	m := &internStringMap{
		mutable: make(map[string]string),
	}
	readonly := make(map[string]internStringMapEntry)
	m.readonly.Store(&readonly)

	go func() {
		cleanupInterval := CacheExpireDuration / 2
		if cleanupInterval < time.Second {
			cleanupInterval = time.Second
		}
		ticker := time.NewTicker(cleanupInterval)
		for range ticker.C {
			m.cleanup()
		}
	}()

	return m
}

func (m *internStringMap) getReadonly() map[string]internStringMapEntry {
	return *m.readonly.Load()
}

func (m *internStringMap) intern(s string) string {
	if m.isSkipCache(s) {
		return strings.Clone(s)
	}

	readonly := m.getReadonly()
	e, ok := readonly[s]
	if ok {
		return e.s
	}

	m.mutableLock.Lock()
	sInterned, ok := m.mutable[s]
	if !ok {
		readonly = m.getReadonly()
		e, ok = readonly[s]
		if !ok {
			sInterned = strings.Clone(s)
			m.mutable[sInterned] = sInterned
		} else {
			sInterned = e.s
		}
	}
	m.mutableReads++
	if m.mutableReads > uint64(len(readonly)) {
		m.migrateMutableToReadonlyLocked()
		m.mutableReads = 0
	}
	m.mutableLock.Unlock()

	return sInterned
}

func (m *internStringMap) migrateMutableToReadonlyLocked() {
	readonly := m.getReadonly()
	deadline := time.Now().Unix() + int64(CacheExpireDuration.Seconds()) + 1
	readonlyCopy := make(map[string]internStringMapEntry, len(readonly)+len(m.mutable))
	for k, e := range readonly {
		readonlyCopy[k] = e
	}
	for k, s := range m.mutable {
		readonlyCopy[k] = internStringMapEntry{
			s:        s,
			deadline: deadline,
		}
	}
	m.mutable = make(map[string]string)
	m.readonly.Store(&readonlyCopy)
}

func (m *internStringMap) cleanup() {
	readonly := m.getReadonly()
	now := time.Now().Unix()
	needCleanup := false
	for _, e := range readonly {
		if e.deadline <= now {
			needCleanup = true
			break
		}
	}
	if !needCleanup {
		return
	}

	readonlyCopy := make(map[string]internStringMapEntry, len(readonly))
	for k, e := range readonly {
		if e.deadline > now {
			readonlyCopy[k] = e
		}
	}
	m.readonly.Store(&readonlyCopy)
}

func (m *internStringMap) isSkipCache(s string) bool {
	return DisableCache || len(s) > MaxLen
}

// InternBytes interns b as a string. Prefer InternString when you already have a string.
func InternBytes(b []byte) string {
	return globalMap.intern(unsafeString(b))
}

// InternString returns an interned copy of s when possible, reducing memory for repeated values.
func InternString(s string) string {
	return globalMap.intern(s)
}

// unsafeString returns a string header for b without copying. The result must not be mutated by the caller.
func unsafeString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

var globalMap = newInternStringMap()
