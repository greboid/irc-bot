### Bot

IRC bot that connects to IRC and supports plugins that connect via RPC, it exposes a web server for any plugins that
need to receive web requests.  Supports password and TLS auth for connecting to the server. The bot will always have 
an RPC port and a http port open. The RPC port only needs to be reachable by your plugins, it should not be exposed to the
internet as any plugin token holder can send messages to IRC. The http port only needs exposing if a plugin you use
receives web requests (e.g. webhooks).
    
 #### Configuration
 
 The bare minimum config is a server, nickname, realname, channel and you very likely want a list of plugins or there
 is no real functionality.
  
 There are some optional settings you might need want to change such as TLS/ports for http/RPC, but the defaults try
 to be sensible

 Each flag can also be set via an environment variable, the flag name uppercased with `-` and `.` replaced by `_`
 (e.g. `-sasl-user` becomes `SASL_USER`).

 | Flag             | Env             | Default       | Description                                                                                          |
 |------------------|-----------------|---------------|------------------------------------------------------------------------------------------------------|
 | `-server`        | `SERVER`        |               | IRC server to connect to (required)                                                                  |
 | `-password`      | `PASSWORD`      |               | Server password, if required                                                                         |
 | `-tls`           | `TLS`           | `true`        | Connect with TLS                                                                                     |
 | `-nick`          | `NICK`          |               | Nickname to use                                                                                      |
 | `-realname`      | `REALNAME`      |               | 'Real name' to use                                                                                   |
 | `-channel`       | `CHANNEL`       |               | Channels to join on connect, comma separated list (with optional space separated key with each channel) |
 | `-sasl-auth`     | `SASL_AUTH`     | `false`       | Authenticate via SASL                                                                                |
 | `-sasl-user`     | `SASL_USER`     |               | SASL username                                                                                        |
 | `-sasl-pass`     | `SASL_PASS`     |               | SASL password                                                                                        |
 | `-rpc-port`      | `RPC_PORT`      | `8001`        | gRPC server port                                                                                     |
 | `-web-port`      | `WEB_PORT`      | `8000`        | Web port for http server                                                                             |
 | `-plugins`       | `PLUGINS`       |               | Comma separated list of plugins, `name=token`                                                        |
 | `-flood-profile` | `FLOOD_PROFILE` | `restrictive` | Flood profile: `restrictive`, `unlimited`                                                            |
 | `-log.level`     | `LOG_LEVEL`     |               | Lowest level of logs that should be output                                                           |
 | `-log.format`    | `LOG_FORMAT`    | `text`        | Format of log output (`json` or `text`)                                                              |
 
 #### Plugins

 Available plugins (by me, there might be others):

 - [irc-adze](https://github.com/greboid/irc-adze) - adze notifications
 - [irc-github](https://github.com/greboid/irc-github) - GitHub/Forgejo notifications
 - [irc-goplum](https://github.com/greboid/irc-goplum) - goplum notifications
 - [irc-webhook](https://github.com/greboid/irc-webhook) - relays HTTP webhook requests

 Each entry in `-plugins` is `name=token`. The token is a secret you choose and also give to that plugin, the plugin
 uses it to authenticate when connecting to the RPC port. Use a different token per plugin.

 The RPC port uses TLS with a self-signed certificate generated on each start, so plugins need to skip certificate
 verification when connecting.

 The http port is plain HTTP. Plugins register a path prefix and requests under it are passed to that plugin, anything
 else returns a 404. If you need HTTPS, put it behind a reverse proxy.

 #### Flood profiles

 - `restrictive` - allows a burst of 3 messages, then 1 message every 2.5 seconds
 - `unlimited` - no rate limiting, only use this if the server allows it

 #### Example running
 
 ```
 ---
 services:
   bot:
     image: ghcr.io/greboid/irc
     environment:
       SERVER: irc.example.tld
       NICK: bot
       REALNAME: bot
       CHANNEL: "#spam"
       PLUGINS: webhook=w9vwvEq5,github=XjG4WM3U
 ```
 
 ```
 bot -server irc.example.tld -nick bot -realname bot -channel "#spam" -plugins webhook=w9vwvEq5,github=XjG4WM3U
 ```
