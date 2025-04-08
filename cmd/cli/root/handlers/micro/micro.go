package micro

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	cobra_utils "github.com/nats-io-custom/nats-jetstream-issue/internal/cobra_utils"
	shared "github.com/nats-io-custom/nats-jetstream-issue/internal/shared"

	nats_micro "github.com/nats-io/nats.go/micro"
	zerolog "github.com/rs/zerolog"
	cobra "github.com/spf13/cobra"
)

const use = "micro"

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

			// the patter is svc.<service>.<method>
			// request handler
			echoHandler := func(req nats_micro.Request) {
				subject := req.Subject()

				// extract the service and method
				printer.Printf(cobra_utils.Blue, "Received request on subject:%s: %s", subject, string(req.Data()))

				message := fmt.Sprintf("subject: %s => %s", subject, string(req.Data()))
				req.Respond([]byte(message))
			}

			srv, err := nats_micro.AddService(nc, nats_micro.Config{
				Name:    "EchoService",
				Version: "1.0.0",
				// base handler
				Endpoint: &nats_micro.EndpointConfig{
					Subject: "greet.*",
					Handler: nats_micro.HandlerFunc(echoHandler),
				},
			})
			if err != nil {
				log.Error().Err(err).Msg("failed to add service")
				return err
			}
			defer srv.Stop()
			// print the help
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
