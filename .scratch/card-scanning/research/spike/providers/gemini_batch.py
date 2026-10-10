"""UNTESTED, PAID-TIER ONLY.  Gemini asynchronous Batch API on the 199 MA6 catalog images (ticket 01, paid pass).

    uv run python providers/gemini_batch.py --dry-run     # offline: build requests, print sizes, no API call
    uv run python providers/gemini_batch.py               # submit, poll, write results/gemini/paid-batch/catalog.npz

Surface used (all from primary sources, nothing here has been executed against the API):
- client.batches.create_embeddings(model=, src={'inlined_requests': EmbedContentBatch(contents=[Content...], config=...)},
  config={'display_name': ...})  https://ai.google.dev/gemini-api/docs/batch-api ("Batch embedding support") and
  google-genai 2.29.0 batches.py (each Content in `contents` becomes one request: requests[].request.content).
- states JOB_STATE_PENDING/RUNNING/SUCCEEDED/FAILED/CANCELLED/EXPIRED; results in job.dest.inlined_embed_content_responses,
  same order as the input; target turnaround 24 h, expiry after 48 h (same page).
- inline requests: keep total request size under 20 MB (same page). The 199 PNG originals are ~320 MB, so they are split into
  several inline jobs of <= 18 MB. The JSONL file format for embedding batches is NOT documented, so file input is not used.
- Batch tier is paid-only: $0.225 per 1M tokens, "Not available" on the free tier (https://ai.google.dev/gemini-api/docs/pricing).
- Enqueued-token cap for "Gemini Embedding": 500,000 at Tier 1 (https://ai.google.dev/gemini-api/docs/rate-limits);
  199 images x 258 tokens = 51,342 tokens, well under.
Turnaround is measured per job: submit time -> first poll that sees SUCCEEDED, reported as the slowest job (= whole catalog).
"""

from __future__ import annotations

import json
import sys
import time
from pathlib import Path

import numpy as np

sys.path.insert(0, str(Path(__file__).resolve().parent.parent))
import common as c  # noqa: E402

MODEL = "gemini-embedding-2"
DIM = 1536
MAX_JOB_BYTES = 18 * 1024 * 1024  # under the documented 20 MB inline cap, leaving room for base64 + JSON overhead
BASE64_OVERHEAD = 4 / 3
POLL_S = 60
OUT = c.RESULTS / "gemini" / "paid-batch"


def chunks(items: list[c.Item]) -> list[list[c.Item]]:
    out, cur, size = [], [], 0
    for it in items:
        n = int(len(it.data) * BASE64_OVERHEAD)
        if cur and size + n > MAX_JOB_BYTES:
            out.append(cur)
            cur, size = [], 0
        cur.append(it)
        size += n
    return out + [cur] if cur else out


def main() -> None:
    dry = "--dry-run" in sys.argv
    c.load_env()
    items = [c.Item(k.id, c.catalog_image(k), "image/png") for k in c.catalog()]
    groups = chunks(items)
    print(f"{len(items)} images -> {len(groups)} inline jobs, bytes/job (raw)", [sum(len(i.data) for i in g) for g in groups])

    from google import genai
    from google.genai import types

    def batch_for(g: list[c.Item]) -> types.EmbedContentBatch:
        # one Content per image => one request and one vector per image; no text part, no task_type
        return types.EmbedContentBatch(
            contents=[types.Content(parts=[types.Part.from_bytes(data=i.data, mime_type=i.mime)]) for i in g],
            config=types.EmbedContentConfig(output_dimensionality=DIM),
        )

    if dry:
        for g in groups:
            batch_for(g)  # construct only, to check the types accept it
        print("dry run OK (no API call)")
        return

    client = genai.Client()  # reads GEMINI_API_KEY from the environment; a BILLED project is required
    jobs = []
    for n, g in enumerate(groups):
        job = client.batches.create_embeddings(
            model=MODEL, src={"inlined_requests": batch_for(g)}, config={"display_name": f"ma6-catalog-{n}"}
        )
        jobs.append({"name": job.name, "t_submit": time.time(), "keys": [i.key for i in g], "done": None})
        print("submitted", job.name, job.state)

    pending = set(range(len(jobs)))
    while pending:
        time.sleep(POLL_S)
        for n in sorted(pending):
            job = client.batches.get(name=jobs[n]["name"])
            state = job.state.name if job.state else "?"
            print(time.strftime("%H:%M:%S"), jobs[n]["name"], state)
            if state in ("JOB_STATE_SUCCEEDED", "JOB_STATE_FAILED", "JOB_STATE_CANCELLED", "JOB_STATE_EXPIRED"):
                jobs[n]["done"] = time.time()
                jobs[n]["state"] = state
                if state == "JOB_STATE_SUCCEEDED":
                    rs = job.dest.inlined_embed_content_responses
                    assert len(rs) == len(jobs[n]["keys"]), (len(rs), len(jobs[n]["keys"]))
                    jobs[n]["vecs"] = [r.response.embedding.values for r in rs]
                    jobs[n]["tokens"] = sum(r.response.token_count or 0 for r in rs)
                pending.discard(n)

    OUT.mkdir(parents=True, exist_ok=True)
    ok = [j for j in jobs if j.get("vecs")]
    ids = [k for j in ok for k in j["keys"]]
    vecs = np.stack([c.normalise(v) for j in ok for v in j["vecs"]])
    np.savez(OUT / "catalog.npz", ids=np.array(ids), vecs=vecs)
    stats = {"jobs": len(jobs), "states": [j["state"] for j in jobs], "n_embedded": len(ids),
             "turnaround_s_per_job": [j["done"] - j["t_submit"] for j in jobs],
             "turnaround_s_slowest": max(j["done"] - j["t_submit"] for j in jobs),
             "tokens": sum(j.get("tokens", 0) for j in jobs)}
    (OUT / "batch.json").write_text(json.dumps(stats, indent=2))
    print(json.dumps(stats, indent=2))


if __name__ == "__main__":
    main()
