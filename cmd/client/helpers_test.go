package main

import (
	"bytes"
	"strings"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/warenikov/gophkeeper/internal/client/cache"
	"github.com/warenikov/gophkeeper/internal/pb"
)

func TestTypeName(t *testing.T) {
	cases := map[pb.SecretType]string{
		pb.SecretType_SECRET_TYPE_LOGIN_PASSWORD: "логин/пароль",
		pb.SecretType_SECRET_TYPE_TEXT:           "текст",
		pb.SecretType_SECRET_TYPE_CARD:           "карта",
		pb.SecretType_SECRET_TYPE_BINARY:         "бинарные",
		pb.SecretType_SECRET_TYPE_OTP:            "OTP",
		pb.SecretType_SECRET_TYPE_UNSPECIFIED:    "неизвестно",
	}
	for typ, want := range cases {
		if got := typeName(typ); got != want {
			t.Errorf("typeName(%v) = %q, ожидалось %q", typ, got, want)
		}
	}
}

func TestIsOffline(t *testing.T) {
	if !isOffline(status.Error(codes.Unavailable, "нет сети")) {
		t.Error("Unavailable должен считаться офлайном")
	}
	if !isOffline(status.Error(codes.DeadlineExceeded, "таймаут")) {
		t.Error("DeadlineExceeded должен считаться офлайном")
	}
	if isOffline(status.Error(codes.NotFound, "нет записи")) {
		t.Error("NotFound не должен считаться офлайном")
	}
	if isOffline(nil) {
		t.Error("nil не должен считаться офлайном")
	}
}

func TestResolveAddress(t *testing.T) {
	cmd := newRootCmd()

	// Без флага/env/сессии — значение по умолчанию.
	if got := resolveAddress(cmd, ""); got != defaultAddress {
		t.Errorf("resolveAddress = %q, ожидалось %q", got, defaultAddress)
	}
	// Адрес из сессии, когда флаг/env пусты.
	if got := resolveAddress(cmd, "session-host:1"); got != "session-host:1" {
		t.Errorf("resolveAddress(session) = %q", got)
	}
	// Переменная окружения имеет приоритет над сессией.
	t.Setenv(envAddress, "env-host:2")
	if got := resolveAddress(cmd, "session-host:1"); got != "env-host:2" {
		t.Errorf("resolveAddress(env) = %q, ожидалось env-host:2", got)
	}
}

func TestApplyChanges(t *testing.T) {
	local := &cache.Cache{Secrets: map[string]cache.Secret{
		"s1": {ID: "s1", Name: "old"},
	}}
	changes := []*pb.Secret{
		{Id: "s2", Name: "new", Type: pb.SecretType_SECRET_TYPE_TEXT}, // добавление
		{Id: "s1", Deleted: true},                                     // тумбстон
	}

	updated, deleted := applyChanges(local, changes)
	if updated != 1 || deleted != 1 {
		t.Errorf("updated=%d deleted=%d, ожидалось 1 и 1", updated, deleted)
	}
	if _, ok := local.Secrets["s1"]; ok {
		t.Error("удалённая запись s1 осталась в кэше")
	}
	if local.Secrets["s2"].Name != "new" {
		t.Error("новая запись s2 не добавлена")
	}
}

func TestVersionCommand(t *testing.T) {
	cmd := newRootCmd()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"version"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("version: %v", err)
	}
	if !strings.Contains(buf.String(), "gophkeeper") {
		t.Errorf("вывод version не содержит имя: %q", buf.String())
	}
}
