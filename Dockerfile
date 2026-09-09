# syntax=docker/dockerfile:1

# Keep the container toolchains aligned with Mise. Renovate updates each pair.

# ---- Web build ------------------------------------------------------------
# Build the frontend bundle so the Go stage can embed it. The runtime image
# does not include Node.
FROM --platform=$BUILDPLATFORM node:26.8.1-alpine AS web
WORKDIR /workspace/web

# Install dependencies against the lockfile first for layer caching.
COPY web/package.json web/pnpm-lock.yaml web/pnpm-workspace.yaml ./
RUN npm install --global "$(node --print 'require("./package.json").packageManager')"
RUN pnpm install --frozen-lockfile

COPY web/ ./
RUN pnpm build

# ---- Go build -------------------------------------------------------------
FROM --platform=$BUILDPLATFORM golang:1.27.0-alpine AS builder
ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev

RUN apk add --no-cache upx
WORKDIR /workspace

# Cache module downloads before copying source.
COPY go.mod go.sum ./
RUN go mod download

COPY cmd/ cmd/
COPY internal/ internal/
COPY web/ web/

# Overlay the freshly built frontend bundle so go:embed uses the real assets.
COPY --from=web /workspace/web/dist web/dist

RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags "-s -w -X github.com/woodleighschool/woodgate/internal/buildinfo.Version=${VERSION}" -o woodgate ./cmd/woodgate
RUN upx --best --lzma woodgate
RUN mkdir /data

# ---- Runtime --------------------------------------------------------------
FROM gcr.io/distroless/static:nonroot

WORKDIR /
COPY --from=builder /workspace/woodgate /woodgate
COPY --from=builder --chown=65532:65532 /data /data
EXPOSE 8080
USER 65532:65532
ENTRYPOINT ["/woodgate"]
