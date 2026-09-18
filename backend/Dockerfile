FROM golang:1.26.7-alpine AS build-stage

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -buildvcs=false -ldflags='-w -s' \
    -o /api ./cmd/api

FROM gcr.io/distroless/static-debian12 AS build-release-stage

WORKDIR /

COPY --from=build-stage /api /api

EXPOSE 8080

USER nonroot:nonroot

ENTRYPOINT ["/api"]
