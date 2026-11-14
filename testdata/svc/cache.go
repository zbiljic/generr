package svc

// Cache is a generic type; the receivers of its error functions have to list
// the type parameters too.
type Cache[K comparable, V any] struct{}

// Get has a pointer receiver.
func (c *Cache[K, V]) Get(key K) (V, error) {
	var zero V
	return zero, nil
}

// Len has a value receiver and names only one type parameter.
func (c Cache[K, _]) Len() (int, error) { return 0, nil }

// Expire uses time.Duration, which only service.go imports.
//
// +generr:params=time.Duration(ttl);error
func (c *Cache[K, V]) Expire() error { return nil }
