package algorithm

type Vector struct {
	Values []float64
	Size   int
}

type IVector interface {
	DotProd(*Vector) float64
	L2(*Vector) float64
	CosineSim(*Vector) float64
}

// rand.Seed(time.Now().UnixNano())
type RandomPlaneGenerator struct {
	dims int
	bias float64
}

type ILSHIndex interface {
	Build()
	Dump(string)
	Load(string)
	GetHash(*Vector)
}

type LSHIndex struct {
	RandomPlaneGenerator
	nPlanes int
	Planes  []Vector
}
