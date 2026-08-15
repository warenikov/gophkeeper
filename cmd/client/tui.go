package main

import (
	"github.com/spf13/cobra"

	"github.com/warenikov/gophkeeper/internal/client/tui"
)

// newTUICmd возвращает команду запуска интерактивного терминального интерфейса.
func newTUICmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tui",
		Short: "Запустить интерактивный интерфейс (TUI)",
		Long: "Интерактивный терминальный интерфейс: ввод мастер-пароля, список\n" +
			"записей, просмотр с расшифровкой, удаление. Требуется вход (login).",
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, sess, err := authenticatedClient(cmd)
			if err != nil {
				return err
			}
			defer client.Close()

			return tui.Run(sess.Login, client)
		},
	}
}
