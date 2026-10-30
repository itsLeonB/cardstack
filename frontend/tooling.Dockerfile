# Tooling image for both frontend packages (frontend/ and admin/). Build context is
# frontend/, but the .dockerignore excludes everything: the stack bind-mounts the
# repository at /workspace and runs commands with `compose exec`.

# Keep this in sync with the `playwright` version pinned in frontend/package.json.
ARG PLAYWRIGHT_VERSION=1.56.1
ARG BUN_VERSION=1.4.2

FROM oven/bun:${BUN_VERSION} AS bun-stage

FROM mcr.microsoft.com/playwright:v${PLAYWRIGHT_VERSION}-noble

COPY --from=bun-stage /usr/local/bin/bun /usr/local/bin/bunx /usr/local/bin/

# The stack runs this image as the host user (--user uid:gid), which has no home directory.
# /cache is world-writable so a bind mount there, or the default, takes any uid.
ENV HOME=/tmp \
    BUN_INSTALL_CACHE_DIR=/cache/bun

RUN mkdir -p /cache/bun \
    && chmod -R a+rwX /cache \
    && chmod -R a+rX /ms-playwright

WORKDIR /workspace

CMD ["sleep", "infinity"]
