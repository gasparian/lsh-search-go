package extractor

import (
	"io/ioutil"
	// "os"

    // tensorflow part
    tf "github.com/tensorflow/tensorflow/tensorflow/go"
    // "github.com/tensorflow/tensorflow/tensorflow/go/op"
)

type TfModel struct {
	graph *tf.Graph
}

func (tfgraph *TfModel) loadModel(path string) error {
  model, err := ioutil.ReadFile(path)
  if err != nil {
    return err
  }
  tfgraph.graph = tf.NewGraph()
  if err := tfgraph.graph.Import(model, ""); err != nil {
    return err
  }
  return nil
}

