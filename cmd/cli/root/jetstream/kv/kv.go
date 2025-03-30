package kv

import (
	clients_jetstream_kv_bucket "github.com/nats-io-custom/nats-jetstream-issue/cmd/cli/root/jetstream/kv/bucket"
	cobra_utils "github.com/nats-io-custom/nats-jetstream-issue/internal/cobra_utils"

	cobra "github.com/spf13/cobra"
)

const use = "kv"

// Init command
func Init(parentCmd *cobra.Command) {
	var command = &cobra.Command{
		Use:               use,
		Short:             use,
		PersistentPreRunE: cobra_utils.ParentPersistentPreRunE,
	}

	clients_jetstream_kv_bucket.Init(command)

	parentCmd.AddCommand(command)

}
