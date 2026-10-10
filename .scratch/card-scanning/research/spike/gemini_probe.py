"""Gemini one-off probes.  uv run gemini_probe.py small | burst
small: 5 images single vs one batchEmbedContents call (order, equality, speed, tokens) + 3072 vs 1536 truncation check.
burst: ONE batchEmbedContents call with 101 tiny images (> 100 RPM): 429 => counts per image; 200 => counts per call.
"""

import io
import json
import sys
import time

import numpy as np
from PIL import Image

import common as c
from providers import gemini as g

c.load_env()
cards = c.catalog()
which = sys.argv[1]
G = g.Gemini()

if which == "small":
    items = [c.Item(k.id, c.catalog_image(k), "image/png") for k in cards[:5]]
    singles = [G.embed_one(i) for i in items]
    t = time.perf_counter()
    batch = G.embed_batch_sync(items)
    bt = time.perf_counter() - t
    print("dim", [len(s.vec) for s in singles][:1], "norm", float(np.linalg.norm(singles[0].vec)))
    print("single latency", [round(s.latency_s, 3) for s in singles], "tokens", [s.usage for s in singles])
    print("batch call wall", round(bt, 3), "per-image", round(bt / 5, 3), "batch usage/img", batch[0].usage)
    print("single-vs-batch cosine (order check)", [round(float(a.vec @ b.vec), 5) for a, b in zip(singles, batch)])
    print("cross cosine matrix offdiag max", float(np.max(np.array([s.vec for s in singles]) @ np.array([s.vec for s in singles]).T - np.eye(5))))
    G3 = g.Gemini(3072)
    full = [G3.embed_one(i) for i in items[:3]]
    for s, f in zip(singles, full):
        t15 = f.vec[:1536] / np.linalg.norm(f.vec[:1536])
        t768 = f.vec[:768] / np.linalg.norm(f.vec[:768])
        print("3072 len", len(f.vec), "cos(server1536, trunc3072->1536)", round(float(s.vec @ t15), 5),
              "cos(trunc 768 vs 1536 vecs)", round(float(t768 @ (s.vec[:768] / np.linalg.norm(s.vec[:768]))), 5))
    print("requests sent", G.calls + G3.calls)
else:
    def thumb(b):
        im = Image.open(io.BytesIO(b)).convert("RGB")
        im.thumbnail((192, 192))
        o = io.BytesIO()
        im.save(o, "JPEG", quality=70)
        return o.getvalue()

    # server caps one batch at 100 requests, so: 100-image batch, then immediately 3 single requests (101st-103rd
    # in the same minute). 429 on the singles => batch counted per image; success => counted per call (or not at all).
    items = [c.Item(k.id, thumb(c.catalog_image(k)), "image/jpeg") for k in cards[:103]]
    t0 = time.time()
    out = G.embed_batch_sync(items[:100])
    print("batch of 100 OK in", round(time.time() - t0, 1), "s; n429", out[0].n429, "usage per img", out[0].usage)
    for i in items[100:]:
        t = time.time()
        r = G.embed_one(i)
        print(f"single at +{t - t0:.1f}s after batch start: n429={r.n429} wait_s={r.wait_s:.1f} latency={r.latency_s:.2f}")
    print("calls sent", G.calls)
