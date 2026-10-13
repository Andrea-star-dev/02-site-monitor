# syntax=docker/dockerfile:1
FROM golang:1.26-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/monitor ./cmd/monitor

FROM alpine:3.20
RUN addgroup -g 10001 monitor && adduser -D -u 10001 -G monitor monitor \
    && mkdir -p /app/config /app/data \
    && chown -R monitor:monitor /app

WORKDIR /app
COPY --from=build /out/monitor /app/monitor
COPY config/targets.yaml.example /app/config/targets.yaml.example

USER monitor
VOLUME ["/app/data"]

ENTRYPOINT ["/app/monitor"]
CMD ["--targets", "/app/config/targets.yaml", "--db", "/app/data/site_monitor.db"]
