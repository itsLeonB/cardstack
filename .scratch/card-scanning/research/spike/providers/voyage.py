"""Voyage voyage-multimodal-3.5 over raw httpx (SDK not needed; output_dimension verified on raw REST).

Env knobs (all optional):
  VOYAGE_INPUT_TYPE   none (default) | doc_query (PNG catalog -> document, JPEG photos -> query) | doc_doc
  VOYAGE_BATCH_TOKENS max tokens per request (default 9400 = 5 originals; 11300 = 6 originals, observed OK once/min)
  VOYAGE_WINDOW_TOKENS tokens allowed per sliding 60s window (default 11500)
  VOYAGE_RPM          requests per sliding 60s window (default 3)
Docs: https://docs.voyageai.com/reference/multimodal-embeddings-api (tokens = pixels/560)
      https://www.mongodb.com/docs/voyageai/api-reference/overview/ (free trial 3 RPM / 10K TPM)
"""

from __future__ import annotations

import base64
import io
import os
import random
import sys
import time
from collections import deque

import httpx
import numpy as np
from PIL import Image

import common as c

MODEL = "voyage-multimodal-3.5"
DIM = 1024
WINDOW_S = 61.0  # 60s plus a margin; limits are per minute and the window shape is not documented


def est_tokens(data: bytes) -> int:
    w, h = Image.open(io.BytesIO(data)).size
    return max(w * h, 50_000) // 560  # docs: images under 50,000 px bill as 50,000 px; 560 px per token


class Voyage(c.Provider):
    name = "voyage"

    def __init__(self) -> None:
        key = os.environ["VOYAGE_API_KEY"]
        self.host = "https://ai.mongodb.com/v1" if key.startswith("al-") else "https://api.voyageai.com/v1"
        self.http = httpx.Client(timeout=300, headers={"Authorization": "Bearer " + key})
        self.mode = os.environ.get("VOYAGE_INPUT_TYPE", "none")
        assert self.mode in ("none", "doc_query", "doc_doc"), self.mode
        self.batch_tokens = int(os.environ.get("VOYAGE_BATCH_TOKENS", "9400"))
        self.window_tokens = int(os.environ.get("VOYAGE_WINDOW_TOKENS", "11500"))
        self.rpm = int(os.environ.get("VOYAGE_RPM", "3"))
        self.sent: deque[tuple[float, int]] = deque()  # (monotonic time, tokens) of accepted requests
        self.first_429: str | None = None
        self.max_request_tokens = 0

    def input_type(self, mime: str) -> str | None:
        if self.mode == "doc_doc":
            return "document"
        if self.mode == "doc_query":
            return "document" if mime == "image/png" else "query"
        return None

    def _pace(self, tokens: int) -> float:
        """Sleep until the sliding window has room for this request. Returns seconds slept."""
        slept = 0.0
        while True:
            now = time.monotonic()
            while self.sent and now - self.sent[0][0] > WINDOW_S:
                self.sent.popleft()
            used = sum(t for _, t in self.sent)
            if len(self.sent) < self.rpm and (not self.sent or used + tokens <= self.window_tokens):
                return slept
            d = max(0.5, WINDOW_S - (now - self.sent[0][0]) + 0.3)
            time.sleep(d)
            slept += d

    def _request(self, items: list[c.Item]) -> list[c.Embedded]:
        toks = sum(est_tokens(i.data) for i in items)
        body = {
            "inputs": [{"content": [{"type": "image_base64",
                                     "image_base64": f"data:{i.mime};base64,{base64.b64encode(i.data).decode()}"}]}
                       for i in items],
            "model": MODEL,
            "output_dimension": DIM,
        }
        it = self.input_type(items[0].mime)
        if it:
            body["input_type"] = it
        wait = 0.0
        n429 = 0
        backoff = 15.0
        while True:
            wait += self._pace(toks)
            t = c.now()
            r = self.http.post(self.host + "/multimodalembeddings", json=body)
            dt = c.now() - t
            if r.status_code == 429:
                n429 += 1
                if self.first_429 is None:
                    self.first_429 = f"HTTP 429 after {dt:.2f}s, est_tokens={toks}, n_inputs={len(items)}, " \
                                     f"headers_retry_after={r.headers.get('retry-after')}, body={r.text[:1500]}"
                    print("VOYAGE FIRST 429:", self.first_429, file=sys.stderr, flush=True)
                    (c.RESULTS / "voyage").mkdir(parents=True, exist_ok=True)
                    with open(c.RESULTS / "voyage" / "first429.log", "a") as f:
                        f.write(time.strftime("%F %T ") + self.first_429 + "\n")
                d = min(backoff, 70.0) * random.uniform(0.8, 1.2)
                time.sleep(d)
                wait += d
                backoff *= 1.6
                if n429 > 6:
                    raise RuntimeError("voyage: gave up after 7 consecutive 429s")
                continue
            r.raise_for_status()
            break
        self.sent.append((time.monotonic(), toks))
        self.max_request_tokens = max(self.max_request_tokens, toks)
        j = r.json()
        data = sorted(j["data"], key=lambda d: d["index"])
        assert len(data) == len(items)
        out = []
        for n, (i, d) in enumerate(zip(items, data)):
            v = np.asarray(d["embedding"], dtype=np.float32)
            assert v.shape == (DIM,), v.shape
            first = n == 0  # whole-request figures sit on the first item so the runner's sums are exact
            out.append(c.Embedded(i.key, c.normalise(v), dt / len(items), wait if first else 0.0,
                                  n429 if first else 0, dict(j["usage"]) if first else {}))
        return out

    def embed_one(self, item: c.Item) -> c.Embedded:
        return self._request([item])[0]

    def embed_many(self, items: list[c.Item]) -> list[c.Embedded]:
        out: list[c.Embedded] = []
        batch: list[c.Item] = []
        used = 0
        for i in items:
            t = est_tokens(i.data)
            if batch and (used + t > self.batch_tokens or i.mime != batch[0].mime):
                out += self._request(batch)
                print(f"voyage {len(out)}/{len(items)}", file=sys.stderr, flush=True)
                batch, used = [], 0
            batch.append(i)
            used += t
        if batch:
            out += self._request(batch)
        return out

    def info(self) -> dict:
        return {"model": MODEL, "host": self.host, "dim": DIM, "input_type_mode": self.mode,
                "batch_tokens": self.batch_tokens, "window_tokens": self.window_tokens, "rpm": self.rpm,
                "max_request_tokens_est": self.max_request_tokens, "first_429": self.first_429,
                "transport": "raw httpx, image_base64 data URLs, one image per input"}

    def close(self) -> None:
        import json  # spike.py snapshots info() before the stage runs, so persist the final one here
        (c.RESULTS / "voyage").mkdir(parents=True, exist_ok=True)
        (c.RESULTS / "voyage" / "last_run_info.json").write_text(json.dumps(self.info() | {"at": time.strftime("%F %T")}, indent=2))
        self.http.close()


def make(variant: str | None = None) -> c.Provider:
    return Voyage()
