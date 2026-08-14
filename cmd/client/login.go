package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/warenik/gophkeeper/internal/client/keeper"
	"github.com/warenik/gophkeeper/internal/client/session"
)

// newLoginCmd возвращает команду аутентификации существующего пользователя.
func newLoginCmd() *cobra.Command {
	var login, password string

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Войти под существующим пользователем",
		RunE: func(cmd *cobra.Command, _ []string) error {
			address := resolveAddress(cmd, "")
			client, err := keeper.Dial(address, "")
			if err != nil {
				return err
			}
			defer client.Close()

			ctx, cancel := commandContext()
			defer cancel()

			token, err := client.Login(ctx, login, password)
			if err != nil {
				return err
			}

			if err := session.Save(session.Session{ServerAddress: address, Login: login, Token: token}); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Вход выполнен как %q\n", login)
			return nil
		},
	}

	cmd.Flags().StringVar(&login, "login", "", "логин пользователя")
	cmd.Flags().StringVar(&password, "password", "", "пароль пользователя")
	requireFlags(cmd, "login", "password")

	return cmd
}

// newLogoutCmd возвращает команду выхода (удаление локальной сессии).
func newLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Выйти и удалить локальную сессию",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := session.Clear(); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Сессия удалена")
			return nil
		},
	}
}
