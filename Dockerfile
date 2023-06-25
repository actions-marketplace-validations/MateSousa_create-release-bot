FROM golang:1.20 as builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o create-release-bot ./cmd/create-release-bot

FROM alpine:3.14

RUN apk add --no-cache ca-certificates

COPY --from=builder /app/create-release-bot /usr/local/bin/create-release-bot

ENTRYPOINT ["/usr/local/bin/create-release-bot"]


