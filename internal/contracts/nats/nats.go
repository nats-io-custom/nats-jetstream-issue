package nats

import (
	"context"

	nats_go "github.com/nats-io/nats.go"
	nats_jetstream "github.com/nats-io/nats.go/jetstream"
)

type (
	NATSConnConfig struct {
		NatsUrl  string
		Username string
		Password string
	}
	INATSConnection interface {
		Conn(ctx context.Context) (*nats_go.Conn, error)
	}
	ISpan interface {
		Finish()
	}
	StartNewSpanRequest struct {
		Msg           *nats_go.Msg
		OperationName string
		Tags          map[string]string
	}
	StartNewSpanResponse struct {
		Span    ISpan
		Context context.Context
	}
	TraceProviderConfig struct {
		AppName string
	}
	ITracerProvider interface {
		StartNewSpan(ctx context.Context, request *StartNewSpanRequest) (*StartNewSpanResponse, error)
		ContextFromMessage(msg nats_jetstream.Msg) context.Context
	}
	IJetStream interface {
		nats_jetstream.JetStream
		PublishMsgAsyncWithContext(ctx context.Context, msg *nats_go.Msg, opts ...nats_jetstream.PublishOpt) (nats_jetstream.PubAckFuture, error)
		PublishAsyncWithContext(ctx context.Context, subject string, payload []byte, opts ...nats_jetstream.PublishOpt) (nats_jetstream.PubAckFuture, error)

		SetInner(inner nats_jetstream.JetStream)
	}
	MessageHandler func(ctx context.Context, msg nats_jetstream.Msg)

	IConsumer interface {
		nats_jetstream.Consumer
		SetInner(inner nats_jetstream.Consumer)

		ConsumeWithContext(handler MessageHandler, opts ...nats_jetstream.PullConsumeOpt) (nats_jetstream.ConsumeContext, error)
	}
)
