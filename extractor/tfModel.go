package extractor

import (
	"bytes"
	"io/ioutil"

	// "os"

	// tensorflow part
	tf "github.com/tensorflow/tensorflow/tensorflow/go"
	"github.com/tensorflow/tensorflow/tensorflow/go/op"
)

const (
	H, W  = 224, 224
	Mean  = float32(117)
	Scale = float32(1)
)

// Holds preprocessing graph and input/output nodes
type TfPreprocessor struct {
	Graph         *tf.Graph
	Input, Output tf.Output
}

// Main data structure for holding model's graph
type TfModel struct {
	Graph *tf.Graph
	Prep  TfPreprocessor
}

// Preprocessing pipeline
func (tfprep *TfPreprocessor) MakeTransformImageGraph() error {
	var err error
	s := op.NewScope()
	tfprep.Input = op.Placeholder(s, tf.String)
	decode := op.DecodeJpeg(s, tfprep.Input, op.DecodeJpegChannels(3))
	// Div and Sub perform (value-Mean)/Scale for each pixel
	tfprep.Output = op.Div(s,
		op.Sub(s,
			// Resize to 224x224 with bilinear interpolation
			op.ResizeBilinear(s,
				// Create a batch containing a single image
				op.ExpandDims(s,
					// Use decoded pixel values
					op.Cast(s, decode, tf.Float),
					op.Const(s.SubScope("make_batch"), int32(0))),
				op.Const(s.SubScope("size"), []int32{H, W})),
			op.Const(s.SubScope("mean"), Mean)),
		op.Const(s.SubScope("scale"), Scale))
	tfprep.Graph, err = s.Finalize()
	if err != nil {
		return err
	}
	return nil
}

// Converts bytearray into the tf 4D tensor
func (tfprep *TfPreprocessor) MakeImagesBatch(images [][]byte) (*tf.Tensor, error) {
	session, err := tf.NewSession(tfprep.Graph, nil)
	if err != nil {
		return nil, err
	}
	defer session.Close()

	var buf bytes.Buffer
	for _, img := range images {
		tensor, err := tf.NewTensor(string(img))
		if err != nil {
			return nil, err
		}
		normalized, err := session.Run(
			map[tf.Output]*tf.Tensor{tfprep.Input: tensor},
			[]tf.Output{tfprep.Output},
			nil)
		if _, err := normalized[0].WriteContentsTo(&buf); err != nil {
			return nil, err
		}
	}

	batchShape := []int64{int64(len(images)), H, W, 3}
	batch, err := tf.ReadTensor(tf.Float, batchShape, &buf)
	if err != nil {
		return nil, err
	}
	return batch, nil
}

// Load tensorflow model from frozen graph
func (tfmodel *TfModel) LoadModel(path string) error {
	model, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	tfmodel.Graph = tf.NewGraph()
	if err := tfmodel.Graph.Import(model, ""); err != nil {
		return err
	}

	if err := tfmodel.Prep.MakeTransformImageGraph(); err != nil {
		return err
	}
	return nil
}

// Get feature vector
func (tfmodel *TfModel) Predict(images *[][]byte) ([][]float32, error) {
	imagesBatch, err := tfmodel.Prep.MakeImagesBatch(*images)
	if err != nil {
		return nil, err
	}

	// Create a session for inference over graph.
	session, err := tf.NewSession(tfmodel.Graph, nil)
	if err != nil {
		return nil, err
	}
	defer session.Close()

	output, err := session.Run(
		map[tf.Output]*tf.Tensor{
			// tfmodel.Graph.Operation("images").Output(0): imagesBatch,
			// tfmodel.Graph.Operation("hub_input/images").Output(0): normalized[0],
			tfmodel.Graph.Operation("input").Output(0): imagesBatch,
		},
		[]tf.Output{
			// tfmodel.Graph.Operation("pooled_features").Output(0), // batch_sizex1x1280
			// tfmodel.Graph.Operation("MobilenetV1/Logits/AvgPool_1a/AvgPool/ReadForQuantize").Output(0),
			tfmodel.Graph.Operation("output").Output(0), // batch_sizex1x1280
		},
		nil)
	if err != nil {
		return nil, err
	}

	return output[0].Value().([][]float32), nil
}
