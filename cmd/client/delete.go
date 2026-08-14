package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

// newDeleteCmd возвращает команду удаления записи по идентификатору.
func newDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Удалить запись по идентификатору",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, _, err := authenticatedClient(cmd)
			if err != nil {
				return err
			}
			defer client.Close()

			ctx, cancel := commandContext()
			defer cancel()

			if err := client.Delete(ctx, args[0]); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Запись удалена: %s\n", args[0])
			return nil
		},
	}
}
