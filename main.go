package main

import (
	"fmt"

	// tensorflow part
	tf "github.com/tensorflow/tensorflow/tensorflow/go"
	"github.com/tensorflow/tensorflow/tensorflow/go/op"
	// internal packages
)

func main() {
	// Construct a graph with an operation that produces a string constant.
	s := op.NewScope()
	c := op.Const(s, "Hello from TensorFlow version "+tf.Version())
	graph, err := s.Finalize()
	if err != nil {
		panic(err)
	}
	// Execute the graph in a session.
	sess, err := tf.NewSession(graph, nil)
	if err != nil {
		panic(err)
	}
	output, err := sess.Run(nil, []tf.Output{c}, nil)
	if err != nil {
		panic(err)
	}
	fmt.Println(output[0].Value())
	///////////////////////////////////////////////////////////////////////

	// Test downloaded pretrained models
	savedModel, err := tf.LoadSavedModel("./model/efficientnet", []string{"serve"}, nil)
	if err != nil {
		panic(err)
	}
	if op := savedModel.Graph.Operation("images"); op == nil {
		fmt.Printf("not found in graph")
	}
	fmt.Printf("SavedModel: %+v\n", savedModel)
	// for k := range savedModel.Signatures {
	// 	fmt.Println(k)
	// }

	// var tfmodel extractor.TfModel
	// if err := tfmodel.LoadModel("./model/inception_frozen_graph.pb"); err != nil {
	// 	panic(err)
	// }
	// ops := tfmodel.Graph.Operations()
	// for i, op := range ops {
	// 	if i <= 5 || i >= (len(ops)-5) {
	// 		fmt.Println(op.Name())
	// 	}
	// }

	// img1, err := ioutil.ReadFile("./test_data/754c2ca07fa87ce18b6fa0249f8a07b24f50c36a.jpg")
	// if err != nil {
	// 	panic(err)
	// }
	// img2, err := ioutil.ReadFile("./test_data/e65b4de000f7db3d77b9919d018171f1fcfc8a79.jpg")
	// if err != nil {
	// 	panic(err)
	// }
	// images := [][]byte{img1, img2}
	// fmt.Println("Images has been read: " + strconv.Itoa(len(images)))

	// features, err := tfmodel.Predict(&images)
	// if err != nil {
	// 	panic(err)
	// }
	// fmt.Println(features)
	///////////////////////////////////////////////////////////////////////
}
