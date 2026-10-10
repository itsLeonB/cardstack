"""Shared plumbing for the ticket 01 spike: env, catalog, image cache, provider interface, stats."""

from __future__ import annotations

import csv
import io
import os
import time
from dataclasses import dataclass, field
from pathlib import Path

import numpy as np
from PIL import Image

HERE = Path(__file__).parent
CACHE = HERE / "cache"
RESULTS = HERE / "results"
PHOTOS = Path(os.environ.get("SPIKE_PHOTOS_DIR", HERE / "photos"))
LABELS = Path(os.environ.get("SPIKE_LABELS_CSV", HERE / "labels.csv"))
SET_CODE = "MA6"


def load_env() -> None:
    """Load .env next to this file into os.environ (never overrides existing values, never printed)."""
    for line in (HERE / ".env").read_text().splitlines():
        if "=" in line and not line.lstrip().startswith("#"):
            k, v = line.split("=", 1)
            os.environ.setdefault(k.strip(), v.strip().strip("'\""))


@dataclass
class Card:
    id: str
    local_id: str
    name: str
    illustrator: str
    rarity_id: str
    image_key: str


def catalog() -> list[Card]:
    with open(HERE / "ma6-cards.csv", newline="", encoding="utf-8") as f:
        return [
            Card(r["id"], r["local_id"], r["name"], r["illustrator"], r["rarity_id"], r["image_key"])
            for r in csv.DictReader(f)
            if r["image_key"]
        ]


def catalog_image(card: Card) -> bytes:
    """Original hosted bytes (PNG), cached on disk. Same bytes go to every model."""
    import httpx

    path = CACHE / "catalog" / f"{card.id}.png"
    if not path.exists():
        path.parent.mkdir(parents=True, exist_ok=True)
        url = os.environ["IMAGE_BASE_URL"].rstrip("/") + "/" + card.image_key
        r = httpx.get(url, timeout=30, follow_redirects=True)
        r.raise_for_status()
        path.write_bytes(r.content)
    return path.read_bytes()


def frontend_jpeg(data: bytes, long_side: int = 1024, quality: int = 85) -> bytes:
    """Resize like the spec's frontend: ~1024px on the long side, JPEG. Used for every query image."""
    img = Image.open(io.BytesIO(data))
    img = img.convert("RGB")
    img.thumbnail((long_side, long_side), Image.LANCZOS)
    out = io.BytesIO()
    img.save(out, "JPEG", quality=quality)
    return out.getvalue()


def labels(cards: list[Card]) -> list[tuple[str, str]]:
    """labels.csv columns: photo_filename,set_code,local_number -> [(photo_filename, card_id)]."""
    by_local = {c.local_id: c.id for c in cards}
    out = []
    with open(LABELS, newline="", encoding="utf-8") as f:
        for r in csv.DictReader(f):
            assert r["set_code"].upper() == SET_CODE, r
            out.append((r["photo_filename"], by_local[r["local_number"].strip().zfill(3)]))
    return out


def normalise(v: np.ndarray) -> np.ndarray:
    v = np.asarray(v, dtype=np.float32)
    return v / np.linalg.norm(v)


@dataclass
class Item:
    key: str
    data: bytes
    mime: str  # image/png or image/jpeg


@dataclass
class Embedded:
    key: str
    vec: np.ndarray  # float32, L2-normalised
    latency_s: float  # time inside the successful request(s); excludes rate-limit waiting
    wait_s: float = 0.0  # time spent sleeping because of 429/backoff
    n429: int = 0
    usage: dict = field(default_factory=dict)  # provider usage fields, summed by the runner


class Provider:
    """One embedding model. Subclass in providers/<name>.py and expose make(variant) -> Provider."""

    name = "base"

    def embed_one(self, item: Item) -> Embedded:
        raise NotImplementedError

    def embed_many(self, items: list[Item]) -> list[Embedded]:
        """Catalog throughput path. Override to batch or to respect rate limits; default is a loop."""
        return [self.embed_one(i) for i in items]

    def info(self) -> dict:
        """Extra facts for the report (dimension, library versions, settings used)."""
        return {}

    def close(self) -> None:
        pass


def now() -> float:
    return time.perf_counter()


def pct(xs: list[float], q: float) -> float:
    return float(np.percentile(xs, q)) if xs else float("nan")
