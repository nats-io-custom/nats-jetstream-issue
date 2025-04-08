package delete

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

const use = "delete"

var (
	appInputs          = shared.NewInputs()
	appJetStreamConfig = shared.NewStreamConfig()

	appConsumerConfig = nats_jetstream.ConsumerConfig{
		Name:           "",
		DeliverPolicy:  nats_jetstream.DeliverAllPolicy,
		AckPolicy:      nats_jetstream.AckExplicitPolicy,
		ReplayPolicy:   nats_jetstream.ReplayInstantPolicy,
		MaxDeliver:     -1,
		MaxAckPending:  1000,
		HeadersOnly:    false,
		AckWait:        time.Second * 30,
		MaxWaiting:     512,
		FilterSubjects: []string{},
	}
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

			stream, err := js.Stream(ctx, appJetStreamConfig.Name)
			if err != nil {
				printer.Errorf("Error getting stream: %v", err)
				return err
			}

			infoStream, err := stream.Info(ctx)
			if err != nil {
				printer.Errorf("Error getting stream info: %v", err)
				return err
			}
			printer.Info(fluffycore_utils.PrettyJSON(infoStream))
			appConsumerConfig.Durable = appConsumerConfig.Name
			err = stream.DeleteConsumer(ctx, appConsumerConfig.Name)
			if err != nil {
				printer.Errorf("Error creating consumer: %v", err)
				return err
			}
			printer.Infof("successfully deleted consumer: %s", appConsumerConfig.Name)

			return nil

		},
	}
	appInputs.NatsUser = "god"
	appInputs.NatsPass = "god"

	shared.InitCommonConnFlags(appInputs, command)

	flagName := "js.name"
	defaultS := appJetStreamConfig.Name
	command.Flags().StringVar(&appJetStreamConfig.Name, flagName, defaultS, fmt.Sprintf("[required] i.e. --%s=%s", flagName, defaultS))
	viper.BindPFlag(flagName, command.PersistentFlags().Lookup(flagName))

	flagName = "consumer.name"
	defaultS = appConsumerConfig.Name
	command.Flags().StringVar(&appConsumerConfig.Name, flagName, defaultS, fmt.Sprintf("[required] i.e. --%s=%s", flagName, defaultS))
	viper.BindPFlag(flagName, command.PersistentFlags().Lookup(flagName))

	parentCmd.AddCommand(command)

}
