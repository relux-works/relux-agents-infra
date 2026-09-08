# TASK-260830-1e9gse: port-existing-engines-onto-the-adapter

## Description
Move Python mlx-lm, llama.cpp and the MLX Swift prototype onto the adapter so each proves the contract rather than special-casing around it.

## Scope
(define task scope)

## Acceptance Criteria
Python mlx-lm, llama.cpp and the MLX Swift prototype all run through the adapter, and the comparison gate scores a pair without any engine-specific special case or relaxed admission clause.
