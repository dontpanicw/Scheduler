# syntax=docker/dockerfile:1.4
FROM --platform=$BUILDPLATFORM golang:1.25.3-alpine AS build
WORKDIR /app

# НИКАКОГО apk add git — все модули публичные!
# Устанавливаем только если нужны приватные репозитории

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

COPY . .

# КЛЮЧ: используем $TARGETARCH, а не жёстко amd64
ARG TARGETARCH

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH \
    go build -v -x -trimpath -ldflags="-s -w" -o /out/scheduler ./cmd || \
    (echo "=== SCHEDULER BUILD FAILED ==="; find /tmp -name go-build*.log -exec cat {} \; 2>/dev/null || true; exit 1)

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH \
    go build -v -x -trimpath -ldflags="-s -w" -o /out/worker ./worker/cmd || \
    (echo "=== WORKER BUILD FAILED ==="; find /tmp -name go-build*.log -exec cat {} \; 2>/dev/null || true; exit 1)

# Финальные образы
FROM gcr.io/distroless/static:nonroot AS scheduler
WORKDIR /
COPY --from=build /out/scheduler /scheduler
EXPOSE 8090
USER nonroot:nonroot
ENTRYPOINT ["/scheduler"]

FROM gcr.io/distroless/static:nonroot AS worker
WORKDIR /
COPY --from=build /out/worker /worker
USER nonroot:nonroot
ENTRYPOINT ["/worker"]