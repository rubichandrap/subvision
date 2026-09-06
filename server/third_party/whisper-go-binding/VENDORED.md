# Vendored whisper.cpp Go binding

This directory carries the Go binding (`bindings/go`) of the whisper.cpp
release the repo vendors as a submodule at `server/third_party/whisper.cpp`
(v1.9.3). The submodule stays the source of the C library —
`server/scripts/dev.sh` and the Dockerfile build `libwhisper.a` from it —
while this copy is what the server actually compiles against:

```
replace github.com/ggerganov/whisper.cpp/bindings/go => ./third_party/whisper-go-binding
```

The module path is unchanged, so imports keep reading
`github.com/ggerganov/whisper.cpp/bindings/go/...`.

## Why it is vendored

ADR-0007 (issue #25) extends the binding's cgo shim in place rather than
adding a new binding or forking whisper.cpp; the extension must live inside
the binding's own packages (the cgo param struct is opaque outside them),
which a submodule checkout cannot carry as a committable change. Hence this
copy.

The submodule runs v1.9.3, bumped from `d1f114da` during #27: upstream added
the VAD binding surface (`SetVAD`, `SetVADModelPath`, the per-parameter VAD
setters) and — with PR #3910 — token-level VAD timestamp remapping that
`d1f114da` lacked. Without the latter, VAD-gated word timings stay on the
filtered (compressed) timeline while segment times are remapped.

## Extensions over upstream v1.9.3

- `whisper.go`: `Context.Whisper_full_get_token_t0/t1` — cgo wrappers for
  whisper.cpp's token remap getters.
- `pkg/whisper/context.go`: `toTokens` reads token times through those
  getters, so Token times sit on the original audio timeline under VAD (and
  are the raw timestamps without it).

Keep extensions minimal and localized so a vendor upgrade stays feasible:
re-copy the binding from the new upstream commit and re-apply the extension
list above (issue #27).

## Not carried from `bindings/go`

`Makefile`, `examples/`, `samples/`, `models/`, and `build_go/` only work
inside the whisper.cpp tree (the Makefile builds the C library relative to
it; models are large binaries the submodule/compose mount already holds).
The binding's own tests reference those paths and run from the submodule
checkout, not from here.
