package tgbot

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mhsanaei/3x-ui/v3/internal/logger"
	synvpntgbot "github.com/mhsanaei/3x-ui/v3/internal/synvpn/tgbot"
	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
)

var errNoSynVPNInstructionScreenshots = errors.New("no instruction screenshots found")

type instructionUploadFile struct {
	*os.File
	filename string
}

func (f instructionUploadFile) Name() string { return f.filename }

func synVPNInstructionKeyboard(buttons []synvpntgbot.InstructionButton) *telego.InlineKeyboardMarkup {
	rows := make([][]telego.InlineKeyboardButton, 0, len(buttons))
	for _, button := range buttons {
		rows = append(rows, tu.InlineKeyboardRow(
			tu.InlineKeyboardButton(button.Text).WithCallbackData(button.Callback),
		))
	}
	if len(rows) == 0 {
		return nil
	}
	return &telego.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func sendSynVPNInstructionMedia(chatID int64, instructionID, caption string, replyMarkup *telego.InlineKeyboardMarkup) ([]int, int, error) {
	var instruction *synvpntgbot.ClientInstruction
	for _, callback := range []string{
		"client_instruction_payment", "client_instruction_renewal", "client_instruction_upgrade",
		"client_instruction_happ", "client_instruction_incy",
		"client_instruction_clash_mi", "client_instruction_server", "client_instruction_support",
	} {
		page, ok := synvpntgbot.InstructionPageForCallback(callback)
		if ok && page.Guide != nil && page.Guide.ID == instructionID {
			instruction = page.Guide
			break
		}
	}
	if instruction == nil {
		return nil, 0, fmt.Errorf("unknown instruction %q", instructionID)
	}

	entries, err := os.ReadDir(instruction.MediaDirectory())
	if err != nil {
		return nil, 0, err
	}
	paths := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.Type()&os.ModeSymlink != 0 || entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".webp" {
			continue
		}
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() {
			continue
		}
		paths = append(paths, filepath.Join(instruction.MediaDirectory(), entry.Name()))
	}
	if len(paths) == 0 {
		return nil, 0, fmt.Errorf("%w in %s", errNoSynVPNInstructionScreenshots, instruction.MediaDirectory())
	}

	messageIDs := make([]int, 0, len(paths))
	sendPhoto := func(path, photoCaption string, markup *telego.InlineKeyboardMarkup) error {
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		message, sendErr := bot.SendPhoto(context.Background(), &telego.SendPhotoParams{
			ChatID: tu.ID(chatID), Photo: telego.InputFile{File: instructionUploadFile{File: file, filename: filepath.Base(path)}},
			Caption: photoCaption, ReplyMarkup: markup,
		})
		closeErr := file.Close()
		if sendErr != nil {
			return sendErr
		}
		if closeErr != nil {
			logger.Warning("Failed to close client instruction screenshot:", closeErr)
		}
		messageIDs = append(messageIDs, message.MessageID)
		return nil
	}

	if len(paths) == 1 {
		if err := sendPhoto(paths[0], caption, replyMarkup); err != nil {
			return nil, 0, err
		}
		return messageIDs, 0, nil
	}
	if len(paths) > 10 {
		return nil, 0, fmt.Errorf("instruction %q has %d screenshots; Telegram albums support at most 10", instructionID, len(paths))
	}

	files := make([]*os.File, 0, len(paths))
	media := make([]telego.InputMedia, 0, len(paths))
	for i, path := range paths {
		file, err := os.Open(path)
		if err != nil {
			for _, opened := range files {
				_ = opened.Close()
			}
			return nil, 0, err
		}
		files = append(files, file)
		item := &telego.InputMediaPhoto{
			Type:  "photo",
			Media: telego.InputFile{File: instructionUploadFile{File: file, filename: filepath.Base(path)}},
		}
		if i == 0 {
			item.Caption = caption
		}
		media = append(media, item)
	}
	messages, sendErr := bot.SendMediaGroup(context.Background(), &telego.SendMediaGroupParams{
		ChatID: tu.ID(chatID), Media: media,
	})
	for _, file := range files {
		if err := file.Close(); err != nil {
			logger.Warning("Failed to close client instruction screenshot:", err)
		}
	}
	if sendErr != nil {
		return nil, 0, sendErr
	}
	for _, message := range messages {
		messageIDs = append(messageIDs, message.MessageID)
	}
	controlsMessage, err := bot.SendMessage(context.Background(), &telego.SendMessageParams{
		ChatID: tu.ID(chatID), Text: "По вопросам и предложениям обращайтесь в тех. поддержку.",
		ReplyMarkup: replyMarkup,
	})
	if err != nil {
		return messageIDs, 0, err
	}
	return messageIDs, controlsMessage.MessageID, nil
}
