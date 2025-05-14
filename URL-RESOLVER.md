# nats-jetstream-issue

## URL-RESOLVER Operator Mode

Instructions for creating the configs and creds.

1. Run Bash in vscode
2. run the ./generate.sh script
   This will create all the files and creates a sentinel.creds file with a jwt that has the BearerToken == true in it. This is whats needed for the default_sentinel feature.
3.
4. Move the temp folder to somewhere you can work with it. `mv /tmp/DA/ Herb_temp`

In the Herb_temp/jwt folder is a {{pubkey}}.jwt which is your [auth.account.jwt](./configs/dynamic_accounts_url_resolver/auth.account.jwt)

In the herb_temp/data/stores/O folder are where you will find all the other jwts you will need.

the sentinel.jwt is used in this [server.conf](./configs/dynamic_accounts_url_resolver/server.url.resolver.conf)

## Parts

### Account Resolver

The account resolve returns account JWT(s) based on the JWT's PublicKey. It requires the 2 JWT(s) created by nsc: The system.jwt and the auth callout jwt. It also needs the NKey to sign the dynamic accounts.

When nats-server asks for an account, the auth and system accounts are served from memory and the rest are served by getting the account information from a database and minting an account.

In summary, 3 things:
a. Signing nkey (secret)
b. Auth.Account.JWT
c. System.Account.JWT

All can be strings.

### Nats Server

The server config is simple.

The sentinel JWT can be harvested out of the sentinel.creds file.

The `resolver` points to the standalone HTTP Server.

The rest is created by the generate script.

```yaml
default_sentinel: eyJ0eXAiOiJKV1QiLCJhbGciOiJlZDI1NTE5LW5rZXkifQ.eyJqdGkiOiJBMjZPTlJRWkRHWU1TVzNWRFlRQkhGRllHTlZNUTdPR0RIRlRIUldLRzRPRDRPSFFNWDJBIiwiaWF0IjoxNzQ3MTczNzgyLCJpc3MiOiJBQTZXRjRNNDNOU0NRTURSS1ZQTFhDVTNXVVY0VVJFWjRRRzdUNTI3UVBONlNQUTVTVkM2TklPVSIsIm5hbWUiOiJzZW50aW5lbCIsInN1YiI6IlVERzM1WktZSkJHQzU2VFdYWlROM0JCSU5TTENISTVRQlNaR0ozWVBGQldKRkMzQlVOQTU1QllRIiwibmF0cyI6eyJwdWIiOnsiZGVueSI6WyJcdTAwM2UiXX0sInN1YiI6eyJkZW55IjpbIlx1MDAzZSJdfSwic3VicyI6LTEsImRhdGEiOi0xLCJwYXlsb2FkIjotMSwiYmVhcmVyX3Rva2VuIjp0cnVlLCJ0eXBlIjoidXNlciIsInZlcnNpb24iOjJ9fQ.UuRh8Hwl6tqxMnYQSDi_NVJmaM54VRTS5BZ9-ohzct1Ccp9ybj1PO_hI5c6chPUcb_J9o2jT9J8YrEy4iUn1Ag
# Operator named O
operator: eyJ0eXAiOiJKV1QiLCJhbGciOiJlZDI1NTE5LW5rZXkifQ.eyJqdGkiOiJXWVpMS1FFS1VMRkRCSVNPQkVYWEEyT05WVVdJR1NGVlg1RTRVMjNMRkJYS01ZN1BMVktBIiwiaWF0IjoxNzQ3MTczNzgyLCJpc3MiOiJPQUlZS01RTkFFSUdRRk5PVVVOMlNOWTNXNklHM080UkVVU1pFRFFMVkUzQlVOSUdaS0FZWkIyVSIsIm5hbWUiOiJPIiwic3ViIjoiT0FJWUtNUU5BRUlHUUZOT1VVTjJTTlkzVzZJRzNPNFJFVVNaRURRTFZFM0JVTklHWktBWVpCMlUiLCJuYXRzIjp7InN5c3RlbV9hY2NvdW50IjoiQURRWlNRRlBHVlNaRkdKSEZGQUkzUjJYM0tPN0NaM1RFTVVIMkNNREdPNlBJSkJKRVpNWklWTUgiLCJ0eXBlIjoib3BlcmF0b3IiLCJ2ZXJzaW9uIjoyfX0.s3oqfv2du37EdUX6dWAkL9GMEpcnCIqWyaj8zidOZLCWQna1GRJTVSNVKFG30Fq2QSCXcOpOCJ4xM59BCokABA
# System Account named SYS
system_account: ADQZSQFPGVSZFGJHFFAI3R2X3KO7CZ3TEMUH2CMDGO6PIJBJEZMZIVMH

# configuration of the nats based resolver
resolver: URL(http://host.docker.internal:4299/jwt/v1/accounts/id/)
```

### AuthCallout

The app needs the callout.creds to talk to nats and the signing key to sign the users.  
Both are secrets.

## Tests

[nats server issue](https://github.com/nats-io/nats-server/issues/6775)

The url resolver is using a file based json store to store account.
it is located in the same folder as the cli.exe called `golang_db`.

The nats-server stores the jetstream data (stream + consumer configs) in a docker temp folder.  
If you want to start over clean do a

```shell
docker-compose down -v
```

You shouldn't need to delete the `golang_db` folder.

### Bring up the resolver

[code](/cmd/cli/root/callout/services/url_resolver/url_resolver.go)

```bash
go build .\cmd\cli\.

.\cli.exe callout services url_resolver --operator.nk .\configs\dynamic_accounts_url_resolver\operator.nk --auth.account.jwt .\configs\dynamic_accounts_url_resolver\auth.account.jwt --system.account.jwt .\configs\dynamic_accounts_url_resolver\system.account.jwt
```

### Bring up nats

This uses the following [conf](/configs/dynamic_accounts_url_resolver/server.url.resolver.conf).

```bash
docker-compose up
```

### Attach Auth Callout

[code](/cmd/cli/root/callout/services/operator_mode_url_resolver/operator_mode_url_resolver.go)

```bash

.\cli.exe callout services operator_mode_url_resolver --users.file .\configs\users.json --callout.creds .\configs\dynamic_accounts_url_resolver\service.creds --callout.issuer.nk .\configs\dynamic_accounts_url_resolver\C.nk
```

### Request Reply

#### Attach Request Listener

[code](/cmd/cli/root/handlers/request/request.go)

```bash
.\cli.exe handlers request --nats.user god@svc --nats.pass god --sentinel.creds .\configs\dynamic_accounts_url_resolver\sentinel.creds
```

#### Send a request/reply

[code](/cmd/cli/root/clients/request_reply/request_reply.go)

```bash
.\cli.exe clients request_reply --nats.user god@svc --nats.pass god --sentinel.creds .\configs\dynamic_accounts_url_resolver\sentinel.creds
```

### Jetstream

#### Create a stream config

[code](/cmd/cli/root/jetstream/create/create.go)

```bash
.\cli.exe jetstream create --js.name my_stream --js.subject my_stream.a --js.subject my_stream.a.>  --nats.user god@svc --nats.pass god --sentinel.creds .\configs\dynamic_accounts_url_resolver\sentinel.creds

.\cli.exe jetstream info --js.name my_stream --nats.user god@svc --nats.pass god --sentinel.creds .\configs\dynamic_accounts_url_resolver\sentinel.creds
```

#### Create a stream consumer config

[code](/cmd/cli/root/jetstream/consumer/add/add.go)

```bash
.\cli.exe jetstream consumer add --js.name my_stream --consumer.name con_my_stream --consumer.filterSubjects my_stream.a.b --nats.user god@svc --nats.pass god --sentinel.creds .\configs\dynamic_accounts_url_resolver\sentinel.creds

.\cli.exe jetstream consumer info --js.name my_stream --consumer.name con_my_stream --nats.user god@svc --nats.pass god --sentinel.creds .\configs\dynamic_accounts_url_resolver\sentinel.creds
```

#### Start up a consumer

[code](/cmd/cli/root/jetstream/consume/consume.go)

```bash
.\cli.exe jetstream consume --js.name my_stream --consumer.name con_my_stream --nats.user god@svc --nats.pass god   --nats.pass god --sentinel.creds .\configs\dynamic_accounts_url_resolver\sentinel.creds

```

### Publish a message to the stream

[code](/cmd/cli/root/jetstream/publish_one/publish_one.go)

```bash
.\cli.exe jetstream publish_one --subject my_stream.a.b --nats.pass god --sentinel.creds .\configs\dynamic_accounts_url_resolver\sentinel.creds

```

You should see messages come into the consumer.  
Wait for a bit until the consumer gets an account expired event. When we create the account we purposefully set it to a low expiration.

```shell
subject:my_stream.a.b message: {
        "message": "hello",
        "timestamp": "2025-04-08T16:27:04-07:00",
        "sequence": 0
}
subject:my_stream.a.b message: {
        "message": "hello",
        "timestamp": "2025-04-08T16:27:08-07:00",
        "sequence": 0
}
subject:my_stream.a.b message: {
        "message": "hello",
        "timestamp": "2025-04-08T16:27:09-07:00",
        "sequence": 0
}
nats: account authentication expired on connection [18]
```

Next try to publish and this will fail.

```shell
PS C:\work\nats-io-custom\nats-jetstream-issue> .\cli.exe jetstream publish_one --subject my_stream.a.b --nats.pass god --sentinel.creds .\configs\dynamic_accounts_url_resolver\sentinel.creds
{"level":"error","command":"publish_one","error":"context deadline exceeded","caller":"C:/work/nats-io-custom/nats-jetstream-issue/cmd/cli/root/jetstream/publish_one/publish_one.go:115","time":"2025-04-08T16:29:19-07:00","message":"failed to publish message"}
PS C:\work\nats-io-custom\nats-jetstream-issue>
```

This seems to be a jetstream issue and from what I can tell a go routine isn't setting up the accounts jetstream and consumers as they should be.

Try to now do our simple Request/Reply and this still works.

```shell
.\cli.exe clients request_reply --nats.user god@svc --nats.pass god --sentinel.creds .\configs\dynamic_accounts_url_resolver\sentinel.creds
request_reply
god@svc connected to nats://localhost:4222
hello, joe
hello, alice
{"level":"error","command":"request_reply","subject":"greet_junk.alice","error":"nats: no responders available for request","caller":"C:/work/nats-io-custom/nats-jetstream-issue/cmd/cli/root/clients/request_reply/request_reply.go:69","time":"2025-04-08T16:31:40-07:00","message":"failed to get response"}
PS C:\work\nats-io-custom\nats-jetstream-issue>

```

I have observed the following behavior as well. If you publish under heavy sustained load, it seems that nats keeps that account and jetstream in good standings.

Under no load the expration causes the issue and then you can't publish or consume anymore.

ABWTVME4EY6EHPMCGU6GJTHTWYHY7N3BDV6FUOLXDUDHDIMXVON7IL43
