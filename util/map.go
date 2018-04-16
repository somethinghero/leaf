package util

import (
	"sync"
)

//Map wrap of map
type Map struct {
	sync.RWMutex
	m map[interface{}]interface{}
}

func (m *Map) init() {
	if m.m == nil {
		m.m = make(map[interface{}]interface{})
	}
}

//UnsafeGet unsafe get
func (m *Map) UnsafeGet(key interface{}) interface{} {
	if m.m == nil {
		return nil
	}
	return m.m[key]
}

//Get safe get
func (m *Map) Get(key interface{}) interface{} {
	m.RLock()
	defer m.RUnlock()
	return m.UnsafeGet(key)
}

//UnsafeSet unsafe set
func (m *Map) UnsafeSet(key interface{}, value interface{}) {
	m.init()
	m.m[key] = value
}

//Set safe set
func (m *Map) Set(key interface{}, value interface{}) {
	m.Lock()
	defer m.Unlock()
	m.UnsafeSet(key, value)
}

//TestAndSet try set
func (m *Map) TestAndSet(key interface{}, value interface{}) interface{} {
	m.Lock()
	defer m.Unlock()

	m.init()

	if v, ok := m.m[key]; ok {
		return v
	}
	m.m[key] = value
	return nil
}

//UnsafeDel unsafe delete
func (m *Map) UnsafeDel(key interface{}) {
	m.init()
	delete(m.m, key)
}

//Del safe delete
func (m *Map) Del(key interface{}) {
	m.Lock()
	defer m.Unlock()
	m.UnsafeDel(key)
}

//UnsafeLen unsafe get len
func (m *Map) UnsafeLen() int {
	if m.m == nil {
		return 0
	}
	return len(m.m)
}

//Len safe get len
func (m *Map) Len() int {
	m.RLock()
	defer m.RUnlock()
	return m.UnsafeLen()
}

//UnsafeRange unsafe range
func (m *Map) UnsafeRange(f func(interface{}, interface{})) {
	if m.m == nil {
		return
	}
	for k, v := range m.m {
		f(k, v)
	}
}

//RLockRange read lock range
func (m *Map) RLockRange(f func(interface{}, interface{})) {
	m.RLock()
	defer m.RUnlock()
	m.UnsafeRange(f)
}

//LockRange safe range
func (m *Map) LockRange(f func(interface{}, interface{})) {
	m.Lock()
	defer m.Unlock()
	m.UnsafeRange(f)
}
