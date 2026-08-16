package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/spf13/cobra"
	"golang.org/x/term"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/warenikov/gophkeeper/internal/client/keeper"
	"github.com/warenikov/gophkeeper/internal/client/session"
)

// requestTimeout — таймаут одного обращения к серверу.
const requestTimeout = 30 * time.Second

// flagOrEnv возвращает значение флага, а при его отсутствии — значение
// переменной окружения. Приоритет: флаг > окружение.
func flagOrEnv(cmd *cobra.Command, flagName, envName string) string {
	if v, err := cmd.Flags().GetString(flagName); err == nil && v != "" {
		return v
	}
	return os.Getenv(envName)
}

// resolveAddress выбирает адрес сервера: флаг/переменная окружения → сохранённая
// сессия → значение по умолчанию.
func resolveAddress(cmd *cobra.Command, sessionAddr string) string {
	if a := flagOrEnv(cmd, "address", envAddress); a != "" {
		return a
	}
	if sessionAddr != "" {
		return sessionAddr
	}
	return defaultAddress
}

// resolveCACert возвращает путь к корневому сертификату TLS из флага/окружения
// (пустая строка означает незащищённое соединение).
func resolveCACert(cmd *cobra.Command) string {
	return flagOrEnv(cmd, "tls-ca", envTLSCA)
}

// readMasterPassword возвращает мастер-пароль из флага/переменной окружения, а
// при их отсутствии запрашивает его интерактивно без эха. Значение всегда
// триммится (и для флага/env, и для интерактива), иначе один и тот же пароль
// дал бы разные ключи шифрования.
func readMasterPassword(cmd *cobra.Command) (string, error) {
	if mp := flagOrEnv(cmd, "master-password", envMasterPassword); mp != "" {
		return strings.TrimSpace(mp), nil
	}

	mp, err := promptSecret(cmd, "Мастер-пароль: ")
	if err != nil {
		return "", err
	}
	if mp == "" {
		return "", errors.New("мастер-пароль не может быть пустым")
	}
	return mp, nil
}

// promptSecret читает секрет из терминала без эха (пароли и т. п.).
func promptSecret(cmd *cobra.Command, label string) (string, error) {
	fmt.Fprint(cmd.ErrOrStderr(), label)
	data, err := term.ReadPassword(int(os.Stdin.Fd())) //nolint:gosec // файловый дескриптор всегда помещается в int
	fmt.Fprintln(cmd.ErrOrStderr())
	if err != nil {
		return "", fmt.Errorf("чтение ввода: %w", err)
	}
	return strings.TrimSpace(string(data)), nil
}

// secretFlagOrPrompt возвращает значение флага, а при его отсутствии запрашивает
// секрет интерактивно без эха — чтобы не передавать его в аргументах командной
// строки (утечка в history/ps).
func secretFlagOrPrompt(cmd *cobra.Command, flagName, label string) (string, error) {
	if v, err := cmd.Flags().GetString(flagName); err == nil && v != "" {
		return v, nil
	}
	value, err := promptSecret(cmd, label)
	if err != nil {
		return "", err
	}
	if value == "" {
		return "", fmt.Errorf("%s не может быть пустым", flagName)
	}
	return value, nil
}

// authenticatedClient загружает сессию и создаёт клиент с токеном доступа.
func authenticatedClient(cmd *cobra.Command) (*keeper.Client, session.Session, error) {
	sess, err := session.Load()
	if err != nil {
		return nil, session.Session{}, err
	}

	client, err := keeper.Dial(resolveAddress(cmd, sess.ServerAddress), sess.Token, resolveCACert(cmd))
	if err != nil {
		return nil, session.Session{}, err
	}
	return client, sess, nil
}

// commandContext возвращает контекст с таймаутом обращения к серверу.
func commandContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), requestTimeout)
}

// ensureUTF8 проверяет, что значение — корректный UTF-8. Поля, уходящие на
// сервер как proto3-строки, обязаны быть в UTF-8; иначе gRPC падает с
// невнятной ошибкой маршалинга. Проверяем заранее и даём понятное сообщение.
func ensureUTF8(field, value string) error {
	if !utf8.ValidString(value) {
		return fmt.Errorf("поле %q содержит недопустимые символы: нужен корректный UTF-8 "+
			"(возможно, символ введён нестандартной раскладкой или вставлен из другого источника)", field)
	}
	return nil
}

// isOffline сообщает, вызвана ли ошибка недоступностью сервера (нет сети) —
// в этом случае команды переходят в офлайн-режим (read-only из кэша).
func isOffline(err error) bool {
	code := status.Code(err)
	return code == codes.Unavailable || code == codes.DeadlineExceeded
}

// requireFlags помечает перечисленные флаги обязательными. Паникует при ошибке
// (ссылка на несуществующий флаг — ошибка программиста на этапе сборки команды).
func requireFlags(cmd *cobra.Command, names ...string) {
	for _, name := range names {
		if err := cmd.MarkFlagRequired(name); err != nil {
			panic(fmt.Sprintf("mark flag %q required: %v", name, err))
		}
	}
}
