### Bot

IRC bot that connects to IRC and supports plugins that connect via RPC, it exposes a web server for any plugins that
need to receive web requests.  Supports password and TLS auth for connecting to the server. The bot will always have 
an RPC port and a http port open, so firewall/expose as required.

 - go build github.com/greboid/irc-bot/v2/cmd/bot
 - docker run ghcr.io/greboid/irc
    
 #### Configuration
 
 The bare minimum config is a server, nickname, realname, channel and you very likely want a list of plugins or there
 is no real functionality.
  
 There are some optional settings you might need want to change such as TLS/ports for http/RPC, but the defaults try
 to be sensible
 
 #### Example running
 
 ```
 ---
 version: "3.5"
 service:
   goplum:
     image: ghcr.io/greboid/irc
     environment:
       SERVER: irc.example.tld
       NICKNAME: bot
       REALNAME: bot
       CHANNEL: #spam
       PLUGINS: webhook=w9vwvEq5,github=XjG4WM3U
 ```
 
 ```
 bot -server irc.example.tld -nickname bot -realname bot -channel #spam -plugins webhook=w9vwvEq5,github=XjG4WM3U
 ```
