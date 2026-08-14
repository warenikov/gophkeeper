package main

import (
	"github.com/spf13/cobra"

	"github.com/warenik/gophkeeper/internal/buildmeta"
)

// defaultAddress — адрес сервера по умолчанию.
const defaultAddress = "localhost:8081"

// Имена переменных окружения.
const (
	envAddress        = "GOPHKEEPER_ADDRESS"
	envMasterPassword = "GOPHKEEPER_MASTER_PASSWORD" //nolint:gosec // это имя переменной окружения, а не сам секрет
	envTLSCA          = "GOPHKEEPER_TLS_CA"
)

// newRootCmd собирает корневую команду и всё дерево подкоманд.
func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "gophkeeper",
		Short: "GophKeeper — менеджер приватных данных",
		Long: "GophKeeper — клиент для безопасного хранения логинов/паролей и текста.\n" +
			"Данные шифруются на клиенте; сервер видит только шифртекст.",
		Version:       buildmeta.Version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	// Глобальные флаги, наследуемые подкомандами. Пустое значение по умолчанию
	// означает «взять из окружения или сессии».
	root.PersistentFlags().String("address", "", "адрес сервера (или env "+envAddress+", по умолчанию "+defaultAddress+")")
	root.PersistentFlags().String("master-password", "", "мастер-пароль для шифрования (или env "+envMasterPassword+")")
	root.PersistentFlags().String("tls-ca", "", "путь к корневому сертификату TLS (или env "+envTLSCA+"); без него — незащищённо")

	root.AddCommand(
		newVersionCmd(),
		newRegisterCmd(),
		newLoginCmd(),
		newLogoutCmd(),
		newAddCmd(),
		newListCmd(),
		newGetCmd(),
		newUpdateCmd(),
		newDeleteCmd(),
		newSyncCmd(),
	)

	return root
}

// Execute запускает корневую команду.
func Execute() error {
	return newRootCmd().Execute()
}
