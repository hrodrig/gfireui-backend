# syntax=docker/dockerfile:1
# Local / CI image: compile inside Docker (e.g. docker compose build).
# Release images: GoReleaser builds static binaries, then Dockerfile.release packages them.
# Multi-arch: docker buildx build --platform linux/amd64,linux/arm64 -t gfireui-backend:local .
FROM --platform=$BUILDPLATFORM golang:1.26.5-alpine AS builder
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG VERSION=dev
ARG COMMIT=unknown
ARG BRANCH=unknown
ARG BUILDDATE=unknown
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath \
	-ldflags="-s -w \
	-X 'github.com/hrodrig/gfireui-backend/internal/version.Version=${VERSION}' \
	-X 'github.com/hrodrig/gfireui-backend/internal/version.Commit=${COMMIT}' \
	-X 'github.com/hrodrig/gfireui-backend/internal/version.Branch=${BRANCH}' \
	-X 'github.com/hrodrig/gfireui-backend/internal/version.BuildDate=${BUILDDATE}'" \
	-o /out/gfireui-backend ./cmd/gfireui-backend

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=builder /out/gfireui-backend /app/gfireui-backend
COPY migrations /app/migrations
USER nonroot:nonroot
EXPOSE 8090
ENTRYPOINT ["/app/gfireui-backend"]
CMD ["serve"]
