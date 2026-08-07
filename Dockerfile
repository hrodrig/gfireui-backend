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
ARG APP_VERSION=0.1.0
ARG GIT_COMMIT=unknown
ARG GIT_BRANCH=unknown
ARG BUILD_DATE=unknown
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath \
	-ldflags="-s -w -X github.com/hrodrig/gfireui-backend/internal/version.Version=${APP_VERSION} -X github.com/hrodrig/gfireui-backend/internal/version.Commit=${GIT_COMMIT} -X github.com/hrodrig/gfireui-backend/internal/version.Branch=${GIT_BRANCH} -X github.com/hrodrig/gfireui-backend/internal/version.BuildDate=${BUILD_DATE}" \
	-o /out/gfireui-backend ./cmd/gfireui-backend

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=builder /out/gfireui-backend /app/gfireui-backend
COPY migrations /app/migrations
USER nonroot:nonroot
EXPOSE 8090
ENTRYPOINT ["/app/gfireui-backend"]
CMD ["serve"]
