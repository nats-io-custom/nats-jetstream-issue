package clients

import (
	cobra_utils "github.com/nats-io-custom/nats-jetstream-issue/internal/cobra_utils"

	clients_micro "github.com/nats-io-custom/nats-jetstream-issue/cmd/cli/root/clients/micro"
	clients_request_reply "github.com/nats-io-custom/nats-jetstream-issue/cmd/cli/root/clients/request_reply"

	cobra "github.com/spf13/cobra"
)

const use = "clients"

// Init command
func Init(parentCmd *cobra.Command) {
	var command = &cobra.Command{
		Use:               use,
		Short:             use,
		PersistentPreRunE: cobra_utils.ParentPersistentPreRunE,
	}

	clients_request_reply.Init(command)
	clients_micro.Init(command)

	parentCmd.AddCommand(command)

}
