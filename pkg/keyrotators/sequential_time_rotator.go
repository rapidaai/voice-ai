package keyrotators

import (
	"sync"
	"sync/atomic"

	"github.com/rapidaai/pkg/commons"
)

type SequentialTimeRotator struct {
	keys     []string
	counter  uint32
	keyMutex sync.RWMutex
	logger   commons.Logger
}

func NewSequentialTimeRotator(logger commons.Logger, keys ...string) *SequentialTimeRotator {
	return &SequentialTimeRotator{
		keys:   keys,
		logger: logger,
	}
}

func (r *SequentialTimeRotator) GetNext() string {
	r.keyMutex.RLock()
	defer r.keyMutex.RUnlock()

	if len(r.keys) == 0 {
		r.logger.Infof("SequentialTimeRotator: No keys available")
		return ""
	}

	if len(r.keys) == 1 {
		r.logger.Infof("SequentialTimeRotator: Returning single key: %s", r.keys[0])
		return r.keys[0]
	}

	index := atomic.AddUint32(&r.counter, 1) % uint32(len(r.keys))
	selectedKey := r.keys[int(index)]
	r.logger.Infof("SequentialTimeRotator: Returning key: %s (index: %d)", selectedKey, index)
	return selectedKey
}

func (r *SequentialTimeRotator) AddKey(key string) {
	r.keyMutex.Lock()
	defer r.keyMutex.Unlock()

	r.keys = append(r.keys, key)
}

func (r *SequentialTimeRotator) RemoveKey(key string) {
	r.keyMutex.Lock()
	defer r.keyMutex.Unlock()

	for i, k := range r.keys {
		if k == key {
			r.keys = append(r.keys[:i], r.keys[i+1:]...)
			break
		}
	}
}

func (r *SequentialTimeRotator) GetKeys() []string {
	r.keyMutex.RLock()
	defer r.keyMutex.RUnlock()

	keys := make([]string, len(r.keys))
	copy(keys, r.keys)
	return keys
}
