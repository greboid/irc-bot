FROM reg.g5d.dev/golang@sha256:092432223820de2928ae7bd05e069c52b69349e29250695f56b793e17c32d0a2 as builder

WORKDIR /app
COPY . /app
RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-extldflags "-static"' -trimpath -ldflags=-buildid= -o main ./cmd/bot

FROM reg.g5d.dev/base@sha256:4dc61e45d55285af7ae06b1f56943e825d55793a628d085ff56be2a00a3d5039

COPY --from=builder /app/main /irc-bot
EXPOSE 8080
CMD ["/irc-bot"]
