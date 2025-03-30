package natsconnection

import (
	"context"
	"sync"

	contracts_nats "github.com/nats-io-custom/nats-jetstream-issue/internal/contracts/nats"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	nats_go "github.com/nats-io/nats.go"
	zerolog "github.com/rs/zerolog"
)

type (
	service struct {
		config *contracts_nats.NATSConnConfig
		conn   *nats_go.Conn
		lock   sync.Mutex
	}
)

var stemService = (*service)(nil)
var _ contracts_nats.INATSConnection = (*service)(nil)

func (s *service) Ctor(config *contracts_nats.NATSConnConfig) (contracts_nats.INATSConnection, error) {
	return &service{
		config: config,
	}, nil
}
func (s *service) Conn(ctx context.Context) (*nats_go.Conn, error) {
	//--~--~--~--~-- BARBED WIRE --~--~--~--~--//
	s.lock.Lock()
	defer s.lock.Unlock()
	//--~--~--~--~-- BARBED WIRE --~--~--~--~--//

	log := zerolog.Ctx(ctx).With().Str("service", "NATSConnection").Logger()
	if s.conn == nil {
		conn, err := nats_go.Connect(s.config.NatsUrl,
			nats_go.UserInfo(s.config.Username, s.config.Password))
		if err != nil {
			log.Error().Err(err).Msg("failed to connect to nats server")
			return nil, err
		}
		s.conn = conn
	}
	return s.conn, nil
}

func AddSingletonNATSConnection(builder di.ContainerBuilder) {
	di.AddSingleton[contracts_nats.INATSConnection](
		builder,
		stemService.Ctor,
	)
}
