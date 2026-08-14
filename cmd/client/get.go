package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/warenik/gophkeeper/internal/client/cryptobox"
	"github.com/warenik/gophkeeper/internal/client/payload"
	"github.com/warenik/gophkeeper/internal/pb"
)

// newGetCmd возвращает команду получения и расшифровки одной записи.
func newGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Получить и расшифровать запись по идентификатору",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, sess, err := authenticatedClient(cmd)
			if err != nil {
				return err
			}
			defer client.Close()

			masterPassword, err := readMasterPassword(cmd)
			if err != nil {
				return err
			}

			ctx, cancel := commandContext()
			defer cancel()

			secret, err := client.Get(ctx, args[0])
			if err != nil {
				return err
			}

			key := cryptobox.DeriveKey(masterPassword, sess.Login)
			plaintext, err := cryptobox.Decrypt(key, secret.GetEncryptedPayload())
			if err != nil {
				return err
			}

			return printSecret(cmd, secret, plaintext)
		},
	}
}

// printSecret выводит запись: метаданные в открытом виде и расшифрованную
// нагрузку в зависимости от типа.
func printSecret(cmd *cobra.Command, secret *pb.Secret, plaintext []byte) error {
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "Имя:    %s\n", secret.GetName())
	fmt.Fprintf(out, "Тип:    %s\n", typeName(secret.GetType()))
	if meta := secret.GetMetadata(); meta != "" {
		fmt.Fprintf(out, "Мета:   %s\n", meta)
	}

	switch secret.GetType() {
	case pb.SecretType_SECRET_TYPE_LOGIN_PASSWORD:
		lp, err := payload.DecodeLoginPassword(plaintext)
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "Логин:  %s\n", lp.Login)
		fmt.Fprintf(out, "Пароль: %s\n", lp.Password)
	case pb.SecretType_SECRET_TYPE_TEXT:
		fmt.Fprintf(out, "Текст:  %s\n", payload.DecodeText(plaintext))
	default:
		fmt.Fprintf(out, "Данные: %x\n", plaintext)
	}
	return nil
}
