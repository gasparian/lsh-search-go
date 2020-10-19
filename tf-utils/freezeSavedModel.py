'''
NOTE: 
This code helps to convert tf1 saved model to frozen graph to be able to use in golang
Based on tf-tools file: import_pb_to_tensorboard.py

Dowbload pretrained model:  
   wget "https://tfhub.dev/google/efficientnet/b0/feature-vector/1?tf-hub-format=compressed" \
        -O /tmp/efficientnet_b0_feature-vector_1.tar.gz && \
   tar -C /tmp -zxvf /tmp/efficientnet_b0_feature-vector_1.tar.gz && \
   rm /tmp/efficientnet_b0_feature-vector_1.tar.gz

Usage:  
    python ./tf-utils/freezeSavedModel.py --model_dir=/tmp/saved_model.pb --output_dir=/model
'''

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

import argparse
import sys
import os

import tensorflow as tf
from tensorflow.python.platform import app
from tensorflow.python.platform import gfile
from tensorflow.core.protobuf import saved_model_pb2
from tensorflow.python.util import compat

def import_to_tensorboard(unused_args):
    model_dir, output_dir = FLAGS.model_dir, FLAGS.output_dir
    with tf.Session() as sess:
        with gfile.FastGFile(model_dir, 'rb') as f:
            data = compat.as_bytes(f.read())
            sm = saved_model_pb2.SavedModel()
            sm.ParseFromString(data)
            # print([n.name for n in sm.meta_graphs[0].graph_def.node]) 

            # Dump frozen graph
            # Using first meta_graph by default
            with tf.gfile.GFile(os.path.join(output_dir, "frozen_graph.pb"), 'wb') as of:
                of.write(sm.meta_graphs[0].graph_def.SerializeToString()) 

if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.register("type", "bool", lambda v: v.lower() == "true")
    parser.add_argument(
        "--model_dir",
        type=str,
        default="",
        required=True)
    parser.add_argument(
        "--output_dir",
        type=str,
        default="",
        required=True)
    FLAGS, unparsed = parser.parse_known_args()
    app.run(main=import_to_tensorboard, argv=[sys.argv[0]] + unparsed)