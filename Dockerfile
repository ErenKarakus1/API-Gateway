FROM golang:1.26-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/gateway ./cmd/gateway

FROM alpine:3.22

WORKDIR /app

COPY --from=build /bin/gateway /app/gateway
COPY config /app/config

EXPOSE 8080

ENTRYPOINT ["/app/gateway"]
