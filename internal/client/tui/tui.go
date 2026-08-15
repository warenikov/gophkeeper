// Пакет tui реализует интерактивный терминальный интерфейс (TUI) GophKeeper на
// основе BubbleTea. Это обёртка над готовой клиентской логикой (keeper,
// cryptobox, payload, otp): ввод мастер-пароля, список записей, детальный
// просмотр с расшифровкой, удаление и обновление. Крипто и модель
// zero-knowledge — те же, что в CLI.
package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/warenikov/gophkeeper/internal/client/cryptobox"
	"github.com/warenikov/gophkeeper/internal/client/keeper"
	"github.com/warenikov/gophkeeper/internal/client/otp"
	"github.com/warenikov/gophkeeper/internal/client/payload"
	"github.com/warenikov/gophkeeper/internal/pb"
)

// requestTimeout — таймаут обращения к серверу из TUI.
const requestTimeout = 30 * time.Second

// state — экран, отображаемый в данный момент.
type state int

const (
	statePassword state = iota // ввод мастер-пароля
	stateLoading               // ожидание ответа сервера
	stateList                  // список записей
	stateDetail                // детальный просмотр записи
	stateMessage               // сообщение/ошибка
)

// Стили оформления.
var (
	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("42"))
	helpStyle   = lipgloss.NewStyle().Faint(true)
	cursorStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	errorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
)

// Сообщения BubbleTea, возвращаемые командами.
type (
	secretsMsg struct{ secrets []*pb.Secret }
	deletedMsg struct{}
	errMsg     struct{ err error }
)

// model — состояние TUI (архитектура BubbleTea: Model-Update-View).
type model struct {
	state   state
	login   string
	client  *keeper.Client
	key     []byte
	input   textinput.Model
	secrets []*pb.Secret
	cursor  int
	detail  string
	message string
	width   int
	height  int
}

// Run запускает TUI для авторизованного пользователя login поверх клиента
// client. Возвращает ошибку при сбое интерфейса.
func Run(login string, client *keeper.Client) error {
	ti := textinput.New()
	ti.Placeholder = "мастер-пароль"
	ti.EchoMode = textinput.EchoPassword
	ti.EchoCharacter = '•'
	ti.Focus()
	ti.CharLimit = 256

	m := model{
		state:  statePassword,
		login:  login,
		client: client,
		input:  ti,
	}

	_, err := tea.NewProgram(m, tea.WithAltScreen()).Run()
	return err
}

// Init — начальная команда (мигание курсора ввода).
func (m model) Init() tea.Cmd {
	return textinput.Blink
}

// Update обрабатывает сообщения и обновляет состояние.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case errMsg:
		m.state = stateMessage
		m.message = errorStyle.Render("Ошибка: " + msg.err.Error())
		return m, nil

	case secretsMsg:
		m.secrets = msg.secrets
		m.cursor = 0
		m.state = stateList
		return m, nil

	case deletedMsg:
		m.state = stateLoading
		return m, m.loadSecretsCmd()

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		return m.handleKey(msg)
	}

	// Прочие сообщения на экране ввода пароля отдаём текстовому полю.
	if m.state == statePassword {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
	return m, nil
}

// handleKey обрабатывает нажатия клавиш в зависимости от экрана.
func (m model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.state {
	case statePassword:
		if msg.String() == "enter" {
			if m.input.Value() == "" {
				return m, nil
			}
			m.key = cryptobox.DeriveKey(m.input.Value(), m.login)
			m.state = stateLoading
			return m, m.loadSecretsCmd()
		}
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd

	case stateList:
		return m.handleListKey(msg)

	case stateDetail:
		switch msg.String() {
		case "esc", "q", "enter", "backspace":
			m.state = stateList
		}
		return m, nil

	case stateMessage:
		m.state = stateList
		return m, nil

	default: // stateLoading
		return m, nil
	}
}

// handleListKey обрабатывает навигацию и действия в списке записей.
func (m model) handleListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc":
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.secrets)-1 {
			m.cursor++
		}
	case "r":
		m.state = stateLoading
		return m, m.loadSecretsCmd()
	case "enter":
		if len(m.secrets) == 0 {
			return m, nil
		}
		s := m.secrets[m.cursor]
		plaintext, err := cryptobox.Decrypt(m.key, s.GetEncryptedPayload())
		if err != nil {
			m.state = stateMessage
			m.message = errorStyle.Render("Не удалось расшифровать (неверный мастер-пароль?): " + err.Error())
			return m, nil
		}
		m.detail = formatSecret(s, plaintext)
		m.state = stateDetail
	case "d":
		if len(m.secrets) == 0 {
			return m, nil
		}
		m.state = stateLoading
		return m, m.deleteCmd(m.secrets[m.cursor].GetId())
	}
	return m, nil
}

// loadSecretsCmd загружает список записей с сервера.
func (m model) loadSecretsCmd() tea.Cmd {
	client := m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
		defer cancel()
		secrets, err := client.List(ctx)
		if err != nil {
			return errMsg{err}
		}
		return secretsMsg{secrets}
	}
}

// deleteCmd удаляет запись по идентификатору.
func (m model) deleteCmd(id string) tea.Cmd {
	client := m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
		defer cancel()
		if err := client.Delete(ctx, id); err != nil {
			return errMsg{err}
		}
		return deletedMsg{}
	}
}

// View рендерит текущий экран.
func (m model) View() string {
	switch m.state {
	case statePassword:
		return fmt.Sprintf("%s\n\nВойти как %s\n\n%s\n\n%s",
			titleStyle.Render("GophKeeper"),
			m.login,
			m.input.View(),
			helpStyle.Render("enter — войти · ctrl+c — выход"),
		)

	case stateLoading:
		return "\n  Загрузка…\n"

	case stateList:
		return m.viewList()

	case stateDetail:
		return fmt.Sprintf("%s\n\n%s\n\n%s",
			titleStyle.Render("Запись"),
			m.detail,
			helpStyle.Render("esc — назад"),
		)

	case stateMessage:
		return fmt.Sprintf("\n%s\n\n%s", m.message, helpStyle.Render("любая клавиша — назад"))

	default:
		return ""
	}
}

// viewList рендерит список записей с курсором.
func (m model) viewList() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n\n", titleStyle.Render("GophKeeper — "+m.login))

	if len(m.secrets) == 0 {
		b.WriteString("  Записей нет.\n")
	}
	for i, s := range m.secrets {
		cursor := "  "
		line := fmt.Sprintf("%-24s %-14s %s", s.GetName(), typeLabel(s.GetType()), s.GetMetadata())
		if i == m.cursor {
			cursor = cursorStyle.Render("▸ ")
			line = cursorStyle.Render(line)
		}
		fmt.Fprintf(&b, "%s%s\n", cursor, line)
	}

	fmt.Fprintf(&b, "\n%s",
		helpStyle.Render("↑/↓ — выбор · enter — открыть · d — удалить · r — обновить · q — выход"))
	return b.String()
}

// formatSecret формирует текст детального просмотра расшифрованной записи.
func formatSecret(s *pb.Secret, plaintext []byte) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Имя:  %s\nТип:  %s\n", s.GetName(), typeLabel(s.GetType()))
	if meta := s.GetMetadata(); meta != "" {
		fmt.Fprintf(&b, "Мета: %s\n", meta)
	}

	switch s.GetType() {
	case pb.SecretType_SECRET_TYPE_LOGIN_PASSWORD:
		lp, err := payload.DecodeLoginPassword(plaintext)
		if err != nil {
			return b.String() + errorStyle.Render(err.Error())
		}
		fmt.Fprintf(&b, "\nЛогин:  %s\nПароль: %s\n", lp.Login, lp.Password)
	case pb.SecretType_SECRET_TYPE_TEXT:
		fmt.Fprintf(&b, "\nТекст:\n%s\n", payload.DecodeText(plaintext))
	case pb.SecretType_SECRET_TYPE_CARD:
		c, err := payload.DecodeCard(plaintext)
		if err != nil {
			return b.String() + errorStyle.Render(err.Error())
		}
		fmt.Fprintf(&b, "\nНомер:     %s\nДержатель: %s\nСрок:      %s\nCVV:       %s\n",
			c.Number, c.Holder, c.Expiry, c.CVV)
	case pb.SecretType_SECRET_TYPE_OTP:
		o, err := payload.DecodeOTP(plaintext)
		if err != nil {
			return b.String() + errorStyle.Render(err.Error())
		}
		code, remaining, err := otp.Code(o.Secret, time.Now())
		if err != nil {
			return b.String() + errorStyle.Render(err.Error())
		}
		fmt.Fprintf(&b, "\nКод: %s (действует ещё %d с)\n", code, remaining)
	case pb.SecretType_SECRET_TYPE_BINARY:
		fmt.Fprintf(&b, "\nБинарные данные: %d байт (сохранение в файл — через CLI: get --out)\n", len(plaintext))
	default:
		fmt.Fprintf(&b, "\nДанные: %x\n", plaintext)
	}
	return b.String()
}

// typeLabel возвращает человекочитаемое имя типа записи.
func typeLabel(t pb.SecretType) string {
	switch t {
	case pb.SecretType_SECRET_TYPE_LOGIN_PASSWORD:
		return "логин/пароль"
	case pb.SecretType_SECRET_TYPE_TEXT:
		return "текст"
	case pb.SecretType_SECRET_TYPE_CARD:
		return "карта"
	case pb.SecretType_SECRET_TYPE_BINARY:
		return "бинарные"
	case pb.SecretType_SECRET_TYPE_OTP:
		return "OTP"
	default:
		return "неизвестно"
	}
}
