package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/warenik/gophkeeper/internal/client/cryptobox"
	"github.com/warenik/gophkeeper/internal/client/payload"
	"github.com/warenik/gophkeeper/internal/pb"
)

// newGetCmd возвращает команду получения и расшифровки одной записи.
func newGetCmd() *cobra.Command {
	var outFile string

	cmd := &cobra.Command{
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

			return printSecret(cmd, secret, plaintext, outFile)
		},
	}

	cmd.Flags().StringVar(&outFile, "out", "", "файл для сохранения бинарных данных")
	return cmd
}

// printSecret выводит запись: метаданные в открытом виде и расшифрованную
// нагрузку в зависимости от типа. Для бинарных данных при заданном outFile
// содержимое пишется в файл.
func printSecret(cmd *cobra.Command, secret *pb.Secret, plaintext []byte, outFile string) error {
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
	case pb.SecretType_SECRET_TYPE_CARD:
		c, err := payload.DecodeCard(plaintext)
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "Номер:  %s\n", c.Number)
		fmt.Fprintf(out, "Держатель: %s\n", c.Holder)
		fmt.Fprintf(out, "Срок:   %s\n", c.Expiry)
		fmt.Fprintf(out, "CVV:    %s\n", c.CVV)
	case pb.SecretType_SECRET_TYPE_BINARY:
		return writeBinary(cmd, plaintext, outFile)
	default:
		fmt.Fprintf(out, "Данные: %x\n", plaintext)
	}
	return nil
}

// writeBinary сохраняет бинарные данные в файл (--out) или сообщает их размер.
func writeBinary(cmd *cobra.Command, data []byte, outFile string) error {
	if outFile == "" {
		fmt.Fprintf(cmd.OutOrStdout(), "Бинарные данные: %d байт (укажите --out для сохранения в файл)\n", len(data))
		return nil
	}
	if err := os.WriteFile(outFile, data, 0o600); err != nil {
		return fmt.Errorf("запись файла: %w", err)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Сохранено %d байт в %s\n", len(data), outFile)
	return nil
}
