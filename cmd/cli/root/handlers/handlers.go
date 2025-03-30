package handlers

import (
	cobra_utils "github.com/nats-io-custom/nats-jetstream-issue/internal/cobra_utils"

	handlers_micro "github.com/nats-io-custom/nats-jetstream-issue/cmd/cli/root/handlers/micro"
	handlers_request "github.com/nats-io-custom/nats-jetstream-issue/cmd/cli/root/handlers/request"

	cobra "github.com/spf13/cobra"
)

const use = "handlers"

// Init command
func Init(parentCmd *cobra.Command) {
	var command = &cobra.Command{
		Use:               use,
		Short:             use,
		PersistentPreRunE: cobra_utils.ParentPersistentPreRunE,
	}

	handlers_request.Init(command)
	handlers_micro.Init(command)

	parentCmd.AddCommand(command)

}
