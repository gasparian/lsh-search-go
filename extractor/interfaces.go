package extractor

type ITfModel interface {
	LoadModel() error
	Predict([][]byte) ([][]float32, error)
}
