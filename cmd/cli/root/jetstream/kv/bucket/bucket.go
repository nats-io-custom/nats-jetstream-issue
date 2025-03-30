package bucket

import (
	clients_jetstream_kv_bucket_create "github.com/nats-io-custom/nats-jetstream-issue/cmd/cli/root/jetstream/kv/bucket/create"
	clients_jetstream_kv_bucket_delete "github.com/nats-io-custom/nats-jetstream-issue/cmd/cli/root/jetstream/kv/bucket/delete"
	clients_jetstream_kv_bucket_get "github.com/nats-io-custom/nats-jetstream-issue/cmd/cli/root/jetstream/kv/bucket/get"
	clients_jetstream_kv_bucket_put "github.com/nats-io-custom/nats-jetstream-issue/cmd/cli/root/jetstream/kv/bucket/put"
	cobra_utils "github.com/nats-io-custom/nats-jetstream-issue/internal/cobra_utils"

	cobra "github.com/spf13/cobra"
)

const use = "bucket"

// Init command
func Init(parentCmd *cobra.Command) {
	var command = &cobra.Command{
		Use:               use,
		Short:             use,
		PersistentPreRunE: cobra_utils.ParentPersistentPreRunE,
	}

	clients_jetstream_kv_bucket_create.Init(command)
	clients_jetstream_kv_bucket_delete.Init(command)
	clients_jetstream_kv_bucket_put.Init(command)
	clients_jetstream_kv_bucket_get.Init(command)

	parentCmd.AddCommand(command)

}
