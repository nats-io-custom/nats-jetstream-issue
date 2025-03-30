package services

import (
	cobra_utils "github.com/nats-io-custom/nats-jetstream-issue/internal/cobra_utils"

	callout_services_static "github.com/nats-io-custom/nats-jetstream-issue/cmd/cli/root/callout/services/static"

	cobra "github.com/spf13/cobra"
)

const use = "services"

// Init command
func Init(parentCmd *cobra.Command) {
	var command = &cobra.Command{
		Use:               use,
		Short:             use,
		PersistentPreRunE: cobra_utils.ParentPersistentPreRunE,
	}

	callout_services_static.Init(command)

	parentCmd.AddCommand(command)

}
