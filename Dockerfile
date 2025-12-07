# syntax=docker/dockerfile:1
FROM golang:1.25 AS builder

# caching layer
RUN echo "2025-12-07T06:56:42Z" && cd / && git clone https://github.com/1lann/subway-time && \
    cd subway-time && GOPROXY=https://proxy.golang.org,direct CGO_ENABLED=0 go build .

WORKDIR /app

COPY . .

RUN CGO_ENABLED=0 go build -o subway-time .

# Execution container
FROM gcr.io/distroless/static:nonroot

COPY --from=builder /app/subway-time /subway-time

ENTRYPOINT ["/subway-time"]
