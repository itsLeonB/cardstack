"""Gemini gemini-embedding-2 over raw REST (httpx).  make(variant) -> Provider.

Facts: https://ai.google.dev/gemini-api/docs/embeddings (one image per Content, no task_type, no text part,
output_dimensionality; 3072 default, 768/1536/3072 recommended, truncated dims auto-normalised),
https://ai.google.dev/api/embeddings (embedContent / batchEmbedContents, usageMetadata.promptTokenCount),
https://ai.google.dev/gemini-api/docs/troubleshooting (backoff with jitter on 429 / 5xx).
"""

from __future__ import annotations

import base64
import json
import os
import random
import re
import time
from collections import deque
from datetime import datetime, timezone

import httpx

import common as c

MODEL = "gemini-embedding-2"
BASE = "https://generativelanguage.googleapis.com/v1beta"
DIM = int(os.environ.get("GEMINI_DIM", "1536"))
RPM_CAP = int(os.environ.get("GEMINI_RPM", "90"))  # stay under the developer-read free-tier 100 RPM
MAX_TRIES = 8
USE_BATCH = os.environ.get("GEMINI_USE_BATCH", "1") == "1"  # embed_many via batchEmbedContents; see NOTES.md
LOG_429 = c.RESULTS / "gemini" / "429.log"


class DailyQuota(RuntimeError):
    pass


def _part(item: c.Item) -> dict:
    return {"inline_data": {"mime_type": item.mime, "data": base64.b64encode(item.data).decode()}}


class Gemini(c.Provider):
    name = "gemini"

    def __init__(self, dim: int = DIM) -> None:
        self.dim = dim
        self.http = httpx.Client(timeout=120, headers={"x-goog-api-key": os.environ["GEMINI_API_KEY"]})
        self.sent: deque[float] = deque()  # wall times of request attempts in the last 60 s (pacing)
        self.calls = 0

    # -- transport -------------------------------------------------------------------------------
    def _pace(self) -> float:
        """Block until a new request fits under RPM_CAP in a rolling 60 s window. Returns seconds slept."""
        slept = 0.0
        while True:
            now = time.monotonic()
            while self.sent and now - self.sent[0] >= 60:
                self.sent.popleft()
            if len(self.sent) < RPM_CAP:
                self.sent.append(now)
                return slept
            d = 60 - (now - self.sent[0]) + 0.05
            time.sleep(d)
            slept += d

    def _log_429(self, body: str, kind: str, n_images: int) -> None:
        LOG_429.parent.mkdir(parents=True, exist_ok=True)
        with LOG_429.open("a") as f:
            f.write(f"--- {datetime.now(timezone.utc).isoformat()} kind={kind} images_in_call={n_images}\n{body}\n")

    def _post(self, method: str, body: dict, n_images: int = 1) -> tuple[dict, float, float, int]:
        """-> (json, latency_s of the successful request, wait_s, n429)."""
        url = f"{BASE}/models/{MODEL}:{method}"
        wait, n429 = 0.0, 0
        for attempt in range(MAX_TRIES):
            wait += self._pace()
            t = c.now()
            try:
                r = self.http.post(url, json=body)
            except (httpx.TransportError, httpx.TimeoutException):
                r = None
            dt = c.now() - t
            self.calls += 1
            if r is not None and r.status_code == 200:
                return r.json(), dt, wait, n429
            back = min(60.0, 2.0 ** attempt) * random.uniform(0.5, 1.0)
            if r is not None and r.status_code == 429:
                n429 += 1
                daily = "PerDay" in r.text or "per day" in r.text.lower()
                self._log_429(r.text, "per-day" if daily else "per-minute-or-other", n_images)
                if daily:
                    raise DailyQuota(r.text)
                m = re.search(r'"retryDelay":\s*"([\d.]+)s"', r.text)
                back = max(back, float(m.group(1)) + 1 if m else 0)
            elif r is not None and r.status_code < 500:
                raise RuntimeError(f"{method} HTTP {r.status_code}: {r.text[:500]}")
            time.sleep(back)
            wait += back
        raise RuntimeError(f"{method}: gave up after {MAX_TRIES} tries")

    # -- provider interface ----------------------------------------------------------------------
    def _body(self, item: c.Item) -> dict:
        # one image per Content; no text part; no task_type (unsupported for this model)
        return {"content": {"parts": [_part(item)]}, "output_dimensionality": self.dim}

    def embed_one(self, item: c.Item) -> c.Embedded:
        j, dt, wait, n429 = self._post("embedContent", self._body(item))
        um = j.get("usageMetadata") or {}
        usage = {"promptTokenCount": um["promptTokenCount"]} if "promptTokenCount" in um else {}
        return c.Embedded(item.key, c.normalise(j["embedding"]["values"]), dt, wait, n429, usage)

    def embed_batch_sync(self, items: list[c.Item]) -> list[c.Embedded]:
        """One batchEmbedContents call, one request per image. latency/usage split evenly over the images."""
        body = {"requests": [{"model": f"models/{MODEL}", **self._body(i)} for i in items]}
        j, dt, wait, n429 = self._post("batchEmbedContents", body, len(items))
        embs = j["embeddings"]
        assert len(embs) == len(items), (len(embs), len(items))
        um = j.get("usageMetadata") or {}
        usage = {"promptTokenCount": um["promptTokenCount"] / len(items)} if "promptTokenCount" in um else {}
        n = len(items)
        return [c.Embedded(i.key, c.normalise(e["values"]), dt / n, wait / n, n429 if k == 0 else 0, usage)
                for k, (i, e) in enumerate(zip(items, embs))]

    def embed_many(self, items: list[c.Item]) -> list[c.Embedded]:
        if not USE_BATCH:
            return [self.embed_one(i) for i in items]
        out = []
        for s in range(0, len(items), 5):  # ponytail: fixed chunk of 5, tune only if batching is adopted
            out += self.embed_batch_sync(items[s : s + 5])
        return out

    def info(self) -> dict:
        return {"model": MODEL, "output_dimensionality": self.dim, "rpm_cap_used": RPM_CAP, "use_batch": USE_BATCH,
                "transport": "httpx REST x-goog-api-key, 1 image per Content, no text, no task_type"}

    def close(self) -> None:
        self.http.close()


def make(variant: str | None = None) -> c.Provider:
    return Gemini()
