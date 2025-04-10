package subscribe_async

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	fluffycore_async "github.com/fluffy-bunny/fluffycore/async"
	cobra_utils "github.com/nats-io-custom/nats-jetstream-issue/internal/cobra_utils"
	contracts_nats "github.com/nats-io-custom/nats-jetstream-issue/internal/contracts/nats"
	shared "github.com/nats-io-custom/nats-jetstream-issue/internal/shared"
	nats "github.com/nats-io/nats.go"
	async "github.com/reugn/async"
	zerolog "github.com/rs/zerolog"
	cobra "github.com/spf13/cobra"
	viper "github.com/spf13/viper"
)

const use = "subscribe_async"
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
		Subject:             "",
		MessageJsonTemplate: messageJsonTemplate,
		DurationT:           "0s",
		PauseDurationT:      "1s",
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
					Username:      appInputs.NatsUser,
					Password:      appInputs.NatsPass,
					NatsUrl:       appInputs.NatsUrl,
					SentinelCreds: appInputs.SentinelCreds,
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

			sub, err := nc.Subscribe(appCommandInputs.Subject,
				func(msg *nats.Msg) {
					printer.Printf(cobra_utils.Green, "received message: %s\n", msg.Data)

				})
			if err != nil {
				log.Error().Err(err).Msg("failed to subscribe to subject")
				return err
			}

			ctxConsume, cancel := context.WithCancel(ctx)
			futureConsume := fluffycore_async.ExecuteWithPromiseAsync(func(promise async.Promise[*fluffycore_async.AsyncResponse]) {
				var err error
				defer func() {
					promise.Success(&fluffycore_async.AsyncResponse{
						Message: "End Serve - tview",
						Error:   err,
					})
				}()
				quit := false

				sequence := 0
				for {
					if quit {
						break
					}
					select {
					case <-ctxConsume.Done():
						quit = true
						sub.Unsubscribe()

					default:

					}

					sequence++

				}
			})

			//printer.Printf(cobra_utils.Green, "published %d messages\n", sequence+1)
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

			futureConsume.Join()
			return nil
		},
	}
	appInputs.NatsUser = "god@SVC"
	appInputs.NatsPass = "god"

	shared.InitCommonConnFlags(appInputs, command)

	flagName := "subject"
	defaultS := appCommandInputs.Subject
	command.Flags().StringVar(&appCommandInputs.Subject, flagName, defaultS, fmt.Sprintf("[required] i.e. --%s=%s", flagName, defaultS))
	viper.BindPFlag(flagName, command.PersistentFlags().Lookup(flagName))

	flagName = "message.json.template"
	defaultS = appCommandInputs.MessageJsonTemplate
	command.Flags().StringVar(&appCommandInputs.MessageJsonTemplate, flagName, defaultS, fmt.Sprintf("[required] i.e. --%s=%s", flagName, defaultS))
	viper.BindPFlag(flagName, command.PersistentFlags().Lookup(flagName))

	parentCmd.AddCommand(command)

}
