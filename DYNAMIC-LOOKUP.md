# nats-jetstream-issue

## Dynamic Accounts Operator Mode

This is based on the [Synadia Dynamic Accounts](https://github.com/synadia-io/callout.go/tree/main/examples/dynamic_accounts) example.

it is located in the same folder as the cli.exe called `golang_db`.

The nats-server stores the jetstream data (stream + consumer configs) in a docker temp folder.  
If you want to start over clean do a

```shell
docker-compose down -v
```

### Bring up nats

This uses the following [conf](/configs/dynamic_accounts_url_resolver/server.original.conf).

```bash
docker-compose -f .\docker-compose-decentalized-dynamic-accounts.yml up
```

### Attach Auth Callout with Dynamic Accounts via "$SYS.REQ.ACCOUNT.\*.CLAIMS.LOOKUP"

[code](/cmd/cli/root/callout/services/decentralized_dynamic_accounts/decentralized_dynamic_accounts.go)

```shell

.\cli.exe callout services decentralized_dynamic_accounts --users.file .\configs\users.json --callout.creds .\configs\dynamic_accounts_url_resolver\service.creds --callout.issuer.nk .\configs\dynamic_accounts_url_resolver\C.nk --sys.creds .\configs\dynamic_accounts_url_resolver\sys.creds --auth.account.jwt .\configs\dynamic_accounts_url_resolver\auth.account.jwt --system.account.jwt .\configs\dynamic_accounts_url_resolver\system.account.jwt --operator.nk .\configs\dynamic_accounts_url_resolver\operator.nk
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
