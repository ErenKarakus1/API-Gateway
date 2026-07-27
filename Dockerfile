FROM golang:1.26-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/gateway ./cmd/gateway

FROM alpine:3.22

WORKDIR /app

RUN apk add --no-cache wget

COPY --from=build /bin/gateway /app/gateway
COPY config /app/config

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -qO- http://localhost:8080/health || exit 1

ENTRYPOINT ["/app/gateway"]
