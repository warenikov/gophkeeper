package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/warenik/gophkeeper/internal/client/cache"
	"github.com/warenik/gophkeeper/internal/pb"
)

// newSyncCmd возвращает команду синхронизации локального кэша с сервером.
func newSyncCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Синхронизировать локальный кэш с сервером",
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, sess, err := authenticatedClient(cmd)
			if err != nil {
				return err
			}
			defer client.Close()

			local, err := cache.Load(sess.Login)
			if err != nil {
				return err
			}

			ctx, cancel := commandContext()
			defer cancel()

			changes, revision, err := client.Sync(ctx, local.Revision)
			if err != nil {
				return err
			}

			updated, deleted := applyChanges(&local, changes)
			local.Revision = revision
			if err := cache.Save(local); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(),
				"Синхронизировано: обновлено %d, удалено %d, всего в кэше %d (ревизия %d)\n",
				updated, deleted, len(local.Secrets), revision)
			return nil
		},
	}
}

// applyChanges применяет изменения сервера к локальному кэшу: тумбстоны удаляют
// записи, остальные — обновляют/добавляют. Возвращает число обновлённых и
// удалённых записей.
func applyChanges(local *cache.Cache, changes []*pb.Secret) (updated, deleted int) {
	if local.Secrets == nil {
		local.Secrets = map[string]cache.Secret{}
	}
	for _, s := range changes {
		if s.GetDeleted() {
			if _, ok := local.Secrets[s.GetId()]; ok {
				delete(local.Secrets, s.GetId())
				deleted++
			}
			continue
		}
		local.Secrets[s.GetId()] = cache.Secret{
			ID:               s.GetId(),
			Type:             int32(s.GetType()),
			Name:             s.GetName(),
			Metadata:         s.GetMetadata(),
			EncryptedPayload: s.GetEncryptedPayload(),
			Version:          s.GetVersion(),
			UpdatedAt:        s.GetUpdatedAt().AsTime(),
		}
		updated++
	}
	return updated, deleted
}

// cacheToProto восстанавливает protobuf-запись из кэшированной для единообразного
// вывода (например, в офлайн-режиме).
func cacheToProto(s cache.Secret) *pb.Secret {
	return &pb.Secret{
		Id:               s.ID,
		Type:             pb.SecretType(s.Type),
		Name:             s.Name,
		Metadata:         s.Metadata,
		EncryptedPayload: s.EncryptedPayload,
		Version:          s.Version,
		UpdatedAt:        timestamppb.New(s.UpdatedAt),
	}
}
