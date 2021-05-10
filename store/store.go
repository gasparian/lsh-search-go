package store

// Iterator consists from only one method which returns uid of the next vector
type Iterator interface {
	Next() (string, error)
}

// Store methods to be able to hold and use search index
type Store interface {
	SetVector(id string, vec []float64) error
	GetVector(id string) ([]float64, error)
	SetHash(permutation int, hash uint64, vecId string) error
	GetHashIterator(permutation int, hash uint64) (Iterator, error)
	Clear() error
}
