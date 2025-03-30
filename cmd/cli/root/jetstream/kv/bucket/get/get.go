package get

import (
	"fmt"
	"time"

	cobra_utils "github.com/nats-io-custom/nats-jetstream-issue/internal/cobra_utils"
	shared "github.com/nats-io-custom/nats-jetstream-issue/internal/shared"

	fluffycore_utils "github.com/fluffy-bunny/fluffycore/utils"
	nats_jetstream "github.com/nats-io/nats.go/jetstream"
	zerolog "github.com/rs/zerolog"
	cobra "github.com/spf13/cobra"
	viper "github.com/spf13/viper"
)

const use = "get"

type (
	KeyValueData struct {
		Key   string
		Value string
	}

	KeyValueEntry struct {
		// Bucket is the bucket the data was loaded from.
		Bucket string

		// Key is the name of the key that was retrieved.
		Key string

		// Value is the retrieved value.
		Value string

		// Revision is a unique sequence for this value.
		Revision uint64

		// Created is the time the data was put in the bucket.
		Created time.Time

		// Delta is distance from the latest value (how far the current sequence
		// is from the latest).
		Delta uint64

		// Operation returns Put or Delete or Purge, depending on the manner in
		// which the current revision was created.
		Operation nats_jetstream.KeyValueOp
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

			store, err := js.KeyValue(ctx, keyValueConfig.Bucket)
			if err != nil {
				log.Error().Err(err).Msg("failed to get key value")
				return err
			}
			entry, err := store.Get(ctx,
				keyValueData.Key)
			if err != nil {
				log.Error().Err(err).Msg("failed to put key value")
				return err
			}
			toKeyValueEntry := func(entry nats_jetstream.KeyValueEntry) *KeyValueEntry {

				return &KeyValueEntry{
					Bucket:    entry.Bucket(),
					Key:       entry.Key(),
					Value:     string(entry.Value()),
					Revision:  entry.Revision(),
					Created:   entry.Created(),
					Delta:     entry.Delta(),
					Operation: entry.Operation(),
				}
			}
			printer.Print(cobra_utils.Green, fluffycore_utils.PrettyJSON(toKeyValueEntry(entry)))
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

	parentCmd.AddCommand(command)

}
