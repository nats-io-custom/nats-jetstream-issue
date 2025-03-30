package publish_one

import (
	"fmt"
	"strings"
	"time"

	cobra_utils "github.com/nats-io-custom/nats-jetstream-issue/internal/cobra_utils"
	shared "github.com/nats-io-custom/nats-jetstream-issue/internal/shared"

	contracts_nats "github.com/nats-io-custom/nats-jetstream-issue/internal/contracts/nats"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	nats_jetstream "github.com/nats-io/nats.go/jetstream"
	zerolog "github.com/rs/zerolog"
	cobra "github.com/spf13/cobra"
	viper "github.com/spf13/viper"
)

const use = "publish_one"
const (
	serviceName = "nats-tracing-example"
)

type (
	commandInputs struct {
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

			js, err := nats_jetstream.New(nc)
			if err != nil {
				printer.Errorf("Error creating JetStream context: %v", err)
				return err
			}

			sequence := 0
			timestamp := time.Now().Format(time.RFC3339)
			mm := appCommandInputs.MessageJsonTemplate
			mm = strings.ReplaceAll(mm, "$timestamp", timestamp)
			mm = strings.ReplaceAll(mm, "$sequence", fmt.Sprintf("%d", sequence))

			_, err = js.Publish(ctx, appCommandInputs.Subject, []byte(mm),
				nats_jetstream.WithRetryWait(time.Second*5),
				nats_jetstream.WithRetryAttempts(100))

			if err != nil {
				log.Error().Err(err).Msg("failed to publish message")

			} else {
				log.Info().Msg(fmt.Sprintf("published message %d", sequence))
			}

			//printer.Printf(cobra_utils.Green, "published %d messages\n", sequence+1)
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
