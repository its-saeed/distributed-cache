package cache

import "time"

type Cache interface {
	Get(key string) ([]byte, bool)
	SetWithTtl(key string, value []byte, ttl time.Duration)
	Set(key string, value []byte)
	Delete(key string)
	Len() int
	SaveToFile(filename string) error
	LoadFromFile(filename string) error
}
