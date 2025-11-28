# Build stage
FROM golang:1.25.4-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
# Download dependencies (will use replace directive if present)
RUN go mod download

# Copy conductor source
COPY . .

# Build binary with optimizations
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -a -installsuffix cgo -o conductor .

# Final stage - distroless
FROM gcr.io/distroless/static:nonroot

WORKDIR /

COPY --from=builder /app/conductor /conductor

USER nonroot:nonroot

CMD ["/conductor"]

