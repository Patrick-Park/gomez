## SMTP Package

The SMTP package can spawn a highly configurable Sendmail Transfer Protocol server that implements the minimum viable implementation according to [RFC 5321](http://tools.ietf.org/html/rfc5321)

### Terminology

##### Server
Contains the server configuration settings, it's command specification and an attached mailbox which comes from a different package and is served as an interface to allow testability and various implementations (such as PostgreSQL version called PostBox, or a Redis version called RedBox, etc.).

The Server can spawn clients based on incoming connections, through which it exposes it's commands and services as an SMTP Server [interface](https://github.com/gbbr/gomez/blob/master/smtp/server.go#L16). Besides running commands, it can also query the mailbox for users and digest completed messages by adding the necessarry transitional headers and queueing them in the mailbox.

##### CommandSpec
The command spec is the server command specification. It is simply a map of commands to actions. The CommandSpec is attached to a server and will be exposed to connected clients.

##### Client
A client is a server-independent connection and serves as context for the current transaction. It holds the current SMTP state (such as HELLO, MAIL or RECIPIENT), the gathered information and the created message. It's scope is to pick-up commands from the connection and serve them to be run by the server on the command spec.

##### Commands
The commands.go file contains all the commands that are initialized on the server at instantiation.

### Usage

A new server can be spawned using the `smtp.Start` method, for example:

```go
var mailbox gomez.Mailbox

mailbox = new(PostBox)
config = smtp.Config{
  ListenAddr: ":25",
  Hostname: "mydomain.com",
}

smtp.Start(mailbox, config)
```
