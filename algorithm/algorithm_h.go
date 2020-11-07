package algorithm

// IVector here just to quicly observe which methods exists on Vector struct
type IVector interface {
	Add(*Vector) *Vector
	ConstMul(float64) *Vector
	DotProd(*Vector) float64
	L2(*Vector) float64
	CosineSim(*Vector) float64
}

// Vector is basic data structure to hold slice of floats and it's size
type Vector struct {
	Values []float64
	Size   int
}

// Indexer basic interface that should implement any indexer object
type Indexer interface {
	Build() error
	Dump(string) error
	Load(string) error
	GetHash(*Vector) uint64
}

// Plane struct holds data needed to work with plane
type Plane struct {
	Coefs      *Vector
	InnerPoint *Vector
}

// LSHIndex holds data for local sensetive hashing algorithm
// Add in the main code: rand.Seed(time.Now().UnixNano())
type LSHIndex struct {
	dims    int
	bias    float64
	nPlanes int
	Planes  []Plane
}
