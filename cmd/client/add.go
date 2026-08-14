package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/warenikov/gophkeeper/internal/client/cryptobox"
	"github.com/warenikov/gophkeeper/internal/client/payload"
	"github.com/warenikov/gophkeeper/internal/pb"
)

// newAddCmd возвращает родительскую команду добавления записей.
func newAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Добавить новую запись",
	}
	cmd.AddCommand(
		newAddLoginPasswordCmd(), newAddTextCmd(), newAddCardCmd(),
		newAddBinaryCmd(), newAddOTPCmd(),
	)
	return cmd
}

// newAddOTPCmd — добавление TOTP-секрета (для генерации одноразовых кодов).
func newAddOTPCmd() *cobra.Command {
	var name, secret, issuer, account, meta string

	cmd := &cobra.Command{
		Use:   "otp",
		Short: "Добавить TOTP-секрет (одноразовые пароли)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			plaintext, err := payload.EncodeOTP(payload.OTP{
				Secret: secret, Issuer: issuer, Account: account,
			})
			if err != nil {
				return err
			}
			return createEncrypted(cmd, pb.SecretType_SECRET_TYPE_OTP, name, meta, plaintext)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "название записи")
	cmd.Flags().StringVar(&secret, "secret", "", "секрет TOTP в base32")
	cmd.Flags().StringVar(&issuer, "issuer", "", "издатель (например, GitHub)")
	cmd.Flags().StringVar(&account, "account", "", "аккаунт")
	cmd.Flags().StringVar(&meta, "meta", "", "произвольная метаинформация (не шифруется)")
	requireFlags(cmd, "name", "secret")

	return cmd
}

// maxBinarySize — максимальный размер бинарной записи в Спринте 2 (без
// стриминга). Стриминговая загрузка произвольного размера — Спринт 3.
const maxBinarySize = 3 << 20 // 3 MiB

// newAddCardCmd — добавление записи типа «банковская карта».
func newAddCardCmd() *cobra.Command {
	var name, number, holder, expiry, meta string

	cmd := &cobra.Command{
		Use:   "card",
		Short: "Добавить банковскую карту",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cvv, err := secretFlagOrPrompt(cmd, "cvv", "CVV: ")
			if err != nil {
				return err
			}
			plaintext, err := payload.EncodeCard(payload.Card{
				Number: number, Holder: holder, Expiry: expiry, CVV: cvv,
			})
			if err != nil {
				return err
			}
			return createEncrypted(cmd, pb.SecretType_SECRET_TYPE_CARD, name, meta, plaintext)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "название записи")
	cmd.Flags().StringVar(&number, "number", "", "номер карты")
	cmd.Flags().StringVar(&holder, "holder", "", "держатель карты")
	cmd.Flags().StringVar(&expiry, "expiry", "", "срок действия (MM/YY)")
	cmd.Flags().String("cvv", "", "CVV (при отсутствии запрашивается без эха)")
	cmd.Flags().StringVar(&meta, "meta", "", "произвольная метаинформация (не шифруется)")
	requireFlags(cmd, "name", "number", "holder", "expiry")

	return cmd
}

// newAddBinaryCmd — добавление бинарных данных из файла.
func newAddBinaryCmd() *cobra.Command {
	var name, file, meta string

	cmd := &cobra.Command{
		Use:   "binary",
		Short: "Добавить бинарные данные из файла",
		RunE: func(cmd *cobra.Command, _ []string) error {
			data, err := os.ReadFile(file)
			if err != nil {
				return fmt.Errorf("чтение файла: %w", err)
			}
			if len(data) > maxBinarySize {
				return fmt.Errorf("файл слишком большой (%d байт), лимит %d байт", len(data), maxBinarySize)
			}
			return createEncrypted(cmd, pb.SecretType_SECRET_TYPE_BINARY, name, meta, data)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "название записи")
	cmd.Flags().StringVar(&file, "file", "", "путь к файлу")
	cmd.Flags().StringVar(&meta, "meta", "", "произвольная метаинформация (не шифруется)")
	requireFlags(cmd, "name", "file")

	return cmd
}

// newAddLoginPasswordCmd — добавление записи типа «логин/пароль».
func newAddLoginPasswordCmd() *cobra.Command {
	var name, login, meta string

	cmd := &cobra.Command{
		Use:   "login-password",
		Short: "Добавить пару логин/пароль",
		RunE: func(cmd *cobra.Command, _ []string) error {
			password, err := secretFlagOrPrompt(cmd, "password", "Пароль: ")
			if err != nil {
				return err
			}
			plaintext, err := payload.EncodeLoginPassword(payload.LoginPassword{Login: login, Password: password})
			if err != nil {
				return err
			}
			return createEncrypted(cmd, pb.SecretType_SECRET_TYPE_LOGIN_PASSWORD, name, meta, plaintext)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "название записи")
	cmd.Flags().StringVar(&login, "login", "", "логин")
	cmd.Flags().String("password", "", "пароль (при отсутствии запрашивается без эха)")
	cmd.Flags().StringVar(&meta, "meta", "", "произвольная метаинформация (не шифруется)")
	requireFlags(cmd, "name", "login")

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
