# syntax=docker/dockerfile:1

### builder
FROM golang:1.23 AS builder
WORKDIR /src
# Download dependencies first (caching)
COPY go.mod go.sum ./
RUN go mod download
# Bring in the rest of the code and build
COPY . .
RUN CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64 \
    go build -o smartctld .

### 
FROM scratch
COPY --from=builder /src/smartctld /smartctld
USER 1000
EXPOSE 8080
ENTRYPOINT ["/smartctld"]