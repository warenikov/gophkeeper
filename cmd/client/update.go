package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/warenikov/gophkeeper/internal/client/cryptobox"
	"github.com/warenikov/gophkeeper/internal/client/payload"
)

// newUpdateCmd возвращает родительскую команду обновления записей.
func newUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Обновить существующую запись",
	}
	cmd.AddCommand(
		newUpdateLoginPasswordCmd(),
		newUpdateTextCmd(),
		newUpdateCardCmd(),
		newUpdateBinaryCmd(),
	)
	return cmd
}

func newUpdateLoginPasswordCmd() *cobra.Command {
	var id, name, login, password, meta string
	cmd := &cobra.Command{
		Use:   "login-password",
		Short: "Обновить пару логин/пароль",
		RunE: func(cmd *cobra.Command, _ []string) error {
			plaintext, err := payload.EncodeLoginPassword(payload.LoginPassword{Login: login, Password: password})
			if err != nil {
				return err
			}
			return updateEncrypted(cmd, id, name, meta, plaintext)
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "идентификатор записи")
	cmd.Flags().StringVar(&name, "name", "", "название записи")
	cmd.Flags().StringVar(&login, "login", "", "логин")
	cmd.Flags().StringVar(&password, "password", "", "пароль")
	cmd.Flags().StringVar(&meta, "meta", "", "метаинформация")
	requireFlags(cmd, "id", "name", "login", "password")
	return cmd
}

func newUpdateTextCmd() *cobra.Command {
	var id, name, text, meta string
	cmd := &cobra.Command{
		Use:   "text",
		Short: "Обновить произвольный текст",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return updateEncrypted(cmd, id, name, meta, payload.EncodeText(text))
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "идентификатор записи")
	cmd.Flags().StringVar(&name, "name", "", "название записи")
	cmd.Flags().StringVar(&text, "text", "", "текст")
	cmd.Flags().StringVar(&meta, "meta", "", "метаинформация")
	requireFlags(cmd, "id", "name", "text")
	return cmd
}

func newUpdateCardCmd() *cobra.Command {
	var id, name, number, holder, expiry, cvv, meta string
	cmd := &cobra.Command{
		Use:   "card",
		Short: "Обновить банковскую карту",
		RunE: func(cmd *cobra.Command, _ []string) error {
			plaintext, err := payload.EncodeCard(payload.Card{
				Number: number, Holder: holder, Expiry: expiry, CVV: cvv,
			})
			if err != nil {
				return err
			}
			return updateEncrypted(cmd, id, name, meta, plaintext)
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "идентификатор записи")
	cmd.Flags().StringVar(&name, "name", "", "название записи")
	cmd.Flags().StringVar(&number, "number", "", "номер карты")
	cmd.Flags().StringVar(&holder, "holder", "", "держатель карты")
	cmd.Flags().StringVar(&expiry, "expiry", "", "срок действия (MM/YY)")
	cmd.Flags().StringVar(&cvv, "cvv", "", "CVV")
	cmd.Flags().StringVar(&meta, "meta", "", "метаинформация")
	requireFlags(cmd, "id", "name", "number", "holder", "expiry", "cvv")
	return cmd
}

func newUpdateBinaryCmd() *cobra.Command {
	var id, name, file, meta string
	cmd := &cobra.Command{
		Use:   "binary",
		Short: "Обновить бинарные данные из файла",
		RunE: func(cmd *cobra.Command, _ []string) error {
			data, err := os.ReadFile(file)
			if err != nil {
				return fmt.Errorf("чтение файла: %w", err)
			}
			if len(data) > maxBinarySize {
				return fmt.Errorf("файл слишком большой (%d байт), лимит %d байт", len(data), maxBinarySize)
			}
			return updateEncrypted(cmd, id, name, meta, data)
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "идентификатор записи")
	cmd.Flags().StringVar(&name, "name", "", "название записи")
	cmd.Flags().StringVar(&file, "file", "", "путь к файлу")
	cmd.Flags().StringVar(&meta, "meta", "", "метаинформация")
	requireFlags(cmd, "id", "name", "file")
	return cmd
}

// updateEncrypted шифрует новую нагрузку и обновляет запись, автоматически
// подставляя текущую версию (оптимистичная блокировка).
func updateEncrypted(cmd *cobra.Command, id, name, meta string, plaintext []byte) error {
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

	// Текущая версия нужна для контроля конфликтов.
	current, err := client.Get(ctx, id)
	if err != nil {
		return err
	}

	updated, err := client.Update(ctx, id, name, meta, encrypted, current.GetVersion())
	if err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Запись обновлена: %s (версия %d)\n", updated.GetId(), updated.GetVersion())
	return nil
}
