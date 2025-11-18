# Build stage
FROM --platform=$BUILDPLATFORM golang:latest AS go_builder
WORKDIR /src

ARG TARGETARCH

# Copy only the required files
COPY go.mod .
COPY main.go .
COPY pathless.html .

# Cache Go modules
RUN --mount=type=cache,target=/go/pkg/mod \
    true

# Build static binary
RUN --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOARCH=$TARGETARCH go build -ldflags="-s -w" -o /out/pathless .

# Final stage
FROM scratch
COPY --from=go_builder /out/pathless /pathless
USER 10001
ENTRYPOINT ["/pathless"]