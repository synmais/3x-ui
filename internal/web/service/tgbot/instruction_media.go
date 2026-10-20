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

var errNoInstructionScreenshots = errors.New("no instruction screenshots found")

type instructionUploadFile struct {
	*os.File
	filename string
}

func (f instructionUploadFile) Name() string { return f.filename }

func instructionKeyboard(buttons []synvpntgbot.InstructionButton) *telego.InlineKeyboardMarkup {
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

func sendClientInstructionMedia(chatID int64, instructionID, caption string) ([]int, error) {
	var instruction *synvpntgbot.ClientInstruction
	for _, callback := range []string{
		"client_instruction_payment", "client_instruction_happ", "client_instruction_incy",
		"client_instruction_clash_mi", "client_instruction_server", "client_instruction_support",
	} {
		page, ok := synvpntgbot.InstructionPageForCallback(callback)
		if ok && page.Guide != nil && page.Guide.ID == instructionID {
			instruction = page.Guide
			break
		}
	}
	if instruction == nil {
		return nil, fmt.Errorf("unknown instruction %q", instructionID)
	}

	entries, err := os.ReadDir(instruction.MediaDirectory())
	if err != nil {
		return nil, err
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
		return nil, fmt.Errorf("%w in %s", errNoInstructionScreenshots, instruction.MediaDirectory())
	}

	messageIDs := make([]int, 0, len(paths))
	if len(paths) == 1 {
		file, err := os.Open(paths[0])
		if err != nil {
			return nil, err
		}
		message, sendErr := bot.SendPhoto(context.Background(), &telego.SendPhotoParams{
			ChatID: tu.ID(chatID), Photo: telego.InputFile{File: instructionUploadFile{File: file, filename: filepath.Base(paths[0])}},
			Caption: caption,
		})
		closeErr := file.Close()
		if sendErr != nil {
			return nil, sendErr
		}
		if closeErr != nil {
			logger.Warning("Failed to close client instruction screenshot:", closeErr)
		}
		return []int{message.MessageID}, nil
	}

	for start := 0; start < len(paths); {
		end := min(start+10, len(paths))
		if len(paths)-end == 1 {
			end--
		}
		files := make([]*os.File, 0, end-start)
		media := make([]telego.InputMedia, 0, end-start)
		openErr := error(nil)
		for i, path := range paths[start:end] {
			file, err := os.Open(path)
			if err != nil {
				openErr = err
				break
			}
			files = append(files, file)
			item := &telego.InputMediaPhoto{
				Type:  "photo",
				Media: telego.InputFile{File: instructionUploadFile{File: file, filename: filepath.Base(path)}},
			}
			if start == 0 && i == 0 {
				item.Caption = caption
			}
			media = append(media, item)
		}
		if openErr != nil {
			for _, file := range files {
				_ = file.Close()
			}
			return nil, openErr
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
			return nil, sendErr
		}
		for _, message := range messages {
			messageIDs = append(messageIDs, message.MessageID)
		}
		start = end
	}
	return messageIDs, nil
}
