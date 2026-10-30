# Tooling image for backend checks: build, vet, lint, race-enabled tests.
# The image copies nothing; the compose file bind-mounts the repository at /workspace.
# glibc (Debian), not alpine: `go test -race` needs cgo, which needs gcc.

ARG GO_VERSION=1.26.3
FROM golang:${GO_VERSION}-bookworm

ARG TARGETARCH
# Pinned to the latest release at the time of writing; bump the version and checksums together.
ARG GOLANGCI_LINT_VERSION=2.15.0

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates curl gcc git \
    && rm -rf /var/lib/apt/lists/*

RUN set -eux; \
    case "${TARGETARCH}" in \
        amd64) sha256=9b7fe6a6bf6334b98decf8d86b11415ea7889afdabdd1959fdce23af2a9ebcea ;; \
        arm64) sha256=b4d8f17068b35ee6060cc2f69a16fb48740d706fbba81661c318649c114a6998 ;; \
        *) echo "unsupported arch: ${TARGETARCH}" >&2; exit 1 ;; \
    esac; \
    archive="golangci-lint-${GOLANGCI_LINT_VERSION}-linux-${TARGETARCH}.tar.gz"; \
    curl -fsSL -o "/tmp/${archive}" \
        "https://github.com/golangci/golangci-lint/releases/download/v${GOLANGCI_LINT_VERSION}/${archive}"; \
    echo "${sha256}  /tmp/${archive}" | sha256sum -c -; \
    tar -xzf "/tmp/${archive}" -C /usr/local/bin --strip-components=1 \
        "golangci-lint-${GOLANGCI_LINT_VERSION}-linux-${TARGETARCH}/golangci-lint"; \
    rm "/tmp/${archive}"; \
    golangci-lint version

# The container runs as the host user (`--user`), which has no home directory.
# Caches live under /cache so a bind mount there persists them between runs.
ENV HOME=/tmp \
    GOCACHE=/cache/go-build \
    GOMODCACHE=/cache/go-mod \
    GOFLAGS=-buildvcs=false

RUN mkdir -p /cache/go-build /cache/go-mod \
    && chmod 0777 /cache /cache/go-build /cache/go-mod

WORKDIR /workspace/backend

# Long-lived so `docker compose exec` can run each step in it.
CMD ["sleep", "infinity"]
