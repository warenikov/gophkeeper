package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/warenikov/gophkeeper/internal/buildmeta"
)

// newVersionCmd возвращает команду вывода версии и даты сборки.
func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Показать версию и дату сборки",
		RunE: func(cmd *cobra.Command, _ []string) error {
			fmt.Fprintf(cmd.OutOrStdout(), "gophkeeper %s\n", buildmeta.String())
			return nil
		},
	}
}
