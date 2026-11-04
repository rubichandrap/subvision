# Third-party notices

Subvision includes or depends on the following third-party works. This file is
informational and does not replace the complete license texts shipped with each
dependency.

## NVIDIA Parakeet TDT 0.6B v3

- **Work:** NVIDIA Parakeet TDT 0.6B v3 (speech recognition model)
- **Source:** https://huggingface.co/nvidia/parakeet-tdt-0.6b-v3
- **Prepared form:** int8 ONNX export published as
  `csukuangfj/sherpa-onnx-nemo-parakeet-tdt-0.6b-v3-int8`
  (https://huggingface.co/csukuangfj/sherpa-onnx-nemo-parakeet-tdt-0.6b-v3-int8)
- **License:** Creative Commons Attribution 4.0 International (CC-BY-4.0)
  https://creativecommons.org/licenses/by/4.0/
- **Changes:** quantized to int8 ONNX for local inference with sherpa-onnx.
- **Attribution:** model architecture and weights by NVIDIA; ONNX export by the
  sherpa-onnx community. Subvision does not claim ownership of the model.

Model weights are not distributed in this repository. Deployments must fetch the
pinned files and retain this attribution.

## sherpa-onnx

- **Work:** sherpa-onnx runtime
- **Source:** https://github.com/k2-fsa/sherpa-onnx
- **License:** Apache License 2.0

## Python worker dependencies

The ASR worker in `asr/worker` installs pinned versions of `numpy`,
`sherpa-onnx`, and `sherpa-onnx-core` at image build time. Their license texts
are available from their published package distributions.
