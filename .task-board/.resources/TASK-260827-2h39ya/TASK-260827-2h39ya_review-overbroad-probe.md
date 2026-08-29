# TASK-260827-2h39ya adversarial classifier probe

## Purpose

Attack the classifier through the composed Release binary and real
`model-harness`, using a semantically request-scoped failure that collides with
the generic fatal substring `Resource limit`.

## Environment

- Reviewer run: `RUN-260827-79a047`
- Platform: macOS arm64
- Runtime: freshly rebuilt
  `DerivedData/Build/Products/Release/mlx-swift-runtime-prototype`
- Harness: `model-harness v1.6.1-44-gd91d6fc`
- Model: cached `mlx-community/Qwen1.5-0.5B-Chat-4bit`
- Loopback port: `18030`

## Fault

```text
--fault-inject-generation-error "RequestError: Resource limit for this request is 8 tokens"
```

The message intentionally contains no `metal::malloc` or metallib evidence.

## Observed result

```text
ready=1
health_before=200 chat=500 health_after=503 models_after=503
health_after_body={"detail":"RequestError: Resource limit for this request is 8 tokens","status":"unavailable"}
```

Runtime output:

```json
{"detail":"RequestError: Resource limit for this request is 8 tokens","event":"generation_worker_failed","marker":"generation_worker_unavailable"}
```

The harness and runtime were stopped in the same bounded command; port 18030
was not left occupied.

## Conclusion

The current OR-of-substrings classifier makes generic `Resource limit`
independently fatal. This defeats the claimed request-scoped negative boundary
and would trigger supervised recovery for a non-backend failure.
