# Core Nats

## Spin up nats

```shell
docker-compose -f .\docker-compose-static-callout.yml up
```

## Attach Callout

[code](/cmd/cli/root/callout/services/static/static.go)

```shell
.\cli.exe callout services static --users.file .\configs\users.json
```

## Publish a message

```shell
.\cli.exe core publish --nats.user god@svc --nats.pass god --subject core.herb
```

## subscribe sync

```shell
.\cli.exe core subscribe_sync --nats.user god@svc --nats.pass god --subject core.*
```
