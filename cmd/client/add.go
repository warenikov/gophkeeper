package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/warenik/gophkeeper/internal/client/cryptobox"
	"github.com/warenik/gophkeeper/internal/client/payload"
	"github.com/warenik/gophkeeper/internal/pb"
)

// newAddCmd возвращает родительскую команду добавления записей.
func newAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Добавить новую запись",
	}
	cmd.AddCommand(newAddLoginPasswordCmd(), newAddTextCmd())
	return cmd
}

// newAddLoginPasswordCmd — добавление записи типа «логин/пароль».
func newAddLoginPasswordCmd() *cobra.Command {
	var name, login, password, meta string

	cmd := &cobra.Command{
		Use:   "login-password",
		Short: "Добавить пару логин/пароль",
		RunE: func(cmd *cobra.Command, _ []string) error {
			plaintext, err := payload.EncodeLoginPassword(payload.LoginPassword{Login: login, Password: password})
			if err != nil {
				return err
			}
			return createEncrypted(cmd, pb.SecretType_SECRET_TYPE_LOGIN_PASSWORD, name, meta, plaintext)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "название записи")
	cmd.Flags().StringVar(&login, "login", "", "логин")
	cmd.Flags().StringVar(&password, "password", "", "пароль")
	cmd.Flags().StringVar(&meta, "meta", "", "произвольная метаинформация (не шифруется)")
	requireFlags(cmd, "name", "login", "password")

	return cmd
}

// newAddTextCmd — добавление записи типа «произвольный текст».
func newAddTextCmd() *cobra.Command {
	var name, text, meta string

	cmd := &cobra.Command{
		Use:   "text",
		Short: "Добавить произвольный текст",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return createEncrypted(cmd, pb.SecretType_SECRET_TYPE_TEXT, name, meta, payload.EncodeText(text))
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "название записи")
	cmd.Flags().StringVar(&text, "text", "", "текст")
	cmd.Flags().StringVar(&meta, "meta", "", "произвольная метаинформация (не шифруется)")
	requireFlags(cmd, "name", "text")

	return cmd
}

// createEncrypted шифрует нагрузку мастер-ключом пользователя и создаёт запись
// на сервере.
func createEncrypted(cmd *cobra.Command, typ pb.SecretType, name, meta string, plaintext []byte) error {
	client, sess, err := authenticatedClient(cmd)
	if err != nil {
		return err
	}
	defer client.Close()

	masterPassword, err := readMasterPassword(cmd)
	if err != nil {
		return err
	}
	key := cryptobox.DeriveKey(masterPassword, sess.Login)

	encrypted, err := cryptobox.Encrypt(key, plaintext)
	if err != nil {
		return err
	}

	ctx, cancel := commandContext()
	defer cancel()

	secret, err := client.Create(ctx, typ, name, meta, encrypted)
	if err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Запись создана: %s\n", secret.GetId())
	return nil
}
