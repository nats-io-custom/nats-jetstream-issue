package put

import (
	"fmt"

	cobra_utils "github.com/nats-io-custom/nats-jetstream-issue/internal/cobra_utils"
	contracts_nats "github.com/nats-io-custom/nats-jetstream-issue/internal/contracts/nats"
	shared "github.com/nats-io-custom/nats-jetstream-issue/internal/shared"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	fluffycore_utils "github.com/fluffy-bunny/fluffycore/utils"
	nats_jetstream "github.com/nats-io/nats.go/jetstream"
	zerolog "github.com/rs/zerolog"
	cobra "github.com/spf13/cobra"
	viper "github.com/spf13/viper"
)

const use = "put"

type (
	KeyValueData struct {
		Key   string
		Value string
	}
)

var (
	keyValueData   = KeyValueData{}
	appInputs      = shared.NewInputs()
	keyValueConfig = nats_jetstream.KeyValueConfig{}
)

// Init command
func Init(parentCmd *cobra.Command) {
	var command = &cobra.Command{
		Use:               use,
		Short:             use,
		PersistentPreRunE: cobra_utils.ParentPersistentPreRunE,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := shared.GetContext()
			log := zerolog.Ctx(ctx).With().Str("command", use).Logger()

			printer := cobra_utils.NewPrinter()
			printer.EnableColors = true
			printer.PrintBold(cobra_utils.Bold, use)
			builder := di.Builder()
			di.AddInstance[*contracts_nats.NATSConnConfig](builder,
				&contracts_nats.NATSConnConfig{
					Username: appInputs.NatsUser,
					Password: appInputs.NatsPass,
					NatsUrl:  appInputs.NatsUrl,
				})
			shared.AddCommonServices(builder, "nats.cli")
			ctn := builder.Build()

			if fluffycore_utils.IsEmptyOrNil(keyValueConfig.Bucket) {
				log.Error().Msg("bucket is required")
				return fmt.Errorf("bucket is required")
			}
			natsConn, err := di.TryGet[contracts_nats.INATSConnection](ctn)
			if err != nil {
				log.Error().Err(err).Msg("failed to get nats connection")
				return err
			}
			nc, err := natsConn.Conn(ctx)
			if err != nil {
				log.Error().Err(err).Msg("failed to connect to nats server")
				return err
			}

			defer nc.Drain()

			printer.Infof("%s connected to %s", appInputs.NatsUser, nc.ConnectedUrl())

			js, err := nats_jetstream.New(nc)
			if err != nil {
				printer.Errorf("Error creating JetStream context: %v", err)
				return err
			}

			store, err := js.KeyValue(ctx, keyValueConfig.Bucket)
			if err != nil {
				log.Error().Err(err).Msg("failed to get key value")
				return err
			}
			_, err = store.PutString(ctx,
				keyValueData.Key,
				keyValueData.Value)
			if err != nil {
				log.Error().Err(err).Msg("failed to put key value")
				return err
			}
			printer.Print(cobra_utils.Green, fluffycore_utils.PrettyJSON(keyValueData))
			return nil
		},
	}

	appInputs.NatsUser = "god"
	appInputs.NatsPass = "god"

	shared.InitCommonConnFlags(appInputs, command)

	flagName := "kv.bucket"
	defaultS := keyValueConfig.Bucket
	command.Flags().StringVar(&keyValueConfig.Bucket, flagName, defaultS, fmt.Sprintf("[required] i.e. --%s=%s", flagName, defaultS))
	viper.BindPFlag(flagName, command.PersistentFlags().Lookup(flagName))

	flagName = "kv.entry.key"
	defaultS = keyValueData.Key
	command.Flags().StringVar(&keyValueData.Key, flagName, defaultS, fmt.Sprintf("[required] i.e. --%s=%s", flagName, defaultS))
	viper.BindPFlag(flagName, command.PersistentFlags().Lookup(flagName))

	flagName = "kv.entry.value"
	defaultS = keyValueData.Value
	command.Flags().StringVar(&keyValueData.Value, flagName, defaultS, fmt.Sprintf("[required] i.e. --%s=%s", flagName, defaultS))
	viper.BindPFlag(flagName, command.PersistentFlags().Lookup(flagName))

	parentCmd.AddCommand(command)

}
