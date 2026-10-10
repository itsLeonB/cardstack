"""Run one stage for one model.  uv run spike.py <model> <stage> [--phase free|paid] [--repeats N]

models: voyage, gemini, dino-base, dino-small   (providers/<voyage|gemini|dino>.py, make(variant))
stages: catalog  embed all MA6 catalog images (original PNG bytes) -> catalog.npz + throughput stats
        latency  single-image request latency, p50/p95 (queries: photos if present, else catalog images as proxy)
        photos   embed the labeled phone photos (resized like the frontend) -> queries.npz
Outputs: results/<model>/<phase>/{catalog,queries}.npz and <stage>.json. Evaluate with evaluate.py.
"""

from __future__ import annotations

import argparse
import importlib
import json
import platform
import resource
import time

import numpy as np

import common as c

MODULES = {"voyage": ("providers.voyage", None), "gemini": ("providers.gemini", None),
           "dino-base": ("providers.dino", "base"), "dino-small": ("providers.dino", "small")}


def summarise(rs: list[c.Embedded]) -> dict:
    usage: dict = {}
    for r in rs:
        for k, v in r.usage.items():
            usage[k] = usage.get(k, 0) + v
    lat = [r.latency_s for r in rs]
    return {"n": len(rs), "latency_p50_s": c.pct(lat, 50), "latency_p95_s": c.pct(lat, 95),
            "latency_mean_s": float(np.mean(lat)), "wait_s": sum(r.wait_s for r in rs),
            "n429": sum(r.n429 for r in rs), "usage": usage}


def main() -> None:
    ap = argparse.ArgumentParser()
    ap.add_argument("model", choices=MODULES)
    ap.add_argument("stage", choices=["catalog", "latency", "photos"])
    ap.add_argument("--phase", default="free", choices=["free", "paid"])
    ap.add_argument("--repeats", type=int, default=25)
    ap.add_argument("--limit", type=int, default=0, help="only first N catalog cards (smoke test)")
    a = ap.parse_args()
    c.load_env()

    mod, variant = MODULES[a.model]
    t0 = time.perf_counter()
    cpu0 = time.process_time()
    prov = importlib.import_module(mod).make(variant)
    load_s = time.perf_counter() - t0
    out = c.RESULTS / a.model / a.phase
    out.mkdir(parents=True, exist_ok=True)
    cards = c.catalog()
    if a.limit:
        cards = cards[: a.limit]
    stats: dict = {"model": a.model, "stage": a.stage, "phase": a.phase, "load_s": load_s,
                   "cpu": platform.processor(), "py": platform.python_version()}

    cpu_start = time.process_time()
    wall = time.perf_counter()
    if a.stage == "catalog":
        items = [c.Item(k.id, c.catalog_image(k), "image/png") for k in cards]  # download is outside timing below
        wall = time.perf_counter()
        cpu_start = time.process_time()
        rs = prov.embed_many(items)
        wall_s = time.perf_counter() - wall
        np.savez(out / "catalog.npz", ids=np.array([r.key for r in rs]), vecs=np.stack([r.vec for r in rs]))
        stats |= summarise(rs) | {"wall_s": wall_s, "images_per_s": len(rs) / wall_s}
    elif a.stage == "photos":
        lab = c.labels(cards)
        items = [c.Item(f, c.frontend_jpeg((c.PHOTOS / f).read_bytes()), "image/jpeg") for f, _ in lab]
        wall = time.perf_counter()
        cpu_start = time.process_time()
        rs = prov.embed_many(items)
        wall_s = time.perf_counter() - wall
        np.savez(out / "queries.npz", files=np.array([f for f, _ in lab]), truth=np.array([i for _, i in lab]),
                 vecs=np.stack([r.vec for r in rs]))
        stats |= summarise(rs) | {"wall_s": wall_s}
    else:
        if c.LABELS.exists():
            raw = [(c.PHOTOS / f).read_bytes() for f, _ in c.labels(cards)]
            stats["queries"] = "photos"
        else:
            raw = [c.catalog_image(k) for k in cards[: a.repeats]]
            stats["queries"] = "catalog images resized like the frontend (PROXY: no phone photos yet)"
        items = [c.Item(f"q{i}", c.frontend_jpeg(raw[i % len(raw)]), "image/jpeg") for i in range(a.repeats)]
        prov.embed_one(items[0])  # warm-up (connection, lazy init); not counted
        cpu_start = time.process_time()
        wall = time.perf_counter()
        rs = [prov.embed_one(i) for i in items]
        wall_s = time.perf_counter() - wall
        stats |= summarise(rs) | {"wall_s": wall_s, "repeats": a.repeats}

    stats["provider_info"] = prov.info()  # after the stage so run-time facts (first 429, ...) are included
    stats["cpu_s_total"] = time.process_time() - cpu_start
    stats["cpu_s_per_image"] = stats["cpu_s_total"] / stats["n"]
    stats["peak_rss_mb"] = resource.getrusage(resource.RUSAGE_SELF).ru_maxrss / 1024
    stats["load_cpu_s"] = cpu_start - cpu0 if a.stage == "catalog" else None
    (out / f"{a.stage}.json").write_text(json.dumps(stats, indent=2, default=str))
    print(json.dumps(stats, indent=2, default=str))
    prov.close()


if __name__ == "__main__":
    main()
