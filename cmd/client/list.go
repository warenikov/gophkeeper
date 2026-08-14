package main

import (
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/warenikov/gophkeeper/internal/client/cache"
	"github.com/warenikov/gophkeeper/internal/pb"
)

// newListCmd возвращает команду вывода списка записей пользователя. При
// недоступности сервера переходит в офлайн-режим и читает из локального кэша.
func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Показать список записей (без расшифровки)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, sess, err := authenticatedClient(cmd)
			if err != nil {
				return err
			}
			defer client.Close()

			ctx, cancel := commandContext()
			defer cancel()

			secrets, err := client.List(ctx)
			if err != nil {
				if isOffline(err) {
					return listFromCache(cmd, sess.Login)
				}
				return err
			}
			return renderList(cmd, secrets, false)
		},
	}
}

// listFromCache выводит записи из локального кэша (офлайн-режим).
func listFromCache(cmd *cobra.Command, login string) error {
	local, err := cache.Load(login)
	if err != nil {
		return err
	}
	secrets := make([]*pb.Secret, 0, len(local.Secrets))
	for _, s := range local.Sorted() {
		secrets = append(secrets, cacheToProto(s))
	}
	return renderList(cmd, secrets, true)
}

// renderList печатает таблицу записей. offline включает пометку режима.
func renderList(cmd *cobra.Command, secrets []*pb.Secret, offline bool) error {
	out := cmd.OutOrStdout()
	if offline {
		fmt.Fprintln(out, "[офлайн-режим: данные из локального кэша, только чтение]")
	}
	if len(secrets) == 0 {
		fmt.Fprintln(out, "Записей нет")
		return nil
	}

	w := tabwriter.NewWriter(out, 0, 2, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tТИП\tИМЯ\tМЕТА\tВЕРСИЯ\tОБНОВЛЕНО")
	for _, s := range secrets {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\t%s\n",
			s.GetId(), typeName(s.GetType()), s.GetName(), s.GetMetadata(),
			s.GetVersion(), s.GetUpdatedAt().AsTime().Format("2006-01-02 15:04"),
		)
	}
	return w.Flush()
}

// typeName возвращает человекочитаемое имя типа записи.
func typeName(t pb.SecretType) string {
	switch t {
	case pb.SecretType_SECRET_TYPE_LOGIN_PASSWORD:
		return "логин/пароль"
	case pb.SecretType_SECRET_TYPE_TEXT:
		return "текст"
	case pb.SecretType_SECRET_TYPE_CARD:
		return "карта"
	case pb.SecretType_SECRET_TYPE_BINARY:
		return "бинарные"
	case pb.SecretType_SECRET_TYPE_OTP:
		return "OTP"
	default:
		return "неизвестно"
	}
}
