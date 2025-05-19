package consistent

type HashFunc interface {
	Sum64([]byte) uint64
	Sum64String(string) uint64
}

type Node interface {
	String() string
}
