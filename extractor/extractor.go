package extractor

import (
	"io/ioutil"
	// "os"

	// tensorflow part
	tf "github.com/tensorflow/tensorflow/tensorflow/go"
	// "github.com/tensorflow/tensorflow/tensorflow/go/op"
)

type TfModel struct {
	Graph *tf.Graph
}

func (tfgraph *TfModel) LoadModel(path string) error {
	model, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	tfgraph.Graph = tf.NewGraph()
	if err := tfgraph.Graph.Import(model, ""); err != nil {
		return err
	}
	return nil
}
