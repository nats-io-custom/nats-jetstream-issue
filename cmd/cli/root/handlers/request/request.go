package request

import (
	"os"
	"os/signal"
	"syscall"

	cobra_utils "github.com/nats-io-custom/nats-jetstream-issue/internal/cobra_utils"
	shared "github.com/nats-io-custom/nats-jetstream-issue/internal/shared"

	nats "github.com/nats-io/nats.go"
	zerolog "github.com/rs/zerolog"
	cobra "github.com/spf13/cobra"
)

const use = "request"

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

			// In addition to vanilla publish-request, NATS supports request-reply
			// interactions as well. Under the covers, this is just an optimized
			// pair of publish-subscribe operations.
			// The _request handler_ is just a subscription that _responds_ to a message
			// sent to it. This kind of subscription is called a _service_.
			// For this example, we can use the built-in asynchronous
			// subscription in the Go SDK.
			sub, _ := nc.Subscribe("greet.*", func(msg *nats.Msg) {
				// Parse out the second token in the subject (everything after greet.)
				// and use it as part of the response message.
				name := msg.Subject[len("greet")+1:]
				msg.Respond([]byte("hello, " + name))
			})

			// What happens if the service is _unavailable_? We can simulate this by
			// unsubscribing our handler from above. Now if we make a request, we will
			// expect an error.
			defer sub.Unsubscribe()

			// don't exit until sigterm
			quit := make(chan os.Signal, 1)
			signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
			<-quit
			return nil
		},
	}
	appInputs.NatsUser = "greeter"
	appInputs.NatsPass = "greeter"

	shared.InitCommonConnFlags(appInputs, command)

	parentCmd.AddCommand(command)

}
