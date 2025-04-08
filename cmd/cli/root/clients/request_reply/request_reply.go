package request_reply

import (
	"time"

	cobra_utils "github.com/nats-io-custom/nats-jetstream-issue/internal/cobra_utils"
	shared "github.com/nats-io-custom/nats-jetstream-issue/internal/shared"

	zerolog "github.com/rs/zerolog"
	cobra "github.com/spf13/cobra"
)

const use = "request_reply"

var (
	appInputs = shared.NewInputs()
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

			// Now we can use the built-in `Request` method to do the service request.
			// We simply pass a nil body since that is being used right now. In addition,
			// we need to specify a timeout since with a request we are _waiting_ for the
			// reply and we likely don't want to wait forever.

			subject := "greet.joe"
			subLog := log.With().Str("subject", subject).Logger()
			rep, err := nc.Request(subject, nil, time.Second)
			if err != nil {
				subLog.Error().Err(err).Msg("failed to get response")
			} else {
				printer.Println(cobra_utils.Blue, string(rep.Data))
			}

			subject = "greet.alice"
			subLog = log.With().Str("subject", subject).Logger()
			rep, err = nc.Request(subject, nil, time.Second)
			if err != nil {
				subLog.Error().Err(err).Msg("failed to get response")
			} else {
				printer.Println(cobra_utils.Blue, string(rep.Data))
			}

			subject = "greet_junk.alice"
			subLog = log.With().Str("subject", subject).Logger()
			rep, err = nc.Request(subject, nil, time.Second)
			if err != nil {
				subLog.Error().Err(err).Msg("failed to get response")
			} else {
				printer.Println(cobra_utils.Blue, string(rep.Data))
			}

			return nil
		},
	}
	appInputs.NatsUser = "alice"
	appInputs.NatsPass = "alice"

	shared.InitCommonConnFlags(appInputs, command)

	parentCmd.AddCommand(command)

}
