#!/bin/bash
# A "server" that dies immediately, as llama.cpp does on a corrupt or missing model.
echo "error: failed to load model '$*'" >&2
exit 1
