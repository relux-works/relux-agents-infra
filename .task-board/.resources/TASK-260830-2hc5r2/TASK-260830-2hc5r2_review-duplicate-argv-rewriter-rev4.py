import os
import sys


port, model, mode = sys.argv[1:4]
os.execv(
    sys.executable,
    [
        sys.executable,
        "/Users/alexis/src/relux-works/relux-agents-infra/.temp/STORY-260830-2vrhg1/worktree/.temp/TASK-260830-2hc5r2-review/benchmark-gate-out/fake-runtime.py",
        port,
        model,
        mode,
        "--host",
        "127.0.0.1",
        "--model",
        model,
        "--max-kv-size",
        "76800",
        "--prefill-step-size",
        "2048",
        "--prefill-step-size",
        "999",
        "--reasoning-effort",
        "medium",
    ],
)
