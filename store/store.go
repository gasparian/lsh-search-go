package store

// Iterator consists from only one method which returns uid of the next vector
type Iterator interface {
	Next() (string, bool)
}

// Store methods to be able to hold and use search index
// It implies storage vectors at one place, and
// LSH hashes with vectors uid in other places
// to not duplicate vectors themselves
type Store interface {
	SetVector(id string, vec []float64) error
	GetVector(id string) ([]float64, error)
	SetHash(permutation int, hash uint64, vecId string) error
	GetHashIterator(permutation int, hash uint64) (Iterator, error)
	Clear() error
}
