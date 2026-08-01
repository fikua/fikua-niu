# Niu — multi-stage build (PLAN.md §5.2, §2.3).
#
# Pure Go module (modernc.org/sqlite has no CGO dependency), so
# CGO_ENABLED=0 produces a fully static binary. That is what makes a
# distroless/scratch runtime image possible at all.

FROM golang:1.25-alpine AS build

WORKDIR /src

# Cache module downloads separately from source changes.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# GOOS/GOARCH are left to the builder (docker buildx sets them per
# target platform in release.yml's linux/amd64,linux/arm64 matrix).
# -trimpath drops local build paths from the binary; -s -w strips the
# symbol table and DWARF info — smaller binary, nothing user-facing lost
# since Niu ships no CLI stack traces (S9: never leak internals anyway).
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /niu ./cmd/niu

# distroless/static: no shell, no package manager, no libc — matches the
# static binary above and gives S9 (no secrets, no OS cruft in the image)
# for free. nonroot variant runs as uid 65532 by default.
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build /niu /niu

# NIU_PORT default is 8080 (internal/config/config.go); Traefik reaches
# the container on this port over the traefik-public network, never
# published to the host.
ENV NIU_PORT=8080
EXPOSE 8080

# distroless/static:nonroot already runs as USER nonroot (65532:65532);
# explicit for clarity and to fail loudly if the base image ever changes.
USER 65532:65532

ENTRYPOINT ["/niu"]
