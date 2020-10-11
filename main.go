package main

import (
	"fmt"

	// tensorflow part
	tf "github.com/tensorflow/tensorflow/tensorflow/go"
	"github.com/tensorflow/tensorflow/tensorflow/go/op"

	// db stuff
	// "github.com/boltdb/bolt"

	// internal packages
	"github.com/gasparian/visual-search-go/extractor"
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

	// test downloaded pretrained models
	var tfgraph extractor.TfModel
	if err := tfgraph.LoadModel("/model/frozen_graph.pb"); err != nil {
		panic(err)
	}

	// var tfgraph extractor.TfModel
	// if err := tfgraph.LoadModel("/model", "train"); err != nil {
	// 	panic(err)
	// }

	// tfmodel, err := LoadModel("/model/saved_model.pb", "train")
	// if err != nil {
	// 	panic(err)
	// }

	// // checking the boltdb
	// db, err := bolt.Open("./db-dump/my.db", 0600, &bolt.Options{Timeout: 1 * time.Second})
	// if err != nil {
	// 	panic(err)
	// }
	// fmt.Println("Bolt db created successfully!")
	// defer db.Close()
}
