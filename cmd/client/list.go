package main

import (
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/warenik/gophkeeper/internal/pb"
)

// newListCmd возвращает команду вывода списка записей пользователя.
func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Показать список записей (без расшифровки)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, _, err := authenticatedClient(cmd)
			if err != nil {
				return err
			}
			defer client.Close()

			ctx, cancel := commandContext()
			defer cancel()

			secrets, err := client.List(ctx)
			if err != nil {
				return err
			}

			if len(secrets) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "Записей нет")
				return nil
			}

			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 2, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tТИП\tИМЯ\tМЕТА\tВЕРСИЯ\tОБНОВЛЕНО")
			for _, s := range secrets {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\t%s\n",
					s.GetId(), typeName(s.GetType()), s.GetName(), s.GetMetadata(),
					s.GetVersion(), s.GetUpdatedAt().AsTime().Format("2006-01-02 15:04"),
				)
			}
			return w.Flush()
		},
	}
}

// typeName возвращает человекочитаемое имя типа записи.
func typeName(t pb.SecretType) string {
	switch t {
	case pb.SecretType_SECRET_TYPE_LOGIN_PASSWORD:
		return "логин/пароль"
	case pb.SecretType_SECRET_TYPE_TEXT:
		return "текст"
	default:
		return "неизвестно"
	}
}
