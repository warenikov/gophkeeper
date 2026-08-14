package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/warenik/gophkeeper/internal/client/keeper"
	"github.com/warenik/gophkeeper/internal/client/session"
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

// readMasterPassword возвращает мастер-пароль из флага/переменной окружения, а
// при их отсутствии запрашивает его интерактивно без эха.
func readMasterPassword(cmd *cobra.Command) (string, error) {
	if mp := flagOrEnv(cmd, "master-password", envMasterPassword); mp != "" {
		return mp, nil
	}

	fmt.Fprint(cmd.ErrOrStderr(), "Мастер-пароль: ")
	data, err := term.ReadPassword(int(os.Stdin.Fd())) //nolint:gosec // файловый дескриптор всегда помещается в int

	fmt.Fprintln(cmd.ErrOrStderr())
	if err != nil {
		return "", fmt.Errorf("чтение мастер-пароля: %w", err)
	}

	mp := strings.TrimSpace(string(data))
	if mp == "" {
		return "", errors.New("мастер-пароль не может быть пустым")
	}
	return mp, nil
}

// authenticatedClient загружает сессию и создаёт клиент с токеном доступа.
func authenticatedClient(cmd *cobra.Command) (*keeper.Client, session.Session, error) {
	sess, err := session.Load()
	if err != nil {
		return nil, session.Session{}, err
	}

	client, err := keeper.Dial(resolveAddress(cmd, sess.ServerAddress), sess.Token)
	if err != nil {
		return nil, session.Session{}, err
	}
	return client, sess, nil
}

// commandContext возвращает контекст с таймаутом обращения к серверу.
func commandContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), requestTimeout)
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
