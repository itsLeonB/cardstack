# Primary-source facts for the card-scanning accuracy spike

Research note, 2026-10-10 (all pages accessed 2026-10-10). Purpose: give the author of the ticket 01 spike script and report the verified facts they need, so no further doc-hunting is required. Companion to `.scratch/card-scanning/research/hosted-image-embedding-apis.md` (2026-10-06), whose two UNVERIFIED rate-limit items are resolved or narrowed here. Only provider-owned pages, source repositories, model cards, papers and this repo are cited. Pages were fetched as raw HTML and converted to text locally (not through a summarising fetcher), except the arXiv abstract and PDF, which were read from the PDF text. Anything not confirmed on a primary page is marked UNVERIFIED. No product code was written and the spike was not run.

## Bottom line

- **Voyage free-trial rate limit is now confirmed: 3 RPM and 10K TPM for an account with no payment method.** Stated on the Atlas Embedding and Reranking API overview: "Free trial rate limits without a payment method are 3 RPM and 10K TPM. To qualify for higher rate limits, add a payment method to your account." (https://www.mongodb.com/docs/voyageai/api-reference/overview/). The page that the prior note read (`.../management/rate-limits/`) only lists Usage Tier 1 to 3, which is why the note called it UNVERIFIED. Adding a payment method moves the org to Usage Tier 1: `voyage-multimodal-3.5` at 2,000,000 TPM and 2,000 RPM (https://www.mongodb.com/docs/voyageai/management/rate-limits/). The free allowance (200M text tokens, 150B pixels) still applies after adding a card (https://docs.voyageai.com/docs/rate-limits).
- **Gemini free-tier limits for `gemini-embedding-2` (read by the developer from the AI Studio rate-limit screen on 2026-10-10, not on any public page): 100 RPM, 30K TPM, 1,000 RPD.** The screen listed the model under "Other models". The public rate-limits page only says limits "can be viewed in Google AI Studio" and "Specified rate limits are not guaranteed and actual capacity may vary" (https://ai.google.dev/gemini-api/docs/rate-limits). Image embedding is "Free of charge" on the Free tier, with the trade-off that free-tier content is "Used to improve our products" (https://ai.google.dev/gemini-api/docs/pricing). The daily cap is the binding limit: at one image per request, 1,000 RPD means the 12,000-card batch takes about 12 days on the free tier (100 RPM and 30K TPM, about 112 images per minute at roughly 267 tokens per image, would allow it in about 2 hours if the daily cap did not exist). The ~230-image spike (200 catalog images plus ~30 photos) fits inside one day. The ongoing 300 user photos per month fit comfortably. A paid account removes the free-tier cap (the pricing for the full catalog is about $1.44).
- **Paid batch comparison for the 12,000-card run (added 2026-10-10).** Gemini async Batch API: $0.00006 per image, about $0.72 for 12k, paid tier only, enqueued-token cap of 500,000 at Tier 1 (about 1,870 images at roughly 267 tokens each, so about 7 job waves), completion window not checked (https://ai.google.dev/gemini-api/docs/pricing, https://ai.google.dev/gemini-api/docs/rate-limits). Voyage has a Batch API with a "12-hour completion window and a 33% discount", but its model list is text, code, context and rerank models only and does not include `voyage-multimodal-3.5`, so there is no paid Voyage batch for images (https://docs.voyageai.com/docs/pricing, "Batch and Files API"; https://docs.voyageai.com/docs/batch-inference, "Model Availability"). Voyage standard paid is $0.60 per 1B pixels: about $0.90 for 12k small (299x418) images or about $7.58 for 12k large (868x1213) images, and it takes 1 to 11 minutes at Tier 1 (section 1.6). The 150B-pixel free allowance still covers the whole batch even after adding a card, so the real Voyage cost is $0.
- **Gemini quirk that breaks naive scripts: one request with several parts yields ONE aggregated embedding.** Send exactly one image per `Content` (one image per `embedContent` call, or one `requests[]` entry per image in `batchEmbedContents`) (https://ai.google.dev/gemini-api/docs/embeddings, "Embedding aggregation"). `task_type` is not supported on `gemini-embedding-2`, and there is no documented image-to-image task prefix, so embed catalog and query images identically with no text.
- **Recommended local control: DINOv2 ViT-B/14 (`facebook/dinov2-base`, 768-d, Apache-2.0).** Its owners list "image retrieval using nearest neighbors" as a direct use (https://github.com/facebookresearch/dinov2/blob/main/MODEL_CARD.md), and the paper evaluates it as instance-level retrieval by cosine similarity (Section 7.3, https://arxiv.org/abs/2304.07193). SigLIP's model cards list only zero-shot classification and image-text retrieval (https://huggingface.co/google/siglip-base-patch16-224).
- **Catalog source images are small and vary in size, which changes the Voyage free-trial maths.** Sampled source images were 299x418 (older sets) and 868x1213 (newer sets), see section 5.3. At 3 RPM and 10K TPM the free trial would need roughly 4.5 hours (small images) to 38 hours (large images) for the 12,000-card batch, versus about 1 to 11 minutes at Tier 1. The ~200-card spike fits the free trial if requests are batched, see section 1.6.
- **Scoring unit:** use cosine similarity everywhere (`1 - (embedding <=> query)` in pgvector). Voyage says its embeddings are normalised to length 1, Gemini normalises 3072-d and truncated dims, DINOv2 and SigLIP image vectors are not normalised and must be L2-normalised by the script (section 4).
- **Ground truth for the report:** `cards.id` (UUID). A Card is unique per `(expansion_set_id, local_id)`, so a reprint in another Expansion Set is a different Card with its own image. The repo has no "artwork identity" column, so the reprint tie rate must be defined empirically (section 5.2).
- **Remaining UNVERIFIED:** whether `batchEmbedContents` counts one request or N toward the 100 RPM and 1,000 RPD (if it counts one, the 12-day estimate shrinks sharply, so measure it); whether Voyage's free-trial 10K TPM rejects a single request whose tokens exceed 10K; whether the REST `output_dimension` parameter is accepted for `voyage-multimodal-3.5` (the REST reference page does not list it, the official Python client sends it); Gemini image token count per embedded image; CPU inference speed from the model owners; any provider guidance on glare, rotation or perspective (none found).

## 1. Voyage `voyage-multimodal-3.5`

### 1.1 Endpoint, auth and key type

- Two hosts exist. Atlas-issued keys use `https://ai.mongodb.com/v1/...`; older Voyage-dashboard keys use `https://api.voyageai.com/v1/...`. The migration page maps `https://api.voyageai.com/v1/multimodalembeddings` to `https://ai.mongodb.com/v1/multimodalembeddings` (https://www.mongodb.com/docs/voyageai/tutorials/migrate-to-atlas/). The official Python client chooses the host from the key prefix: keys starting `al-` go to `https://ai.mongodb.com/v1`, everything else to `https://api.voyageai.com/v1` (https://github.com/voyage-ai/voyageai-python/blob/main/voyageai/util.py, `get_default_base_url`, lines 99 to 103 of `main` at commit 9aca465).
- The API reference page documents `POST https://api.voyageai.com/v1/multimodalembeddings` (https://docs.voyageai.com/reference/multimodal-embeddings-api). Use the `ai.mongodb.com` host for a key created in the Atlas UI (https://www.mongodb.com/docs/voyageai/quickstart/, "Create a Model API Key"). Auth is a bearer token, see https://www.mongodb.com/docs/voyageai/api-reference/overview/.
- Python client: `voyageai` 0.5.0 (released 2026-07-10, https://pypi.org/project/voyageai/ and https://api.github.com/repos/voyage-ai/voyageai-python/releases/latest). Reads `VOYAGE_API_KEY` from the environment (https://docs.voyageai.com/docs/multimodal-embeddings).

### 1.2 Request shape (image URL and base64)

Body fields on the reference page (https://docs.voyageai.com/reference/multimodal-embeddings-api):

- `inputs` (required): list of `{"content": [ ... ]}`; each content item has `type` and a matching key: `text`, `image_url`, `image_base64`, `video_url`, `video_base64`.
- `model` (required): `voyage-multimodal-3.5` or `voyage-multimodal-3`.
- `input_type`: `null` (default), `query` or `document`.
- `truncation`: boolean, default `true`.
- `output_encoding`: `null` (default, floats) or `base64` (base64 NumPy float32 array).
- `image_url`: a URL to PNG, JPEG, WEBP or GIF. Since 2025-12-08 URL inputs must "Limit the number of redirects", "Require that responses include a content-length header" and "Respect robots.txt".
- `image_base64`: a data URL `data:[<mediatype>];base64,<data>` with mediatype `image/png`, `image/jpeg`, `image/webp` or `image/gif`.
- Consistency rule: "each request should use either image_base64/video_base64 or image_url/video_url exclusively, not both."

Equivalent minimal JSON for one image per input (SDK source confirms the same segment keys, https://github.com/voyage-ai/voyageai-python/blob/main/voyageai/object/multimodal_embeddings.py, `MultimodalInputSegmentImageURL`, `MultimodalInputSegmentImageBase64`):

```json
{"inputs": [{"content": [{"type": "image_base64", "image_base64": "data:image/png;base64,..."}]}], "model": "voyage-multimodal-3.5", "output_dimension": 1024}
```

Response (reference page): `object: "list"`, `data[]` with `object`, `embedding` (floats, or a base64 string when requested), `index`; `model`; `usage` with `text_tokens`, `image_pixels`, `video_pixels`, `total_tokens`. One input produces one embedding, in order; sending one image per input (one content item) is therefore the safe way to get one vector per image. The docs describe inputs as "interleaved" sequences of text and images, so several content items inside one input are fused into one vector; this is inferred from the architecture text at https://www.mongodb.com/docs/voyageai/models/multimodal-embeddings/ and is not an explicit statement, so keep one item per input.

Discrepancy to know about: the Python client defaults `encoding_format` to `"base64"` and decodes the reply itself (https://github.com/voyage-ai/voyageai-python/blob/main/voyageai/api_resources/embedding.py), while the REST reference page documents the parameter as `output_encoding`. For a raw-HTTP script, omit both and read float arrays; for the SDK, let it handle encoding.

### 1.3 Limits per request and per input

All from https://docs.voyageai.com/reference/multimodal-embeddings-api (identical text on https://docs.voyageai.com/docs/multimodal-embeddings):

- At most 1,000 inputs per request.
- Each image at most 16 million pixels and 20 MB.
- Token accounting: "every 560 pixels of an image ... counted as a token"; each input at most 32,000 tokens; at most 320,000 tokens across all inputs in a request.
- Billing treats images under 50,000 px as 50,000 px and downsamples images over 2,000,000 px to 2,000,000 px (https://www.mongodb.com/docs/voyageai/management/billing/, multimodal pricing section). Whether the rate-limit token count uses the downsampled size is UNVERIFIED.

### 1.4 Output dimensions, dtype, normalisation

- Model page: context 32,000 tokens; dimensions "1024 (default), 256, 512, 2048" (https://www.mongodb.com/docs/voyageai/models/multimodal-embeddings/ and https://docs.voyageai.com/docs/multimodal-embeddings). The launch post says the model supports 2048, 1024, 512 and 256 via Matryoshka learning plus float32, signed and unsigned int8, and binary quantisation (https://blog.voyageai.com/2026/01/15/voyage-multimodal-3-5/).
- Parameter names: the REST reference page does not list `output_dimension` or `output_dtype` for the multimodal endpoint (checked in the raw page text). The official Python client's `multimodal_embed` does accept `output_dtype` and `output_dimension` and forwards them in the request body (https://github.com/voyage-ai/voyageai-python/blob/main/voyageai/client.py lines 195 to 217; request model in `voyageai/object/multimodal_embeddings.py`). Dtype values recognised by the client: `float`, `int8`, `uint8`, `binary`, `ubinary` (`voyageai/util.py`, `_resolve_numpy_dtype`). So: prefer the SDK (`pip install voyageai`) in the spike, or verify with one raw call that `output_dimension: 1024` returns 1024 floats before relying on it. Whether the server accepts the field on raw REST is UNVERIFIED.
- Matryoshka truncation by hand: keep the leading k entries and re-normalise (https://docs.voyageai.com/docs/flexible-dimensions-and-quantization). For pgvector use float and 1024 or fewer (the `vector` index cap is 2,000, section 4).

### 1.5 `input_type` guidance

- "When input_type is None, the embedding model directly converts the inputs into numerical vectors." For retrieval, where a query "can be text or image", Voyage recommends setting `query` or `document`, and "automatically prepends a prompt": `Represent the query for retrieving supporting documents:` or `Represent the document for retrieval:`. "Embeddings generated with and without the input_type argument are compatible." (https://docs.voyageai.com/reference/multimodal-embeddings-api and https://docs.voyageai.com/docs/multimodal-embeddings).
- Nothing on either page addresses image-to-image retrieval. Both prompts are text and are prepended to an image-only input. The spike can compare three settings cheaply: `null` on both sides (the prior note's choice), `document` on catalog and `query` on photos, and `document` on both. Mark this as a free experiment, not a documented recommendation.

### 1.6 Free tier and rate limits

- Free allowance: "The first 200M text tokens and 150B pixels for voyage-multimodal-3.5 and voyage-multimodal-3 are free for every account" (https://docs.voyageai.com/docs/pricing); the Atlas billing page repeats it as a free tier of "200 million text tokens and 150 billion pixels for multimodal models" and lists a "Free trial" level that advances to Usage Tier 1 by adding a payment method (https://www.mongodb.com/docs/voyageai/management/billing/, "Usage Tiers"). Free tokens do not apply to the Batch API (https://docs.voyageai.com/docs/pricing).
- Free-trial rate limit (no payment method): 3 RPM and 10K TPM (https://www.mongodb.com/docs/voyageai/api-reference/overview/, "Rate Limits and Usage Tiers", "Free Trial"). Limits are "applied per API key" and measured in RPM and TPM there; the rate-limits page says they are set per organization with optional lower project limits. Exceeding either returns HTTP 429 (https://www.mongodb.com/docs/voyageai/management/rate-limits/; error table at https://docs.voyageai.com/docs/error-codes). Voyage recommends exponential backoff with jitter (https://docs.voyageai.com/docs/rate-limits, "Exponential Backoff").
- With a payment method (Usage Tier 1): `voyage-multimodal-3.5` and `voyage-multimodal-3` at 2,000,000 TPM and 2,000 RPM; Tier 2 (spend at least $100) is 2x and Tier 3 (at least $1,000) is 3x (https://www.mongodb.com/docs/voyageai/management/rate-limits/; qualification table at https://www.mongodb.com/docs/voyageai/management/billing/).
- Unknown: whether a single request over 10K tokens is rejected outright or merely consumes the next minutes' budget (UNVERIFIED). The script should start with batches of 20 images and halve on a 429 that does not clear after the `Retry-After` wait, logging what it observes. The docs do not mention a `Retry-After` header; the script should not assume one.
- Derived arithmetic (not a published figure; tokens = pixels / 560 as documented, rounded up):

| Image size | Pixels | Tokens per image | Free trial (10K TPM, 3 RPM) for 12,000 images | Tier 1 (2M TPM, 2,000 RPM) for 12,000 images |
| --- | --- | --- | --- | --- |
| 299x418 (older sets) | 124,982 | about 223 | about 45 images per minute, about 4.5 hours, needs batches of about 44 per request | about 2.7M tokens, about 1.4 minutes of TPM |
| 868x1213 (newer sets) | 1,052,884 | about 1,880 | about 5 images per minute, about 38 hours | about 22.6M tokens, about 11 minutes of TPM |
| Phone photo, 768x1024 (about 1024 on the long side, as the spec crops) | 786,432 | about 1,404 | about 7 photos per minute, the 30 photos take about 4 minutes | not binding |

- Pixel allowance used by the full batch: at most 12,000 x 1.05M = about 12.6B px, 8.4% of 150B.

## 2. Gemini `gemini-embedding-2`

### 2.1 Model id and endpoints

- Stable model id `gemini-embedding-2` (multimodal); `gemini-embedding-001` is text-only (https://ai.google.dev/gemini-api/docs/embeddings and https://ai.google.dev/gemini-api/docs/models/gemini-embedding-2, latest update April 2026).
- `POST https://generativelanguage.googleapis.com/v1beta/models/gemini-embedding-2:embedContent` and `...:batchEmbedContents`, header `x-goog-api-key` (https://ai.google.dev/gemini-api/docs/embeddings, REST examples). Resource names in a batch use `models/gemini-embedding-2` (same page). `batchEmbedContents` is the synchronous multi-request endpoint; the asynchronous Batch API is a separate product (https://ai.google.dev/api/embeddings, "Method: models.batchEmbedContents" and the `EmbedContentBatch` resource).

### 2.2 Request shape for inline image data

Documented REST body (https://ai.google.dev/gemini-api/docs/embeddings, "Embedding images"):

```json
{"content": {"parts": [{"inline_data": {"mime_type": "image/png", "data": "<base64>"}}]}}
```

Python from the same page: `client.models.embed_content(model='gemini-embedding-2', contents=[types.Part.from_bytes(data=image_bytes, mime_type='image/png')])`. Images may also be "provided as inline data or as uploaded files through the Files API"; no public-URL input is documented. Response: `embedding.values[]` for `embedContent`, `embeddings[]` in request order for `batchEmbedContents`, each with `usageMetadata` (https://ai.google.dev/api/embeddings, `EmbedContentResponse`, `BatchEmbedContentsResponse`, `ContentEmbedding`). Batch example with an image: `{"requests": [{"model": "models/gemini-embedding-2", "content": {"parts": [{"inline_data": {...}}]}}]}`.

### 2.3 Image limits

- "Maximum of 6 images per request. Supported formats: PNG, JPEG." The overall input limit is 8,192 tokens across modalities; no per-file size limit is documented (https://ai.google.dev/gemini-api/docs/embeddings, "Supported modalities and limits"). The catalog originals are PNG (section 5.3), so no conversion is needed; phone photos must be JPEG or PNG (not HEIC or WebP).
- Image token count per embedded image is not stated on the embeddings page. The pricing page implies about 267 tokens (see 2.6). The only documented "258 tokens" figure on the embeddings page is for PDF pages. Treat per-image tokens as UNVERIFIED.

### 2.4 Output dimensionality

- "Flexible, supports: 128 - 3072, Recommended: 768, 1536, 3072"; default 3072 (https://ai.google.dev/gemini-api/docs/embeddings and the model page). Request field `output_dimensionality` in the REST body next to `content` (example in the same page); the API reference shows it both at top level (deprecated) and inside `embedContentConfig` (https://ai.google.dev/api/embeddings).
- "Gemini Embedding 2 also auto-normalizes truncated dimensions (e.g., 768, 1536)" and the 3072 default "is always normalized" (https://ai.google.dev/gemini-api/docs/embeddings, "Ensuring quality for smaller dimensions").

### 2.5 Task semantics for image-to-image

- "You cannot use the `task_type` field for the `gemini-embedding-2` model." Task instructions are text prefixes only: `task: search result | query: {content}` for queries, `title: {title} | text: {content}` for documents, and symmetric forms `task: classification | query: ...`, `task: clustering | query: ...`, `task: sentence similarity | query: ...` (the last "Do not use this for search or retrieval") (https://ai.google.dev/gemini-api/docs/embeddings, "Task types with Embeddings 2"). The page says "The text portion of the multimodal input shouldn't include task type information", and gives no image-to-image guidance. Embed catalog and query images identically with no text part.
- One vector or many: "Adding multiple inputs directly to the `contents` parameter produces one aggregated embedding for all inputs." Wrapping each input in its own `Content` object "returns separate embeddings for each entry"; `batchEmbedContents` with one request per input also yields separate vectors (same page, "Embedding aggregation" and the migration note comparing it with `gemini-embedding-001`). So never put two images in one `parts` array.

### 2.6 Free tier and rate limits

- Pricing page, `gemini-embedding-2`: Standard tier Free Tier is "Free of charge" for text, image, audio and video input; Paid image input is $0.45 per 1M tokens ($0.00012 per image); the async Batch tier is "Not available" on the free tier and $0.225 per 1M tokens ($0.00006 per image) when paid; "Used to improve our products" is Yes for free and No for paid (https://ai.google.dev/gemini-api/docs/pricing). $0.00012 / $0.45 per 1M implies about 267 tokens per image; this is arithmetic, not a stated figure.
- Rate limits: measured in RPM, TPM (input) and RPD, per project (not per API key), RPD resets at midnight Pacific, limits vary by model and tier, and "Rate limits can be viewed in Google AI Studio"; "Specified rate limits are not guaranteed and actual capacity may vary" (https://ai.google.dev/gemini-api/docs/rate-limits). The Free tier qualification is "Active project or free trial" with no billing requirement (same page, "Usage tiers"). The page contains no per-model free-tier table; the per-model numbers live on the AI Studio rate-limit screen, which needs a signed-in session. The developer read that screen on 2026-10-10 and reported for "Gemini Embedding 2" (category "Other models"): RPM 100, TPM 30K, RPD 1K. These figures come from a transcription of the screen, not a public page, and Google states they are not guaranteed. Derived: 12,000 images at 1 request each is 12 days at 1,000 RPD; 30K TPM at about 267 tokens per image is about 112 images per minute, above the 100 RPM cap, so RPM (100) then RPD (1,000) bind first.
- What the page does publish for embeddings is only the Batch API enqueued-token cap: "Gemini Embedding" 500,000 at Tier 1, 5,000,000 at Tier 2 and 10,000,000 at Tier 3 (https://ai.google.dev/gemini-api/docs/rate-limits, "Batch API rate limits"). Not applicable to the free tier.
- Retry guidance: use exponential backoff with jitter on 429 RESOURCE_EXHAUSTED and 5xx; the official Python SDK retries transient errors up to four times with a delay between about 1 and 60 seconds (https://ai.google.dev/gemini-api/docs/troubleshooting). Whether `batchEmbedContents` counts one request or N toward RPM is UNVERIFIED, so the script should log both the per-call and per-image rates and the 429 body text.
- What the human should record by hand: open AI Studio's rate-limit page for the project, note the `gemini-embedding-2` RPM, TPM and RPD, and paste them into the report; the script's observed 429s then confirm or contradict them.

## 3. Local control model

### 3.1 DINOv2 (recommended)

- Weights and licence: `facebook/dinov2-small|base|large|giant`, all `license: apache-2.0` (https://huggingface.co/facebook/dinov2-base, raw README front matter, same for the other three). The repo says "DINOv2 code and model weights are released under the Apache License 2.0" (https://github.com/facebookresearch/dinov2, README "License"; HEAD 7764ea0). The repo README also lists non-commercial licences for the separate Cell-DINO and X-Ray-DINO weights, which are not the models above.
- Embedding dims, from each model's `config.json`: small 384, base 768, large 1024, giant 1536; `image_size` 518, `patch_size` 14 (https://huggingface.co/facebook/dinov2-base/blob/main/config.json and siblings). Parameter counts in the README table: ViT-S/14 21M, ViT-B/14 86M, ViT-L/14 300M, ViT-g/14 1,100M.
- Input resolution: the Hugging Face preprocessor resizes the shortest edge to 256, center-crops 224x224, and normalises with ImageNet mean/std (https://huggingface.co/facebook/dinov2-base/blob/main/preprocessor_config.json). The paper trains at 224 and then raises resolution to 518x518 in a short final phase (https://arxiv.org/abs/2304.07193, Section 4). The HF model interpolates position encodings for any height and width (https://github.com/huggingface/transformers/blob/main/src/transformers/models/dinov2/modeling_dinov2.py, `interpolate_pos_encoding`, used in `Dinov2Embeddings.forward`).
- Consequence for cards (derived from those two facts, not an owner recommendation): the default 256-resize plus 224 center crop of a portrait card (about 5:7) keeps roughly 63% of the height and drops the top and bottom of the card, which is where the name and number sit. A script that sets `do_center_crop=False` and passes a fixed portrait size that is a multiple of 14 (for example 224x308) keeps the whole card. Treat both settings as an experiment axis and report which was used.
- Pooled embedding: the HF `Dinov2Model` returns `pooler_output`, which is the layer-normed CLS token (`sequence_output[:, 0, :]`, https://github.com/huggingface/transformers/blob/main/src/transformers/models/dinov2/modeling_dinov2.py, `Dinov2Model.forward`). In the original repo, `forward()` returns `self.head(ret["x_norm_clstoken"])` outside training (https://github.com/facebookresearch/dinov2/blob/main/dinov2/models/vision_transformer.py, `DinoVisionTransformer.forward`, lines 348 to 353), i.e. the CLS token. The HF model card only shows `outputs.last_hidden_state`; the pooled vector is `outputs.pooler_output` (or `last_hidden_state[:, 0]`).
- Intended use: the repo MODEL_CARD lists "on image retrieval using nearest neighbors" under Direct Use and reports Oxford-H retrieval mAP in its table (https://github.com/facebookresearch/dinov2/blob/main/MODEL_CARD.md). The paper's Section 7.3 ranks a database "according to their cosine similarity with a query image" on Oxford, Paris, Met (artworks) and AmsterTime, and reports Table 9: for example Oxford-H mAP 49.5 for DINOv2 ViT-B/14 versus 19.7 for OpenCLIP ViT-G/14 and 12.7 for iBOT ViT-L/16 (https://arxiv.org/abs/2304.07193, PDF text, Section 7.3 and Table 9). The same paper notes OpenCLIP is ahead on a few fine-grained classification sets, for example Cars by 4.7 points (Section 7.2, Table 8). None of this is a trading-card benchmark.
- CPU-only: the owners state only that PyTorch is the sole loading dependency and "Installing PyTorch with CUDA support is strongly recommended" (https://github.com/facebookresearch/dinov2, README "Pretrained backbones (via PyTorch Hub)"). They publish no CPU latency figures: CPU throughput is UNVERIFIED, but ViT-S/14 (21M) and ViT-B/14 (86M) are small enough that a few hundred images is a laptop-scale job; this is an inference from the parameter counts, and the script should print its own images-per-second.
- Minimal install and inference (model card, https://huggingface.co/facebook/dinov2-base, adapted only to read the pooled output and normalise):

```bash
pip install torch transformers pillow
```

```python
from transformers import AutoImageProcessor, AutoModel
from PIL import Image
import torch

processor = AutoImageProcessor.from_pretrained('facebook/dinov2-base')
model = AutoModel.from_pretrained('facebook/dinov2-base').eval()
inputs = processor(images=Image.open('card.png').convert('RGB'), return_tensors="pt")
with torch.no_grad():
    outputs = model(**inputs)
vec = torch.nn.functional.normalize(outputs.pooler_output, dim=-1)  # (1, 768), unit length
```

The model-card lines are `AutoImageProcessor`, `AutoModel`, `outputs = model(**inputs)` and `outputs.last_hidden_state`; the `pooler_output` and `normalize` lines are the script author's addition, justified by the source cited above.

### 3.2 SigLIP (secondary)

- `google/siglip-base-patch16-224` and `google/siglip-so400m-patch14-384`, `google/siglip2-base-patch16-224` and `google/siglip2-so400m-patch14-384` are all `license: apache-2.0` (raw README front matter of each).
- Input: SigLIP 224 or 384 square, squash-resize (the SigLIP 2 preprocessor config sets `size` to 224x224 and mean/std 0.5), so a portrait card is aspect-distorted, not cropped (https://huggingface.co/google/siglip2-base-patch16-224/blob/main/preprocessor_config.json; SigLIP card: "Images are resized/rescaled to the same resolution (224x224) and normalized ... mean (0.5, 0.5, 0.5) and standard deviation (0.5, 0.5, 0.5)").
- Dims: the so400m config has vision `hidden_size` 1152; the base models use the `transformers` default `SiglipVisionConfig.hidden_size = 768` (https://huggingface.co/google/siglip-so400m-patch14-384/blob/main/config.json and https://github.com/huggingface/transformers/blob/main/src/transformers/models/siglip/configuration_siglip.py). The vision tower ends in an attention-pooling head whose output is `pooler_output` (https://github.com/huggingface/transformers/blob/main/src/transformers/models/siglip/modeling_siglip.py, `SiglipVisionTransformer`).
- Stated intended use: "zero-shot image classification and image-text retrieval" (SigLIP card), and for SigLIP 2 "zero-shot image classification and image-text retrieval, or as a vision encoder for VLMs (and other vision tasks)" (https://huggingface.co/google/siglip2-base-patch16-224). There is no image-to-image retrieval claim on either card.
- SigLIP 2 card snippet for an image vector: `model.get_image_features(**inputs)`. Caveat that matters for a script pinned to nobody's version: in `transformers` v4.57 `get_image_features` returns the pooled tensor, while on `main` it returns the vision model's output object (`BaseModelOutputWithPooling`) (compare `modeling_siglip.py` at tag v4.57.0 and at HEAD 536ecc0). Version-proof form: `model.vision_model(pixel_values=inputs.pixel_values).pooler_output`. The un-normalised pooled vector is what the model's own `forward` then L2-normalises (`image_embeds / image_embeds.norm(p=2, dim=-1, keepdim=True)` in `SiglipModel.forward`), so normalise it yourself.

### 3.3 Recommendation

Use DINOv2 ViT-B/14 (`facebook/dinov2-base`) as the control, and add SigLIP 2 base as an optional second control only if time allows. Reasons, all from the sources above: the DINOv2 owners explicitly support nearest-neighbour image retrieval from the CLS embedding and benchmark instance-level retrieval by cosine similarity, SigLIP's cards cover image-text tasks only, both are Apache-2.0, and DINOv2 lets the whole card be fed at a non-square size. It remains an unmeasured claim that DINOv2 beats the hosted models on glare and angle photos; that is what the spike measures.

## 4. Similarity and score semantics

- **Voyage:** "Voyage AI embeddings are normalized to length 1, which means that: Cosine similarity is equivalent to dot-product similarity ... Cosine similarity and Euclidean distance will result in the identical rankings." (https://docs.voyageai.com/docs/faq). This is a general statement on the FAQ, not specific to the multimodal model; the model post evaluates with cosine similarity (https://blog.voyageai.com/2026/01/15/voyage-multimodal-3-5/, "Metrics"). After manual Matryoshka truncation, re-normalise (https://docs.voyageai.com/docs/flexible-dimensions-and-quantization). Whether server-side `output_dimension` output is already unit length is UNVERIFIED, so normalise anyway; it is harmless.
- **Gemini:** 3072-d "always normalized", truncated dims auto-normalised for `gemini-embedding-2` (section 2.4). Google's sample for similarity uses sklearn `cosine_similarity` (https://ai.google.dev/gemini-api/docs/embeddings, "Semantic similarity" example). No distance metric is formally mandated.
- **DINOv2 and SigLIP:** neither returns unit vectors (sections 3.1 and 3.2); the DINOv2 paper compares images by cosine similarity (Section 7.3 and the appendix "We employ cosine similarity to compare image features"). L2-normalise before storing.
- **pgvector:** `<=>` is cosine distance, `<#>` is negative inner product, `<->` is L2 distance; "For cosine similarity, use 1 - cosine distance" (`SELECT 1 - (embedding <=> '[3,1,2]') AS cosine_similarity FROM items;`); "If vectors are normalized to length 1 ... use inner product for best performance"; `vector` indexes up to 2,000 dimensions and `halfvec` up to 4,000; zero vectors are not indexed for cosine distance (https://github.com/pgvector/pgvector README, sections "Querying", "Distances", "Indexing", "Performance"; HEAD f37c13f).
- **Unit for the report:** express every score as cosine similarity in [-1, 1] (in practice [0, 1] for these models), `1 - (embedding <=> query)`, and express the runner-up margin as the difference of two such similarities. Because all vectors are unit length after normalisation, an in-memory dot product in the script gives identical numbers to what pgvector's `<=>` will give in ticket 05, so the `confident` thresholds transfer directly. Note that scores are not comparable across models, so the report needs one threshold per provider.

## 5. Repo facts: what a Card is, reprints and image addresses

### 5.1 Card, Expansion Set, image columns

- A **Card** is "A specific printed card design within exactly one Expansion Set — number and rarity are scoped to that Expansion Set, not shared across regions" (`GLOSSARY.md:19`). An **Expansion Set** is identified by its own set code and scoped to one print edition/region (`GLOSSARY.md:15`). Print finish ("Card Variant") is not modelled, so holo and non-holo prints are the same Card (`GLOSSARY.md:37`, `docs/adr/0006-drop-card-variant-tracking-for-mvp.md`).
- Entity: `backend/internal/domain/entity/card.go:27-47`. Fields relevant to the spike: `ExpansionSetID` (line 29), `LocalID` (30, the per-set collector number), `Name` (31), `Illustrator` (33), `SourceImageURL` (39, scraped address "never served"), `ImageKey` (40, object-store key of the hosted copy, empty until hosted).
- Uniqueness: `CREATE UNIQUE INDEX idx_cards_expansion_set_id_local_id ON cards (expansion_set_id, local_id)` (`backend/internal/adapters/db/postgres/migrations/20260922000000_catalog_schema.sql:64`) and unique `(game_id, code)` on Expansion Sets (same file, line 40). Primary key is a UUIDv7 (same file, `cards` table).
- Hosted image: ADR-0016 copies each original unmodified into R2 under a deterministic key (`docs/adr/0016-card-images-are-rehosted-on-r2-and-transformed-at-read-time.md:5`), the key is `"cards/" + <card uuid>` (`backend/internal/adapters/ingestion/pokemonasia/images.go:36`), and the API returns `IMAGE_BASE_URL` plus the key, or empty when there is no key (`backend/internal/domain/mapper/catalog_mapper.go:23-28`; `docs/agents/deployment/images.md:5`). The concrete public host is deployment configuration and is not in the repo: UNVERIFIED. The spike does not need it, because the API response carries the full `imageUrl`.

### 5.2 What the script should treat as ground truth, and how to define a reprint tie

- Ground truth for each phone photo: the catalog Card's `id` (UUID) that the person physically photographed. Top-1 is correct when the highest-scoring catalog vector's card id equals it; top-5 when it is among the five highest. This is the exact definition of "top-k accuracy" in Google's ML glossary: "The percentage of times that a 'target label' appears within the first k positions of generated lists" (https://developers.google.com/machine-learning/glossary, "top-k accuracy").
- Reprints are separate Cards: the spec says "Reprints share artwork across Expansion Sets, which is why an embedding match can tie" (`.scratch/card-scanning/spec.md:140`) and "A near-tie, such as the same artwork reprinted in two Expansion Sets, is non-confident" (`.scratch/card-scanning/spec.md:83`). The repo has no column or table that links reprints; a grep for "reprint" and "artwork" found only the spec and the research notes. So there is no stored "same artwork" ground truth: UNVERIFIED that any field identifies it.
- Practical definition for the tie rate (a script decision, flagged as an approximation): group catalog Cards by `(name, illustrator)` across different `expansion_set_id` (both are real columns, `card.go:31,33`) to list candidate reprint pairs, and in addition compute, from the catalog-vs-catalog embedding similarities, the share of Cards whose nearest other Card scores within the candidate `confident` margin. Report both, and score each photo twice: strict (exact card id) and lenient (any Card in the same `(name, illustrator)` group counts). Strict is the product-relevant number because Inventory Entries track a specific Card; lenient only separates "wrong artwork" from "right artwork, wrong set". Include some known reprints deliberately in the 200-card sample (the prior note suggests 10).

### 5.3 Fetching about 200 catalog images

- Catalog search: `GET /catalog/cards` (`backend/openapi.json:905`) with query parameters `expansionSetId`, `localId`, `rarityId`, `category`, `tag`, `cardId`, `page`, `limit`. Each result is a `CardSummary` with `id`, `name`, `localId`, `illustrator`, `expansionSet`, `rarity` and `imageUrl` (`backend/openapi.json` around lines 90 to 130, `imageUrl` is the hosted address, empty when unhosted).
- Guest limits: a Guest gets one page of at most 24 results and a 401 for page greater than 1, or rarity, category or tag filters (`backend/internal/domain/service/catalog_service.go:21-22,126-131`). Authenticated callers can use `limit` up to 100 (`catalog_service.go:22,180-181`) and paging. So the script either takes a Clerk bearer token from the developer's environment (never committed) and pages 100 at a time, or, as a guest, loops over Expansion Sets with `expansionSetId` and takes one 24-card page each. Cards with an empty `imageUrl` must be skipped (spec: unhosted cards are not matchable, `.scratch/card-scanning/spec.md:85`).
- Source-site fallback: `https://asia.pokemon-card.com/id/card-img/id%08d.png`, where the number is the site's detail-page id, not `LocalID` (`backend/internal/adapters/ingestion/pokemonasia/mapper.go:156-165`, called at `ingest.go:470`; host constants at `client.go:19-28`). The ingester rate-limits and allow-lists this host (`images.go:54-64,84`); prefer the hosted copy for the spike. Sampled source files on 2026-10-10: ids 1, 50, 1000 were 299x418 RGBA PNG (about 223 KB for id 1); id 8000 was 868x1213; ids 12000 and 14000 returned HTTP 403, so the id space is sparse. Image sizes therefore differ by set, and the Voyage token estimates in section 1.6 span a 8x range; the report should record the actual `usage.image_pixels` returned by Voyage per run.
- Image types are PNG for the source (the ingester also accepts jpeg, gif and webp, `images.go:26-31`), which Voyage and Gemini both accept; Gemini would reject WebP or GIF if any hosted copy were one.
- Voyage `image_url` inputs require a content-length header, limited redirects and `robots.txt` compliance on the image host (section 1.2); a Cloudflare-fronted R2 domain may or may not satisfy that, so use `image_base64` in the spike, which also gives Voyage and Gemini identical bytes.
- Recreating the photo set: nothing in the repo prescribes the photo set's location. Ticket 01 requires it to be documented and not committed (`.scratch/card-scanning/issues/01-accuracy-spike-and-provider-pick.md:34-39`), so the script should read a local directory path and a `labels.csv` of `photo_filename,card_id` from arguments or environment variables.

## 6. Spike-design facts from primary sources only

- **Top-k accuracy** is defined by Google's ML glossary as quoted in section 5.2 (https://developers.google.com/machine-learning/glossary). The glossary adds that the target label need not be the ground-truth class, which does not apply here since the target is the photographed Card's id.
- **Provider caveats on glare, rotation, perspective, blur or low light:** none found. A text search of the fetched Voyage multimodal pages, the Voyage launch post, and the Gemini embeddings page for glare, rotation, perspective, blur, orientation and low-resolution guidance returned nothing relevant (the only hit was an unrelated "Media resolution" navigation link). The Voyage launch post evaluates only visual-document and video retrieval with NDCG@10 by cosine similarity. The DINOv2 paper makes a general "all-purpose visual features" robustness claim but reports no glare or perspective test. So the absence of owner guidance is itself a finding: robustness to glare and tilt must come from the spike's own per-condition breakdown (glare, tilt, holo), as the prior note's section 5 already proposes.
- Secondary write-ups on card-recognition accuracy were deliberately not used.

## 7. Open items for the script author and the human running it

- Create an Atlas model API key (`al-` prefix) and decide whether to add a payment method: without one the Voyage run is capped at 3 RPM and 10K TPM (section 1.6), with one it is 2,000 RPM and 2M TPM and the free token and pixel allowance still applies (https://docs.voyageai.com/docs/rate-limits). Keys go in environment variables only (`VOYAGE_API_KEY`, `GEMINI_API_KEY`), never in the repo or chat; use the `request_secret` flow from the ticket.
- Transcribe the `gemini-embedding-2` free-tier RPM, TPM and RPD from AI Studio into the report (section 2.6); no public page has them.
- Record, per provider: wall time per batch, number of 429s, the response text of the first 429, tokens or pixels consumed (`usage.image_pixels` for Voyage, `usageMetadata` for Gemini), and the images-per-second of the local model on CPU.
- Pin the `transformers` version printed in the report (section 3.2 shows its SigLIP return type changed between v4.57 and `main`; DINOv2's `pooler_output` is stable across both).
- Open question for the report, not answerable from docs: whether a single Voyage request above the 10K TPM budget is rejected on the free trial (section 1.6).

## Sources (all accessed 2026-10-10)

- Voyage on MongoDB Atlas: https://www.mongodb.com/docs/voyageai/api-reference/overview/ , https://www.mongodb.com/docs/voyageai/management/rate-limits/ , https://www.mongodb.com/docs/voyageai/management/billing/ , https://www.mongodb.com/docs/voyageai/models/multimodal-embeddings/ , https://www.mongodb.com/docs/voyageai/quickstart/ , https://www.mongodb.com/docs/voyageai/tutorials/migrate-to-atlas/ , https://www.mongodb.com/docs/voyageai/management/model-training-data/
- Voyage legacy docs: https://docs.voyageai.com/reference/multimodal-embeddings-api , https://docs.voyageai.com/docs/multimodal-embeddings , https://docs.voyageai.com/docs/rate-limits , https://docs.voyageai.com/docs/pricing , https://docs.voyageai.com/docs/faq , https://docs.voyageai.com/docs/error-codes , https://docs.voyageai.com/docs/flexible-dimensions-and-quantization , https://blog.voyageai.com/2026/01/15/voyage-multimodal-3-5/
- Voyage Python client source: https://github.com/voyage-ai/voyageai-python (`voyageai/client.py`, `voyageai/util.py`, `voyageai/object/multimodal_embeddings.py`, `voyageai/api_resources/embedding.py`), https://pypi.org/project/voyageai/
- Gemini API: https://ai.google.dev/gemini-api/docs/embeddings , https://ai.google.dev/gemini-api/docs/models/gemini-embedding-2 , https://ai.google.dev/api/embeddings , https://ai.google.dev/gemini-api/docs/pricing , https://ai.google.dev/gemini-api/docs/rate-limits , https://ai.google.dev/gemini-api/docs/troubleshooting
- DINOv2: https://github.com/facebookresearch/dinov2 (README, MODEL_CARD.md, `dinov2/models/vision_transformer.py`), https://huggingface.co/facebook/dinov2-small , https://huggingface.co/facebook/dinov2-base , https://huggingface.co/facebook/dinov2-large , https://huggingface.co/facebook/dinov2-giant , https://arxiv.org/abs/2304.07193
- SigLIP: https://huggingface.co/google/siglip-base-patch16-224 , https://huggingface.co/google/siglip-so400m-patch14-384 , https://huggingface.co/google/siglip2-base-patch16-224 , https://huggingface.co/google/siglip2-so400m-patch14-384
- Transformers source: https://github.com/huggingface/transformers (`models/dinov2/modeling_dinov2.py`, `models/siglip/modeling_siglip.py`, `models/siglip/configuration_siglip.py`, tag v4.57.0 for the older return type)
- pgvector: https://github.com/pgvector/pgvector (README)
- Google ML glossary: https://developers.google.com/machine-learning/glossary
- Repo files: `GLOSSARY.md`, `docs/adr/0006-drop-card-variant-tracking-for-mvp.md`, `docs/adr/0016-card-images-are-rehosted-on-r2-and-transformed-at-read-time.md`, `docs/agents/deployment/images.md`, `backend/internal/domain/entity/card.go`, `backend/internal/adapters/db/postgres/migrations/20260922000000_catalog_schema.sql`, `backend/internal/adapters/ingestion/pokemonasia/{client,images,mapper,ingest}.go`, `backend/internal/domain/mapper/catalog_mapper.go`, `backend/internal/domain/service/catalog_service.go`, `backend/openapi.json`, `.scratch/card-scanning/spec.md`, `.scratch/card-scanning/issues/01-accuracy-spike-and-provider-pick.md`
