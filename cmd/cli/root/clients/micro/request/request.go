package request

import (
	"context"
	"fmt"
	"time"

	cobra_utils "github.com/nats-io-custom/nats-jetstream-issue/internal/cobra_utils"
	shared "github.com/nats-io-custom/nats-jetstream-issue/internal/shared"

	zerolog "github.com/rs/zerolog"
	cobra "github.com/spf13/cobra"
	viper "github.com/spf13/viper"
)

const use = "request"

var (
	appInputs          = shared.NewInputs()
	requestData string = "hello"
	durationS          = "0s"
	subject            = "greet.joe"
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

			duration, err := time.ParseDuration(durationS)
			if err != nil {
				log.Error().Err(err).Msg("failed to parse duration")
				return err
			}
			nc, err := appInputs.MakeConn(ctx)
			if err != nil {
				log.Error().Err(err).Msg("failed to connect to nats server")
				return err
			}
			defer nc.Drain()

			printer.Infof("%s connected to %s", appInputs.NatsUser, nc.ConnectedUrl())

			doSubRequest := func(ctx context.Context, subject string) error {
				log := zerolog.Ctx(ctx).With().Str("subject", subject).Logger()
				subReply := fmt.Sprintf("%s.reply", subject)
				log = log.With().Str("subReply", subReply).Logger()
				srd := fmt.Sprintf("%s: %s", subject, requestData)

				log.Info().Msgf("Sending request: %s", srd)
				sub, err := nc.SubscribeSync(subReply)
				if err != nil {
					log.Error().Err(err).Msg("failed to subscribe")
					return err
				}
				now := time.Now()
				for {
					err = nc.PublishRequest(subject, subReply, []byte(srd))
					if err != nil {
						log.Error().Err(err).Msg("failed to get response")
						return err
					} else {

						msg, err := sub.NextMsg(1 * time.Second)
						if err != nil {
							log.Error().Err(err).Msg("failed to get response")
						} else {
							printer.Printf(cobra_utils.Blue, "Received: %s\n", string(msg.Data))

						}
					}
					time.Sleep(500 * time.Millisecond)
					if time.Since(now) > duration {
						break
					}
				}

				return nil
			}

			doSubRequest(ctx, subject)

			return nil
		},
	}
	appInputs.NatsUser = "alice"
	appInputs.NatsPass = "alice"

	shared.InitCommonConnFlags(appInputs, command)

	flagName := "request.data"
	defaultS := requestData
	command.Flags().StringVar(&requestData, flagName, defaultS, fmt.Sprintf("[required] i.e. --%s=%s", flagName, defaultS))
	viper.BindPFlag(flagName, command.PersistentFlags().Lookup(flagName))

	flagName = "request.duration"
	defaultS = durationS
	command.Flags().StringVar(&durationS, flagName, defaultS, fmt.Sprintf("[required] i.e. --%s=%s", flagName, defaultS))
	viper.BindPFlag(flagName, command.PersistentFlags().Lookup(flagName))

	flagName = "request.subject"
	defaultS = subject
	command.Flags().StringVar(&subject, flagName, defaultS, fmt.Sprintf("[required] i.e. --%s=%s", flagName, defaultS))
	viper.BindPFlag(flagName, command.PersistentFlags().Lookup(flagName))

	parentCmd.AddCommand(command)

}
