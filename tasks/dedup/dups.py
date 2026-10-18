import os
NORM = os.environ.get('NORM') == '1'
import subprocess, sys, re, hashlib, collections
W = int(sys.argv[1]) if len(sys.argv) > 1 else 8
files = subprocess.check_output(['git','ls-files','internal','resources/ts','cmd'], text=True).split('\n')
new = set(subprocess.check_output(['git','diff','--diff-filter=A','--name-only','origin/dev...HEAD'], text=True).split('\n'))
def keep(f):
    if not re.search(r'\.(go|tsx?)$', f): return False
    if re.search(r'_test\.go$|\.test\.tsx?$|\.d\.ts$|/gen/|generated|internal/db/sqlc/|\.gen\.', f): return False
    return True
trivial = re.compile(r'^([\{\}\(\)\]\[;,]|\}\)|\}\)?;?|\)|return|else|\} else \{|import .*|package .*|"[^"]*"|//.*|/\*.*)$')
idx = collections.defaultdict(set)   # hash -> {(file,line)}
meta = {}
for f in files:
    if not keep(f): continue
    try: lines = open(f, encoding='utf-8').read().split('\n')
    except Exception: continue
    seq = []
    for i, l in enumerate(lines, 1):
        s = ' '.join(l.split())
        if NORM:
            s = re.sub(r'"[^"]*"|`[^`]*`', 'S', s)
            s = re.sub(r'[A-Za-z_][A-Za-z0-9_]*', 'x', s)
            s = re.sub(r'[0-9]+', '0', s)
        if not s or trivial.match(s): continue
        seq.append((i, s))
    meta[f] = len(seq)
    for k in range(len(seq) - W + 1):
        h = hashlib.md5('\n'.join(s for _, s in seq[k:k+W]).encode()).hexdigest()
        idx[h].add((f, seq[k][0]))
pair = collections.Counter(); covered = collections.defaultdict(set)
for h, occ in idx.items():
    fs = {f for f, _ in occ}
    if len(fs) < 2: continue
    for f, ln in occ: covered[f].add(ln)
    fl = sorted(fs)
    for a in range(len(fl)):
        for b in range(a+1, len(fl)): pair[(fl[a], fl[b])] += 1
tot = sum(meta.values()); dup = sum(len(v) for v in covered.values())
print(f'window={W} meaningful lines={tot} window-starts in cross-file clones={dup} ({100*dup/tot:.1f}%)')
print('--- top file pairs (shared windows ~ duplicated lines) ---')
for (a, b), n in pair.most_common(35):
    print(f'{n:4d} {"N" if a in new else " "}{"N" if b in new else " "} {a}  <->  {b}')
print('--- files by cloned share ---')
rows = sorted(((len(covered[f])/max(meta[f],1), len(covered[f]), meta[f], f) for f in covered if meta[f] > 60), reverse=True)[:20]
for r, n, m, f in rows: print(f'{100*r:5.1f}% {n:4d}/{m:<5d} {"N" if f in new else " "} {f}')
