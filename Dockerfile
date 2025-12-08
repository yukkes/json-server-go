# Build stage
FROM golang:bookworm AS builder

ENV GOTOOLCHAIN=auto
WORKDIR /app

# Copy dependency files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY main.go ./
COPY db ./db
COPY server ./server
COPY handlers ./handlers

# Build static binary with optimization flags
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-s -w" -o json-server main.go

# Run stage
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

# Copy artifacts with non-root ownership
COPY --chown=nonroot:nonroot --from=builder /app/json-server ./json-server
COPY --chown=nonroot:nonroot public ./public
COPY --chown=nonroot:nonroot db_sample.json .

USER nonroot:nonroot
EXPOSE 3000

ENTRYPOINT ["./json-server"]
CMD ["db_sample.json", "--host", "0.0.0.0", "--port", "3000", "--static", "./public"]