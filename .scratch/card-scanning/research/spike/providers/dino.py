"""DINOv2 (HF transformers, CPU) provider.  make("base"|"small").

Embedding = pooler_output (layer-normed CLS token), L2-normalised.
Env switches:
  DINO_PREPROC  whole (default: resize whole card to DINO_SIZE, no crop) | hf (HF default: resize short edge 256, center-crop 224)
  DINO_SIZE     HxW multiple of 14 for "whole" (default 308x224)
  DINO_BATCH    embed_many batch size (default 8); embed_one is always batch 1
  DINO_THREADS  torch intra-op threads (default: torch default = all cores)
"""

from __future__ import annotations

import io
import os

import numpy as np
import torch
import transformers
from PIL import Image
from transformers import AutoImageProcessor, AutoModel

import common as c

MODELS = {"base": "facebook/dinov2-base", "small": "facebook/dinov2-small"}


class Dino(c.Provider):
    def __init__(self, variant: str):
        self.name = f"dino-{variant}"
        self.model_id = MODELS[variant]
        self.mode = os.environ.get("DINO_PREPROC", "whole")
        h, w = (int(x) for x in os.environ.get("DINO_SIZE", "308x224").split("x"))
        assert self.mode in ("whole", "hf") and h % 14 == 0 and w % 14 == 0
        self.size = (h, w)
        self.batch = int(os.environ.get("DINO_BATCH", "8"))
        if os.environ.get("DINO_THREADS"):
            torch.set_num_threads(int(os.environ["DINO_THREADS"]))
        # weights come from the local HF cache after the first run; load time measured by the runner
        self.proc = AutoImageProcessor.from_pretrained(self.model_id)
        if self.mode == "whole":
            self.proc.do_center_crop = False
            self.proc.size = {"height": h, "width": w}
        self.model = AutoModel.from_pretrained(self.model_id).eval()

    def _embed(self, items: list[c.Item]) -> list[c.Embedded]:
        t0 = c.now()
        imgs = [Image.open(io.BytesIO(i.data)).convert("RGB") for i in items]  # RGBA PNG -> RGB
        x = self.proc(images=imgs, return_tensors="pt")
        with torch.inference_mode():
            v = torch.nn.functional.normalize(self.model(**x).pooler_output, dim=-1)
        dt = (c.now() - t0) / len(items)  # per image; whole batch time split evenly
        return [c.Embedded(i.key, v[j].numpy().astype(np.float32), dt) for j, i in enumerate(items)]

    def embed_one(self, item: c.Item) -> c.Embedded:
        return self._embed([item])[0]

    def embed_many(self, items: list[c.Item]) -> list[c.Embedded]:
        return [e for k in range(0, len(items), self.batch) for e in self._embed(items[k : k + self.batch])]

    def info(self) -> dict:
        return {
            "model_id": self.model_id,
            "dim": self.model.config.hidden_size,
            "transformers": transformers.__version__,
            "torch": torch.__version__,
            "torch_threads": torch.get_num_threads(),
            "preprocessing": self.mode,
            "input_hxw": list(self.size) if self.mode == "whole" else "256 short edge + 224x224 center crop",
            "embed_many_batch": self.batch,
            "embed_one_batch": 1,
            "latency_includes": "PNG/JPEG decode + RGB convert + HF processor + forward + normalise",
        }


def make(variant: str) -> Dino:
    return Dino(variant)
