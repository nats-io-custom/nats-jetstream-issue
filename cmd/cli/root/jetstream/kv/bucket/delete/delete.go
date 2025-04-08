package delete

import (
	"fmt"

	cobra_utils "github.com/nats-io-custom/nats-jetstream-issue/internal/cobra_utils"
	shared "github.com/nats-io-custom/nats-jetstream-issue/internal/shared"

	fluffycore_utils "github.com/fluffy-bunny/fluffycore/utils"
	nats_jetstream "github.com/nats-io/nats.go/jetstream"
	zerolog "github.com/rs/zerolog"
	cobra "github.com/spf13/cobra"
	viper "github.com/spf13/viper"
)

const use = "delete"

var (
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

			if fluffycore_utils.IsEmptyOrNil(keyValueConfig.Bucket) {
				log.Error().Msg("bucket is required")
				return fmt.Errorf("bucket is required")
			}

			nc, err := appInputs.MakeConn(ctx)
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

			err = js.DeleteKeyValue(ctx, keyValueConfig.Bucket)
			if err != nil {
				log.Error().Err(err).Msg("failed to delete key value")
				return err
			}
			printer.Infof("jetstream KV Bucket %s deleted", keyValueConfig.Bucket)
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

	parentCmd.AddCommand(command)

}
