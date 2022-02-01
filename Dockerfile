FROM reg.g5d.dev/golang as builder

WORKDIR /app
COPY . /app
RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-extldflags "-static"' -trimpath -ldflags=-buildid= -o main ./cmd/bot

FROM reg.g5d.dev/base

COPY --from=builder /app/main /irc-bot
EXPOSE 8080
CMD ["/irc-bot"]
