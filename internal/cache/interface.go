package cache

type Cache interface {
	Get(key string) ([]byte, bool)
	Set(key string, value []byte)
	Delete(key string)
	Len() int
	SaveToFile(filename string) error
	LoadFromFile(filename string) error
}
