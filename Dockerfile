# Build the frontend
FROM node:25-alpine AS frontend
WORKDIR /workspace/frontend
COPY frontend/package*.json ./
RUN npm ci --no-audit --no-fund
COPY frontend .
COPY backend/api/openapi.yaml /workspace/backend/api/openapi.yaml
RUN npm run gen:api
ENV NODE_ENV=production
RUN npm run build

# Build the woodgate binary
FROM golang:1.26.2 AS backend
ARG TARGETOS
ARG TARGETARCH
ARG LDFLAGS

WORKDIR /workspace
# Copy the Go Modules manifests
COPY backend/go.mod go.mod
COPY backend/go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source and sqlc config
COPY backend/cmd/ cmd/
COPY backend/internal/ internal/

# Build
# the GOARCH has not a default value to allow the binary be built according to the host where the command
# was called. For example, if we call make docker-build in a local env which has the Apple Silicon M1 SO
# the docker BUILDPLATFORM arg will be linux/arm64 when for Apple x86 it will be linux/amd64. Therefore,
# by leaving it empty we can ensure that the container and binary shipped on it will have the same platform.
RUN go generate ./...
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} \
	go build -trimpath -buildvcs=true \
		-ldflags="${LDFLAGS} -w -s" \
		-o woodgate cmd/woodgate/main.go

# Use distroless as minimal base image to package the woodgate binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=backend /workspace/woodgate .
COPY --from=frontend /workspace/frontend/dist ./frontend

USER 65532:65532
EXPOSE 8080

ENTRYPOINT ["/woodgate"]
