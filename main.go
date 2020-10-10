package main

import (
    "time"
    "fmt"

    // tensorflow part
    tf "github.com/tensorflow/tensorflow/tensorflow/go"
    "github.com/tensorflow/tensorflow/tensorflow/go/op"

    // db stuff
    "github.com/boltdb/bolt"

    // internal packages
    "github.com/gasparian/visual-search-go/extractor"
)

func main() {
    // Construct a graph with an operation that produces a string constant.
    s := op.NewScope()
    c := op.Const(s, "Hello from TensorFlow version " + tf.Version())
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
    var graph *extractor.TfModel
    err := graph.loadModel("/model/tensorflow_inception_graph.pb")
    if err != nil {
        panic(err)
    }
    ops := graph.Operations()

    // checking the boltdb
    db, err := bolt.Open("./db-dump/my.db", 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		panic(err)
    }
    fmt.Println("Bolt db created successfully!")
	defer db.Close()
}