#!/usr/bin/env python3
"""Extract shell definitions verbatim from the plan document.

Two modes:
  fn   NAME   -> the function definition whose body opens with `NAME() (` or `NAME() {`
  block MARKER -> the fenced ```bash block containing MARKER
Exits 3 when the requested item is not present, so an absent function in an
older revision is reported as absent rather than silently producing nothing.
"""
import sys, re, pathlib

plan = pathlib.Path(sys.argv[1]).read_text().split('\n')
mode, target = sys.argv[2], sys.argv[3]

if mode == 'fn':
    for opener, closer in (('%s() (' % target, ')'), ('%s() {' % target, '}')):
        for i, line in enumerate(plan):
            if line == opener:
                for j in range(i + 1, len(plan)):
                    if plan[j] == closer:
                        sys.stdout.write('\n'.join(plan[i:j + 1]) + '\n')
                        sys.exit(0)
    sys.exit(3)

if mode == 'block':
    start = None
    for i, line in enumerate(plan):
        if line.startswith('```'):
            if start is None:
                start = i
            else:
                body = plan[start + 1:i]
                if any(target in b for b in body):
                    sys.stdout.write('\n'.join(body) + '\n')
                    sys.exit(0)
                start = None
    sys.exit(3)

sys.exit(2)
