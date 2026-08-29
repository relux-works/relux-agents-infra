#!/usr/bin/env python3
"""Minimal GGUF v3 header/tensor-info reader. Reads only the file prefix, so it
works on a partially downloaded file."""
import collections
import json
import struct
import sys

GGML_TYPE = {0:'F32',1:'F16',2:'Q4_0',3:'Q4_1',6:'Q5_0',7:'Q5_1',8:'Q8_0',9:'Q8_1',
 10:'Q2_K',11:'Q3_K',12:'Q4_K',13:'Q5_K',14:'Q6_K',15:'Q8_K',16:'IQ2_XXS',17:'IQ2_XS',
 18:'IQ3_XXS',19:'IQ1_S',20:'IQ4_NL',21:'IQ3_S',22:'IQ2_S',23:'IQ4_XS',24:'I8',25:'I16',
 26:'I32',27:'I64',28:'F64',29:'IQ1_M',30:'BF16',34:'TQ1_0',35:'TQ2_0',39:'MXFP4'}

class R:
    def __init__(s, f): s.f=f
    def raw(s,n):
        b=s.f.read(n)
        if len(b)<n: raise EOFError
        return b
    def u32(s): return struct.unpack('<I', s.raw(4))[0]
    def u64(s): return struct.unpack('<Q', s.raw(8))[0]
    def i64(s): return struct.unpack('<q', s.raw(8))[0]
    def st(s):
        n=s.u64(); return s.raw(n).decode('utf-8','replace')
    def val(s, t):
        if t==0: return struct.unpack('<B', s.raw(1))[0]
        if t==1: return struct.unpack('<b', s.raw(1))[0]
        if t==2: return struct.unpack('<H', s.raw(2))[0]
        if t==3: return struct.unpack('<h', s.raw(2))[0]
        if t==4: return s.u32()
        if t==5: return struct.unpack('<i', s.raw(4))[0]
        if t==6: return struct.unpack('<f', s.raw(4))[0]
        if t==7: return bool(struct.unpack('<B', s.raw(1))[0])
        if t==8: return s.st()
        if t==9:
            et=s.u32(); n=s.u64()
            return [s.val(et) for _ in range(n)]
        if t==10: return s.u64()
        if t==11: return s.i64()
        if t==12: return struct.unpack('<d', s.raw(8))[0]
        raise ValueError(f'unknown value type {t}')

def main(path):
    with open(path,'rb') as f:
        return _read(R(f))

def _read(r):
    magic=r.raw(4)
    assert magic==b'GGUF', magic
    ver=r.u32(); n_tensors=r.u64(); n_kv=r.u64()
    kv={}
    for _ in range(n_kv):
        k=r.st(); t=r.u32(); kv[k]=r.val(t)
    tensors=[]
    for _ in range(n_tensors):
        name=r.st(); nd=r.u32()
        dims=[r.u64() for _ in range(nd)]
        tt=r.u32(); r.u64()
        tensors.append((name, dims, GGML_TYPE.get(tt, f'TYPE_{tt}')))
    out={'version':ver,'n_tensors':n_tensors,'n_kv':n_kv,
         'kv':{k:(v if not (isinstance(v,list) and len(v)>8) else [f'<{len(v)} items>']+v[:4]) for k,v in kv.items()},
         'tensors':[{'name':n,'dims':d,'type':t} for n,d,t in tensors]}
    print(json.dumps(out, indent=1)[:1] and '', end='')
    return out

if __name__=='__main__':
    o=main(sys.argv[1])
    mode=sys.argv[2] if len(sys.argv)>2 else 'summary'
    if mode=='json':
        print(json.dumps(o, indent=1))
    else:
        print(f"gguf_version={o['version']} n_tensors={o['n_tensors']} n_kv={o['n_kv']}")
        print("--- metadata ---")
        for k,v in sorted(o['kv'].items()):
            sv=repr(v)
            if len(sv)>200: sv=sv[:200]+'...'
            print(f"  {k:<45} {sv}")
        print("--- tensor type histogram ---")
        c=collections.Counter(t['type'] for t in o['tensors'])
        for t,n in c.most_common(): print(f"  {t:<8} {n}")
        import re
        print("--- per-name-pattern types ---")
        pat=collections.Counter((re.sub(r'\.\d+\.','.N.',t['name']), t['type']) for t in o['tensors'])
        for (p,t),n in sorted(pat.items()): print(f"  {n:4d}  {t:<8} {p}")
