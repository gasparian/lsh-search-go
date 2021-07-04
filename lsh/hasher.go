package lsh

import (
	"bytes"
	"encoding/gob"
	"errors"
	"gonum.org/v1/gonum/blas/blas64"
	"math"
	"math/rand"
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
	// vec := NewVec(make([]float64, inpVec.N))
	// blas64.Copy(inpVec, vec)
	prodSign := node.plane.getProductSign(inpVec)
	if !prodSign {
		return traverse(node.right, hash, inpVec, depth+1)

	}
	hash |= (1 << depth)
	return traverse(node.left, hash, inpVec, depth+1)
}

// getHash calculates LSH code
func (node *treeNode) getHash(vec blas64.Vector) uint64 {
	var hash uint64
	return traverse(node, hash, vec, 0)
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
func planeByPoints(points []blas64.Vector, ndims int) *plane {
	planeCoefs := &plane{}
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
	randVecs := make([]blas64.Vector, 2)
	norms := make([]float64, 2)
	ndims := len(vecs[0])
	var i int = 0
	maxPoints := 2
	for i < maxPoints && i < len(vecs)*3 {
		idx := rand.Intn(len(vecs))
		if _, has := randIndeces[idx]; !has {
			randIndeces[idx] = true
			randVecs[i] = NewVec(vecs[idx])
			norms[i] = blas64.Nrm2(randVecs[i])
			i++
		}
	}
	if norms[0] > norms[1] {
		randVecs[0], randVecs[1] = randVecs[1], randVecs[0]
		norms[0], norms[1] = norms[1], norms[0]
	}
	// NOTE: normilize vectors when dealing with angular distance metric (not sure)
	if isAngular {
		normedVecs := make([]blas64.Vector, len(randVecs))
		for i, vec := range randVecs {
			normedVec := NewVec(make([]float64, ndims))
			norm := norms[i]
			if norm > tol {
				blas64.Axpy(1/norm, vec, normedVec)
			}
			normedVecs[i] = normedVec
		}
		return planeByPoints(normedVecs, ndims)
	}
	return planeByPoints(randVecs, ndims)
}

// growTree ...
func growTree(vecs [][]float64, node *treeNode, depth int, config HasherConfig) {
	if depth > 63 || len(vecs) < 2 { // NOTE: depth <= 63 since we will use 8 byte int to store a hash
		return
	}
	node.plane = getRandomPlane(vecs, config.isAngularMetric)
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
	if len(r) > config.KMinVecs {
		node.right = &treeNode{}
		growTree(r, node.right, depth, config)
	}
	if len(l) > config.KMinVecs {
		node.left = &treeNode{}
		growTree(l, node.left, depth, config)
	}
}

// buildTree creates set of planes which will be used to calculate hash
func buildTree(vecs [][]float64, config HasherConfig) *treeNode {
	rand.Seed(time.Now().UnixNano())
	tree := &treeNode{}
	growTree(vecs, tree, 0, config)
	return tree
}

// build method creates the hasher instances
func (hasher *Hasher) build(vecs [][]float64) {
	hasher.mutex.Lock()
	defer hasher.mutex.Unlock()

	trees := make([]*treeNode, hasher.Config.NTrees)
	wg := sync.WaitGroup{}
	wg.Add(len(trees))
	for i := 0; i < hasher.Config.NTrees; i++ {
		go func(i int, wg *sync.WaitGroup) {
			defer wg.Done()
			tmpTree := buildTree(vecs, hasher.Config)
			trees[i] = tmpTree
		}(i, &wg)
	}
	wg.Wait()
	hasher.trees = trees
}

// getHashes returns map of calculated lsh values for a given vector
func (hasher *Hasher) getHashes(inpVec []float64) map[int]uint64 {
	hasher.mutex.RLock()
	defer hasher.mutex.RUnlock()

	vec := NewVec(make([]float64, len(inpVec)))
	copy(vec.Data, inpVec)
	// NOTE: norm vector when using angular matric (since normed vectors has been used for planes generation in this case)
	if hasher.Config.isAngularMetric {
		normed := NewVec(make([]float64, len(inpVec)))
		norm := blas64.Nrm2(vec)
		if norm > tol {
			blas64.Axpy(1/norm, vec, normed)
			blas64.Copy(normed, vec)
		}
	}
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
