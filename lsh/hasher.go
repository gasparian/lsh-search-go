package lsh

import (
	"bytes"
	"encoding/gob"
	"errors"
	"gonum.org/v1/gonum/blas/blas64"
	"math"
	"math/rand"
	"sort"
	"sync"
	"time"
)

var (
	dimensionsNumberErr     = errors.New("dimensions number must be a positive integer")
	hasherEmptyInstancesErr = errors.New("hasher must contain at least one instance")
)

// plane struct holds data needed to work with plane
type plane struct {
	n blas64.Vector
	d float64
}

func (p *plane) getProductSign(vec blas64.Vector) bool {
	prod := blas64.Dot(vec, p.n) - p.d
	prodSign := math.Signbit(prod) // NOTE: returns true if product < 0
	return prodSign
}

// treeNode holds binary tree with generated planes
type treeNode struct {
	left  *treeNode
	right *treeNode
	plane *plane
}

func traverse(node *treeNode, hash uint64, inpVec blas64.Vector, depth int) uint64 {
	if node == nil || node.plane == nil {
		return hash
	}
	vec := NewVec(make([]float64, inpVec.N))
	blas64.Copy(inpVec, vec)
	prodSign := node.plane.getProductSign(vec)
	if !prodSign {
		return traverse(node.right, hash, inpVec, depth+1)

	}
	hash |= (1 << depth)
	return traverse(node.left, hash, inpVec, depth+1)
}

// getHash calculates LSH code
func (node *treeNode) getHash(inp []float64) uint64 {
	inpVec := NewVec(inp)
	var hash uint64
	return traverse(node, hash, inpVec, 0)
}

type HasherConfig struct {
	NTrees          int
	KMinVecs        int
	Dims            int
	isAngularMetric bool
}

// Hasher holds N_PERMUTS number of trees
type Hasher struct {
	mutex  sync.RWMutex
	Config HasherConfig
	trees  []*treeNode
}

func NewHasher(config HasherConfig) *Hasher {
	return &Hasher{
		Config: config,
		trees:  make([]*treeNode, config.NTrees),
	}
}

// SafeHashesHolder allows to lock map while write values in it
type safeHashesHolder struct {
	sync.Mutex
	v map[int]uint64
}

// planeByPoints generates random coefficients of a plane by given pair of points
func planeByPoints(a, b blas64.Vector) *plane {
	ndims := a.N
	planeCoefs := &plane{}
	points := []blas64.Vector{a, b}
	sort.Slice(points, func(i, j int) bool {
		return blas64.Nrm2(points[i]) < blas64.Nrm2(points[j])
	})
	centerPoint := NewVec(make([]float64, ndims))
	for _, p := range points {
		blas64.Axpy(0.5, p, centerPoint)
	}
	planeCoefs.n = NewVec(make([]float64, ndims))
	blas64.Copy(points[1], planeCoefs.n)
	blas64.Axpy(-1.0, centerPoint, planeCoefs.n)
	planeCoefs.d = blas64.Dot(centerPoint, planeCoefs.n)
	return planeCoefs
}

func getRandomPlane(vecs [][]float64, isAngular bool) *plane {
	randIndeces := make(map[int]bool)
	randIndecesList := make([]int, 2)
	var i int = 0
	maxPoints := 2
	if isAngular {
		maxPoints = 1
	}
	for i < maxPoints {
		idx := rand.Intn(len(vecs))
		if _, has := randIndeces[idx]; !has {
			randIndeces[idx] = true
			randIndecesList[i] = idx
			i++
		}
	}
	vec1 := NewVec(vecs[randIndecesList[0]])
	if isAngular { // NOTE: in case of angular metric we generate plane that always should intersect the origin
		oppositeVec := NewVec(make([]float64, vec1.N))
		blas64.Axpy(-1.0, vec1, oppositeVec)
		return planeByPoints(vec1, oppositeVec)
	}
	vec2 := NewVec(vecs[randIndecesList[1]])
	return planeByPoints(vec1, vec2)
}

// growTree ...
func growTree(vecs [][]float64, node *treeNode, depth, k int, isAngular bool) {
	if depth > 63 || len(vecs) < 2 { // NOTE: depth <= 63 since we will use 8 byte int to store a hash
		return
	}
	node.plane = getRandomPlane(vecs, isAngular)
	var l, r [][]float64
	for _, v := range vecs {
		inpVec := NewVec(v)
		sign := node.plane.getProductSign(inpVec)
		if !sign {
			r = append(r, v)
			continue
		}
		l = append(l, v)
	}
	depth++
	if len(r) > k {
		node.right = &treeNode{}
		growTree(r, node.right, depth, k, isAngular)
	}
	if len(l) > k {
		node.left = &treeNode{}
		growTree(l, node.left, depth, k, isAngular)
	}
}

// buildTree creates set of planes which will be used to calculate hash
func buildTree(vecs [][]float64, kmin int, isAngular bool) *treeNode {
	rand.Seed(time.Now().UnixNano())
	tree := &treeNode{}
	growTree(vecs, tree, 0, kmin, isAngular)
	return tree
}

// build method creates the hasher instances
func (hasher *Hasher) build(vecs [][]float64) {
	hasher.mutex.Lock()
	defer hasher.mutex.Unlock()

	trees := make([]*treeNode, hasher.Config.NTrees)
	kmin := hasher.Config.KMinVecs
	isAngular := hasher.Config.isAngularMetric
	wg := sync.WaitGroup{}
	wg.Add(len(trees))
	for i := 0; i < hasher.Config.NTrees; i++ {
		go func(i int, wg *sync.WaitGroup) {
			defer wg.Done()
			tmpTree := buildTree(vecs, kmin, isAngular)
			trees[i] = tmpTree
		}(i, &wg)
	}
	wg.Wait()
	hasher.trees = trees
}

// getHashes returns map of calculated lsh values for a given vector
func (hasher *Hasher) getHashes(vec []float64) map[int]uint64 {
	hasher.mutex.RLock()
	defer hasher.mutex.RUnlock()

	hashes := &safeHashesHolder{v: make(map[int]uint64)}
	wg := sync.WaitGroup{}
	wg.Add(len(hasher.trees))
	for i, tree := range hasher.trees {
		go func(i int, tree *treeNode, hashes *safeHashesHolder) {
			defer wg.Done()
			hashes.Lock()
			hashes.v[i] = tree.getHash(vec)
			hashes.Unlock()
		}(i, tree, hashes)
	}
	wg.Wait()
	return hashes.v
}

// dump encodes Hasher object as a byte-array
func (hasher *Hasher) dump() ([]byte, error) {
	hasher.mutex.RLock()
	defer hasher.mutex.RUnlock()

	if len(hasher.trees) == 0 {
		return nil, hasherEmptyInstancesErr
	}
	buf := &bytes.Buffer{}
	enc := gob.NewEncoder(buf)
	err := enc.Encode(hasher)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// load loads Hasher struct from the byte-array file
func (hasher *Hasher) load(inp []byte) error {
	hasher.mutex.Lock()
	defer hasher.mutex.Unlock()

	buf := &bytes.Buffer{}
	buf.Write(inp)
	dec := gob.NewDecoder(buf)
	err := dec.Decode(&hasher)
	if err != nil {
		return err
	}
	return nil
}
