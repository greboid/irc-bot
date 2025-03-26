FROM golang:1.24 as builder
WORKDIR /app
COPY . /app
RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-extldflags "-static"' -trimpath -ldflags=-buildid= -o main ./cmd/bot

FROM ghcr.io/greboid/dockerbase/nonroot:1.20250326.0

COPY --from=builder /app/main /irc-bot
EXPOSE 8080
CMD ["/irc-bot"]
