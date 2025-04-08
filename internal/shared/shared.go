package shared

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	services_natsconnection "github.com/nats-io-custom/nats-jetstream-issue/internal/services/natsconnection"

	contracts_nats "github.com/nats-io-custom/nats-jetstream-issue/internal/contracts/nats"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	fluffycore_utils "github.com/fluffy-bunny/fluffycore/utils"
	status "github.com/gogo/status"
	nats "github.com/nats-io/nats.go"
	nats_jetstream "github.com/nats-io/nats.go/jetstream"
	zerolog "github.com/rs/zerolog"
	cobra "github.com/spf13/cobra"
	viper "github.com/spf13/viper"
	codes "google.golang.org/grpc/codes"
)

type (
	Permissions struct {
		Allow []string `json:"allow"`
		Deny  []string `json:"deny"`
	}
	User struct {
		Username        string      `json:"username"`
		Password        string      `json:"password"`
		Sub             Permissions `json:"sub"`
		Pub             Permissions `json:"pub"`
		AllowedAccounts []string    `json:"allowedAccounts"`
	}
	Users struct {
		Users []User `json:"users"`
	}
)

var _ctx context.Context

func SetContext(ctx context.Context) {
	_ctx = ctx
}
func GetContext() context.Context {
	return _ctx
}

type Inputs struct {
	NatsUrl               string   `json:"natsUrl"`
	NatsCreds             string   `json:"natsCreds"`
	IssuerSeed            string   `json:"issuerSeed"`
	NatsUser              string   `json:"natsUser"`
	NatsPass              string   `json:"natsPass"`
	XKeySeed              string   `json:"xkeySeed"`
	SigningKeyFiles       []string `json:"signingKeyFiles"`
	UsersFile             string   `json:"usersFile"`
	OperatorKeyFile       string   `json:"operatorKeyFile"`
	SysCredsFile          string   `json:"sysCredsFile"`
	OperatorNKeyFile      string   `json:"operatorNKeyFile"`
	CalloutIssuerNKeyFile string   `json:"calloutIssuerNKeyFile"`
	AuthAccountJWTFile    string   `json:"authAccountJWTFile"`
	SystemAccountJWTFile  string   `json:"systemAccountJWTFile"`
	CalloutCreds          string   `json:"calloutCreds"`
	SentinelCreds         string   `json:"sentinelCreds"`
}

func NewInputs() *Inputs {
	return &Inputs{
		NatsUrl: "nats://localhost:4222",
	}
}

func InitCommonConnFlags(input *Inputs, command *cobra.Command) {
	flagName := "nats.url"
	defaultS := input.NatsUrl
	command.Flags().StringVar(&input.NatsUrl, flagName, defaultS, fmt.Sprintf("[required] i.e. --%s=%s", flagName, defaultS))
	viper.BindPFlag(flagName, command.PersistentFlags().Lookup(flagName))

	flagName = "nats.user"
	defaultS = input.NatsUser
	command.Flags().StringVar(&input.NatsUser, flagName, defaultS, fmt.Sprintf("[required] i.e. --%s=%s", flagName, defaultS))
	viper.BindPFlag(flagName, command.PersistentFlags().Lookup(flagName))

	flagName = "nats.pass"
	defaultS = input.NatsPass
	command.Flags().StringVar(&input.NatsPass, flagName, defaultS, fmt.Sprintf("[required] i.e. --%s=%s", flagName, defaultS))
	viper.BindPFlag(flagName, command.PersistentFlags().Lookup(flagName))

	flagName = "sentinel.creds"
	defaultS = input.SentinelCreds
	command.Flags().StringVar(&input.SentinelCreds, flagName, defaultS, fmt.Sprintf("[required] i.e. --%s=%s", flagName, defaultS))
	viper.BindPFlag(flagName, command.PersistentFlags().Lookup(flagName))

}

func (appInputs *Inputs) MakeConn(ctx context.Context) (*nats.Conn, error) {
	log := zerolog.Ctx(ctx).With().Str("command", "MakeConn").Logger()
	opts := []nats.Option{}
	if fluffycore_utils.IsNotEmptyOrNil(appInputs.NatsCreds) {
		if !FileExists(appInputs.NatsCreds) {
			log.Error().Msgf("nats creds file does not exist: %s", appInputs.NatsCreds)
			return nil, status.Error(codes.NotFound, fmt.Sprintf("nats creds file does not exist: %s", appInputs.NatsCreds))
		}
		opts = append(opts, nats.UserCredentials(appInputs.NatsCreds))
	}
	if fluffycore_utils.IsEmptyOrNil(appInputs.NatsUser) {
		log.Error().Msg("nats user is required")
		return nil, status.Error(codes.InvalidArgument, "nats user is required")
	}
	if fluffycore_utils.IsEmptyOrNil(appInputs.NatsPass) {
		log.Error().Msg("nats pass is required")
		return nil, status.Error(codes.InvalidArgument, "nats pass is required")
	}
	opts = append(opts, nats.UserInfo(appInputs.NatsUser, appInputs.NatsPass))

	if fluffycore_utils.IsNotEmptyOrNil(appInputs.SentinelCreds) {
		if !FileExists(appInputs.SentinelCreds) {
			log.Error().Msgf("sentinel creds file does not exist: %s", appInputs.SentinelCreds)
			return nil, status.Error(codes.NotFound, fmt.Sprintf("sentinel creds file does not exist: %s", appInputs.SentinelCreds))
		}
		opts = append(opts, nats.UserCredentials(appInputs.SentinelCreds))
	}

	nc, err := nats.Connect(
		appInputs.NatsUrl,
		opts...,
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to connect to nats server")
		return nil, err
	}
	return nc, nil

}

func FileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

func LoadFile(filename string) string {
	data, err := os.ReadFile(filename)
	if err != nil {
		return ""
	}
	return string(data)
}
func LoadUsersData(filename string) (*Users, error) {
	if !FileExists(filename) {
		return nil, fmt.Errorf("file does not exist: %s", filename)
	}
	data := LoadFile(filename)
	usersData := &Users{}
	err := json.Unmarshal([]byte(data), usersData)
	return usersData, err
}

type (
	StreamConfigOption func(*nats_jetstream.StreamConfig)
)

func WithStreamName(name string) StreamConfigOption {
	return func(c *nats_jetstream.StreamConfig) {
		c.Name = name
	}
}

func WithStreamSubject(subjects ...string) StreamConfigOption {
	return func(c *nats_jetstream.StreamConfig) {
		c.Subjects = append(c.Subjects, subjects...)
	}
}
func NewStreamConfig(opts ...StreamConfigOption) *nats_jetstream.StreamConfig {
	sc := &nats_jetstream.StreamConfig{
		Name:              "",
		Subjects:          []string{},
		Storage:           nats_jetstream.FileStorage,
		Replicas:          1,
		Retention:         nats_jetstream.LimitsPolicy,
		Discard:           nats_jetstream.DiscardOld,
		MaxMsgs:           -1,
		MaxMsgsPerSubject: -1,
		MaxMsgSize:        -1,
		MaxConsumers:      -1,
		MaxBytes:          -1,
		AllowRollup:       false,
		DenyPurge:         false,
		DenyDelete:        false,
		AllowDirect:       true,
		NoAck:             false,
	}
	for _, opt := range opts {
		opt(sc)
	}
	return sc
}

func AddCommonServices(builder di.ContainerBuilder, appName string) {
	services_natsconnection.AddSingletonNATSConnection(builder)
	di.AddInstance[*contracts_nats.TraceProviderConfig](builder, &contracts_nats.TraceProviderConfig{
		AppName: appName,
	})
}
