"""Score embeddings.  uv run evaluate.py <model> [--phase free]

Always prints catalog-vs-catalog nearest-neighbour margins (reprint/near-duplicate risk, no photos needed).
Prints top-1/top-5, precision@coverage and a provisional confident rule when queries.npz exists.
"""

from __future__ import annotations

import argparse
import json

import numpy as np

import common as c


def main() -> None:
    ap = argparse.ArgumentParser()
    ap.add_argument("model")
    ap.add_argument("--phase", default="free")
    a = ap.parse_args()
    d = c.RESULTS / a.model / a.phase
    cat = np.load(d / "catalog.npz")
    ids, V = list(cat["ids"]), cat["vecs"]
    res: dict = {"model": a.model, "phase": a.phase, "catalog_n": len(ids)}

    S = V @ V.T
    np.fill_diagonal(S, -1)
    nn = S.max(1)
    res["catalog_nn_cosine"] = {"p50": float(np.percentile(nn, 50)), "p95": float(np.percentile(nn, 95)), "max": float(nn.max())}

    if (d / "queries.npz").exists():
        q = np.load(d / "queries.npz")
        sims = q["vecs"] @ V.T
        order = np.argsort(-sims, 1)
        truth = [ids.index(t) for t in q["truth"]]
        rank = np.array([list(order[i]).index(t) for i, t in enumerate(truth)])
        top1 = rank == 0
        res |= {"n_photos": len(rank), "top1": int(top1.sum()), "top5": int((rank < 5).sum()),
                "top1_pct": 100 * top1.mean(), "top5_pct": 100 * (rank < 5).mean(),
                "floor_ok": bool(top1.mean() >= 0.90 and (rank < 5).mean() >= 0.98)}
        s1 = sims[np.arange(len(rank)), order[:, 0]]
        s2 = sims[np.arange(len(rank)), order[:, 1]]
        score, margin = s1, s1 - s2
        best = None  # smallest-coverage-loss rule with precision >= 0.98; non-final
        for t in np.unique(np.round(score, 3)):
            for m in np.unique(np.round(margin, 3)):
                conf = (score >= t) & (margin >= m)
                if conf.sum() and top1[conf].mean() >= 0.98 and (best is None or conf.sum() > best[2]):
                    best = (float(t), float(m), int(conf.sum()), float(top1[conf].mean()))
        res["confident_rule_provisional"] = None if best is None else {
            "min_score": best[0], "min_margin": best[1], "coverage": best[2] / len(rank), "precision": best[3]}
        res["score_correct_mean"] = float(s1[top1].mean()) if top1.any() else None
        res["score_wrong_mean"] = float(s1[~top1].mean()) if (~top1).any() else None
    print(json.dumps(res, indent=2))
    (d / "eval.json").write_text(json.dumps(res, indent=2))


if __name__ == "__main__":
    main()
