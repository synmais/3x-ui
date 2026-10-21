package tgbot

import "path/filepath"

// InstructionMediaRoot is the administrator-managed directory for client
// instruction screenshots on the server.
const InstructionMediaRoot = "/srv/synvpn/instructions"

type InstructionButton struct {
	Text     string
	Callback string
}

type InstructionPage struct {
	Text    string
	Buttons []InstructionButton
	Guide   *ClientInstruction
}

type ClientInstruction struct {
	ID          string
	Title       string
	Description string
}

func InstructionPageForCallback(callback string) (InstructionPage, bool) {
	switch callback {
	case "client_instructions":
		return InstructionPage{
			Text: "📚 <b>Инструкции</b>\nВыберите тему:",
			Buttons: []InstructionButton{
				{Text: "📦 Подписка", Callback: "client_instruction_subscription"},
				{Text: "📲 Подключение", Callback: "client_instruction_connection"},
				{Text: "🌐 Выбор сервера", Callback: "client_instruction_server"},
				{Text: "🛟 Обращение в тех. поддержку", Callback: "client_instruction_support"},
			},
		}, true
	case "client_instruction_subscription":
		return InstructionPage{
			Text: "📦 <b>Подписка</b>\nВыберите тему:",
			Buttons: []InstructionButton{
				{Text: "💳 Оплата", Callback: "client_instruction_payment"},
				{Text: "🔄 Продление", Callback: "client_instruction_renewal"},
				{Text: "⬆️ Улучшение тарифа", Callback: "client_instruction_upgrade"},
			},
		}, true
	case "client_instruction_connection":
		return InstructionPage{
			Text: "📲 <b>Подключение</b>\nВыберите приложение:",
			Buttons: []InstructionButton{
				{Text: "Happ", Callback: "client_instruction_happ"},
				{Text: "INCY", Callback: "client_instruction_incy"},
				{Text: "Clash Mi", Callback: "client_instruction_clash_mi"},
			},
		}, true
	}

	guides := map[string]ClientInstruction{
		"client_instruction_payment": {
			ID: "payment", Title: "Оплата",
			Description: "Выберите тариф и срок подписки, затем оплатите заказ удобным способом.",
		},
		"client_instruction_renewal": {
			ID: "subscription/renewal", Title: "Продление подписки",
			Description: "В меню «Подписка» нажмите «Продлить подписку», выберите тариф и срок, затем оплатите заказ.",
		},
		"client_instruction_upgrade": {
			ID: "subscription/upgrade", Title: "Улучшение тарифа",
			Description: "В меню «Подписка» выберите «Улучшить тариф», затем выберите подходящий тариф и срок.",
		},
		"client_instruction_happ": {
			ID: "connection/happ", Title: "Подключение через Happ",
			Description: "Установите Happ и добавьте подписку по ссылке из меню «Подписка».",
		},
		"client_instruction_incy": {
			ID: "connection/incy", Title: "Подключение через INCY",
			Description: "Установите INCY и добавьте подписку по ссылке из меню «Подписка».",
		},
		"client_instruction_clash_mi": {
			ID: "connection/clash-mi", Title: "Подключение через Clash Mi",
			Description: "Установите Clash Mi и добавьте подписку по ссылке из меню «Подписка».",
		},
		"client_instruction_server": {
			ID: "server-selection", Title: "Выбор сервера",
			Description: "Откройте список серверов в приложении и выберите подходящий.",
		},
		"client_instruction_support": {
			ID: "support", Title: "Обращение в тех. поддержку",
			Description: "Опишите проблему и приложите скриншоты. Отправить обращение можно кнопкой «Тех. поддержка» в главном меню.",
		},
	}
	guide, ok := guides[callback]
	if !ok {
		return InstructionPage{}, false
	}
	return InstructionPage{Guide: &guide}, true
}

func (i ClientInstruction) MediaDirectory() string {
	return filepath.Join(InstructionMediaRoot, filepath.FromSlash(i.ID))
}
