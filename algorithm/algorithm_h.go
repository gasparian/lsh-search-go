package algorithm

type IVector interface {
	Add(*Vector) *Vector
	ConstMul(float64) *Vector
	DotProd(*Vector) float64
	L2(*Vector) float64
	CosineSim(*Vector) float64
}

type Vector struct {
	Values []float64
	Size   int
}

type Indexer interface {
	Build() error
	Dump(string) error
	Load(string) error
	GetHash(*Vector) uint64
}

type Plane struct {
	Coefs      *Vector
	InnerPoint *Vector
}

// Add in the main code: rand.Seed(time.Now().UnixNano())
type LSHIndex struct {
	dims    int
	bias    float64
	nPlanes int
	Planes  []Plane
}
