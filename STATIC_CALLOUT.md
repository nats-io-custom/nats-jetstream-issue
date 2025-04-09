# Static Callout

## Spin up nats

```shell
docker-compose -f .\docker-compose-static-callout.yml up
```

## Attach Callout

[code](/cmd/cli/root/callout/services/static/static.go)

```shell
.\cli.exe callout services static --users.file .\configs\users.json
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
