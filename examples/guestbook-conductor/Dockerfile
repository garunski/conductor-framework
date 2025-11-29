# Build stage
FROM golang:1.25.4-alpine AS builder

WORKDIR /app

# Copy framework code first (needed for replace directive)
COPY pkg/ ./pkg/

# Copy conductor go mod files
COPY examples/guestbook-conductor/go.mod examples/guestbook-conductor/go.sum ./examples/guestbook-conductor/

# Copy framework go mod files (needed for replace directive)
COPY go.mod go.sum ./

# Set working directory to conductor
WORKDIR /app/examples/guestbook-conductor

# Download dependencies (replace directive will now work)
RUN go mod download

# Copy conductor source
COPY examples/guestbook-conductor/ .

# Build binary with optimizations
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -a -installsuffix cgo -o conductor .

# Final stage - distroless
FROM gcr.io/distroless/static:nonroot

WORKDIR /

COPY --from=builder /app/examples/guestbook-conductor/conductor /conductor

USER nonroot:nonroot

CMD ["/conductor"]

