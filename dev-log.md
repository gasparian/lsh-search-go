### Dev log. Some notes on developing the project.  

#### Useful links:  

- [Tf Hub models](https://tfhub.dev/s?module-type=image-augmentation,image-classification,image-classification-logits,image-classifier,image-feature-vector,image-generator,image-object-detection,image-others,image-pose-detection,image-segmentation,image-style-transfer,image-super-resolution,image-rnn-agent)
- [Tf frozen models](https://www.tensorflow.org/lite/guide/hosted_models)
- [https://outcrawl.com/image-recognition-api-go-tensorflow](https://outcrawl.com/image-recognition-api-go-tensorflow)
- [https://blog.gopheracademy.com/advent-2017/tensorflow-and-go/](https://blog.gopheracademy.com/advent-2017/tensorflow-and-go/)
- [https://github.com/tinrab/go-tensorflow-image-recognition](https://github.com/tinrab/go-tensorflow-image-recognition)
- [https://www.tensorflow.org/api_docs/python/tf/keras/applications/EfficientNetB0](https://www.tensorflow.org/api_docs/python/tf/keras/applications/EfficientNetB0)
- [https://tfhub.dev/google/collections/efficientnet/1](https://tfhub.dev/google/collections/efficientnet/1)
- [https://gist.github.com/monklof/24597be7af323f9cb7c4f8f0caca52e6](https://gist.github.com/monklof/24597be7af323f9cb7c4f8f0caca52e6)

#### Log:  

[12.10.2020] Moving really slow, since working with tensorflow again is a pain: tf-hub provides models in "saved_model" format, which should be changed to "frozen_graph" to be able to run it easily in go. I've found a solution, but ideally pre-trained tf models should be already provided as frozen graphs. I did the conversion manually, because I need feature vectors, but not the predictions themselfs ;)  

[]  
