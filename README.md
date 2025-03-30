# nats-jetstream-issue

## Bring up nats

```bash
docker-compose up
```

## Attach Auth Callout

```bash
go build .\cmd\cli\.

.\cli.exe callout services static --nats.user auth --nats.pass auth
```

## Request Reply

### Attach Request Listener

```bash
.\cli.exe handlers request --nats.user god@SVC --nats.pass god
```

### Send a request/reply

```bash
.\cli.exe clients request_reply --nats.user god@svc --nats.pass god
```

## Jetstream

### Create a stream config

```bash
.\cli.exe jetstream create --js.name my_stream --js.subject my_stream.a --js.subject my_stream.a.>  --nats.user god@svc --nats.pass god

.\cli.exe jetstream info --js.name my_stream --nats.user god@svc --nats.pass god
```

### Create a stream consumer config

```bash
.\cli.exe jetstream consumer add --js.name my_stream --consumer.name con_my_stream --consumer.filterSubjects my_stream.a.b --nats.user god@svc --nats.pass god

.\cli.exe jetstream consumer info --js.name my_stream --consumer.name con_my_stream --nats.user god@svc --nats.pass god
```

### Start up a consumer

```bash
.\cli.exe jetstream consume --js.name my_stream --consumer.name con_my_stream --nats.user god@svc --nats.pass god   
```

### Publish a message to the stream

```bash
.\cli.exe jetstream publish_one --subject my_stream.a.b 
```
