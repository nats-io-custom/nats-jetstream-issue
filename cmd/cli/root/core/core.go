package core

import (
	core_publish "github.com/nats-io-custom/nats-jetstream-issue/cmd/cli/root/core/publish"
	core_subscribe_sync "github.com/nats-io-custom/nats-jetstream-issue/cmd/cli/root/core/subscribe_sync"
	cobra_utils "github.com/nats-io-custom/nats-jetstream-issue/internal/cobra_utils"
	cobra "github.com/spf13/cobra"
)

const use = "core"

// Init command
func Init(parentCmd *cobra.Command) {
	var command = &cobra.Command{
		Use:               use,
		Short:             use,
		PersistentPreRunE: cobra_utils.ParentPersistentPreRunE,
	}

	core_publish.Init(command)
	core_subscribe_sync.Init(command)

	parentCmd.AddCommand(command)

}
