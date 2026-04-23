## Build
FROM golang:1.25-alpine AS buildenv

ADD go.mod go.sum /

RUN go mod download

WORKDIR /app

ADD . .

ENV GO111MODULE=on

RUN go build -o team_distibutor cmd/main.go

## Deploy
FROM alpine

RUN apk add --no-cache curl

WORKDIR /

COPY --from=buildenv /app/ /

HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/healthz || exit 1

CMD ["/team_distibutor"]