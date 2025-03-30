package publish

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	cobra_utils "github.com/nats-io-custom/nats-jetstream-issue/internal/cobra_utils"
	shared "github.com/nats-io-custom/nats-jetstream-issue/internal/shared"

	contracts_nats "github.com/nats-io-custom/nats-jetstream-issue/internal/contracts/nats"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	fluffycore_async "github.com/fluffy-bunny/fluffycore/async"
	nats_jetstream "github.com/nats-io/nats.go/jetstream"
	async "github.com/reugn/async"
	zerolog "github.com/rs/zerolog"
	cobra "github.com/spf13/cobra"
	viper "github.com/spf13/viper"
)

const use = "publish"
const (
	serviceName = "nats-tracing-example"
)

type (
	commandInputs struct {
		DurationT           string
		PauseDurationT      string
		Subject             string
		MessageJsonTemplate string
	}
)

var messageJsonTemplate = `{
	"message": "hello",
	"timestamp": "$timestamp",
	"sequence": $sequence
}`
var (
	appInputs        = shared.NewInputs()
	appCommandInputs = commandInputs{
		DurationT:           "0s",
		PauseDurationT:      "1s",
		Subject:             "",
		MessageJsonTemplate: messageJsonTemplate,
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
			ctx = log.WithContext(ctx)
			printer := cobra_utils.NewPrinter()
			printer.EnableColors = true

			builder := di.Builder()
			di.AddInstance[*contracts_nats.NATSConnConfig](builder,
				&contracts_nats.NATSConnConfig{
					Username: appInputs.NatsUser,
					Password: appInputs.NatsPass,
					NatsUrl:  appInputs.NatsUrl,
				})
			shared.AddCommonServices(builder, serviceName)
			ctn := builder.Build()

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

			//printer.Infof("%s connected to %s", appInputs.NatsUser, nc.ConnectedUrl())

			innerJS, err := nats_jetstream.New(nc)
			if err != nil {
				printer.Errorf("Error creating JetStream context: %v", err)
				return err
			}

			js := di.Get[contracts_nats.IJetStream](ctn)
			js.SetInner(innerJS)

			durataion, err := time.ParseDuration(appCommandInputs.DurationT)
			if err != nil {
				log.Error().Err(err).Msg("failed to parse duration")
				return err
			}
			pauseDuration, err := time.ParseDuration(appCommandInputs.PauseDurationT)
			if err != nil {
				log.Error().Err(err).Msg("failed to parse pause duration")
				return err
			}
			ctxPublish, cancel := context.WithCancel(ctx)
			futurePublish := fluffycore_async.ExecuteWithPromiseAsync(func(promise async.Promise[*fluffycore_async.AsyncResponse]) {
				var err error
				defer func() {
					promise.Success(&fluffycore_async.AsyncResponse{
						Message: "End Serve - tview",
						Error:   err,
					})
				}()

				startTime := time.Now()
				sequence := 0
				quit := false
				for {
					if quit {
						break
					}
					select {
					case <-ctxPublish.Done():
						quit = true
					default:
						timestamp := time.Now().Format(time.RFC3339)
						mm := appCommandInputs.MessageJsonTemplate
						mm = strings.ReplaceAll(mm, "$timestamp", timestamp)
						mm = strings.ReplaceAll(mm, "$sequence", fmt.Sprintf("%d", sequence))

						_, err = js.Publish(ctx, appCommandInputs.Subject, []byte(mm),
							nats_jetstream.WithRetryWait(time.Second*5),
							nats_jetstream.WithRetryAttempts(100))

						if err != nil {
							log.Error().Err(err).Msg("failed to publish message")
							quit = true
							break
						}
						log.Info().Msgf("published message: %s", mm)

						sequence++
					}
					if time.Since(startTime) > durataion {
						break
					}
					time.Sleep(pauseDuration)
				}
			})

			// wait for an interrupt
			// Create a channel to receive OS signals.
			sigs := make(chan os.Signal, 1)
			// Notify the channel on interrupt signals.
			signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
			fmt.Printf("%s ", "Waiting for interrupt signal...")

			fmt.Println("Waiting for interrupt signal...")

			// Block until a signal is received.
			<-sigs
			cancel()

			futurePublish.Join()

			return nil

		},
	}
	appInputs.NatsUser = "god"
	appInputs.NatsPass = "god"

	shared.InitCommonConnFlags(appInputs, command)

	flagName := "subject"
	defaultS := appCommandInputs.Subject
	command.Flags().StringVar(&appCommandInputs.Subject, flagName, defaultS, fmt.Sprintf("[required] i.e. --%s=%s", flagName, defaultS))
	viper.BindPFlag(flagName, command.PersistentFlags().Lookup(flagName))

	flagName = "duration"
	defaultS = appCommandInputs.DurationT
	command.Flags().StringVar(&appCommandInputs.DurationT, flagName, defaultS, fmt.Sprintf("[required] i.e. --%s=%s", flagName, defaultS))
	viper.BindPFlag(flagName, command.PersistentFlags().Lookup(flagName))

	flagName = "pause.duration"
	defaultS = appCommandInputs.PauseDurationT
	command.Flags().StringVar(&appCommandInputs.PauseDurationT, flagName, defaultS, fmt.Sprintf("[required] i.e. --%s=%s", flagName, defaultS))
	viper.BindPFlag(flagName, command.PersistentFlags().Lookup(flagName))

	flagName = "message.json.template"
	defaultS = appCommandInputs.MessageJsonTemplate
	command.Flags().StringVar(&appCommandInputs.MessageJsonTemplate, flagName, defaultS, fmt.Sprintf("[required] i.e. --%s=%s", flagName, defaultS))
	viper.BindPFlag(flagName, command.PersistentFlags().Lookup(flagName))

	parentCmd.AddCommand(command)

}
