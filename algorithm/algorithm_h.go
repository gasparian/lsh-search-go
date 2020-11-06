package algorithm

type IVector interface {
	DotProd(*Vector) float64
	L2(*Vector) float64
	CosineSim(*Vector) float64
}

type Vector struct {
	Values []float64
	Size   int
}

type ILSHIndex interface {
	Build() error
	Dump(string)
	Load(string)
	GetHash(*Vector) uint64
}

// rand.Seed(time.Now().UnixNano())
type LSHIndex struct {
	dims    int
	bias    float64
	nPlanes int
	Planes  []Vector
}
