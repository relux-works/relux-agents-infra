import sys, re, os
plan = sys.argv[1]; outdir = sys.argv[2]
os.makedirs(outdir, exist_ok=True)
lines = open(plan).read().split('\n')
blocks=[]; cur=None
for i,l in enumerate(lines,1):
    if cur is None:
        m=re.match(r'^```(bash|zsh)\s*$', l)
        if m: cur=(m.group(1), i, [])
    else:
        if l.strip()=='```':
            blocks.append(cur); cur=None
        else:
            cur[2].append(l)
assert cur is None, "unterminated fence"
for n,(lang,start,body) in enumerate(blocks,1):
    p=os.path.join(outdir, f'block{n:02d}-line{start}.{lang}')
    open(p,'w').write('\n'.join(body)+'\n')
print(f'{len(blocks)} blocks')
