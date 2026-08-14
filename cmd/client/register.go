package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/warenik/gophkeeper/internal/client/keeper"
	"github.com/warenik/gophkeeper/internal/client/session"
)

// newRegisterCmd возвращает команду регистрации нового пользователя.
func newRegisterCmd() *cobra.Command {
	var login string

	cmd := &cobra.Command{
		Use:   "register",
		Short: "Зарегистрировать нового пользователя и войти",
		RunE: func(cmd *cobra.Command, _ []string) error {
			password, err := secretFlagOrPrompt(cmd, "password", "Пароль: ")
			if err != nil {
				return err
			}

			address := resolveAddress(cmd, "")
			client, err := keeper.Dial(address, "", resolveCACert(cmd))
			if err != nil {
				return err
			}
			defer client.Close()

			ctx, cancel := commandContext()
			defer cancel()

			token, err := client.Register(ctx, login, password)
			if err != nil {
				return err
			}

			if err := session.Save(session.Session{ServerAddress: address, Login: login, Token: token}); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Регистрация успешна, вы вошли как %q\n", login)
			return nil
		},
	}

	cmd.Flags().StringVar(&login, "login", "", "логин пользователя")
	cmd.Flags().String("password", "", "пароль (при отсутствии запрашивается без эха)")
	requireFlags(cmd, "login")

	return cmd
}
