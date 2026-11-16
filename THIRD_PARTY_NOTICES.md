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

## Demucs and PyTorch (optional, vocal isolation)

Only installed when the ASR image is built with `VOCAL_ISOLATION=1`
(ADR-0013); the default image contains none of this.

- **Demucs 4.0.1** (https://github.com/facebookresearch/demucs): MIT License,
  Copyright (c) Meta Platforms, Inc. and affiliates.
- **`htdemucs` pretrained weights:** downloaded from the Demucs project at image
  build time and baked into the image, never at response time, and never
  committed. The Demucs repository states it is released under the MIT license;
  no separate weights licence was found, and it has not been confirmed
  independently. Verify it before distributing an image that contains them.
- **PyTorch / torchaudio** (CPU wheels): BSD-3-Clause style licences. The image
  build also installs `ffmpeg` (LGPL/GPL, from Debian).

## Python worker dependencies

The ASR worker in `asr/worker` installs pinned versions of `numpy`,
`sherpa-onnx`, and `sherpa-onnx-core` at image build time. Their license texts
are available from their published package distributions.
