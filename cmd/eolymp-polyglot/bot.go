package main

import (
	tg "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
)

func BotStart() {
	bot, err := tg.NewBotAPI(conf.Telegram.Token)
	if err != nil {
		log.Panic(err)
	}
	log.Printf("Started %s", bot.Self.UserName)

	bot.Debug = false

	u := tg.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message != nil && update.Message.Chat.ID == conf.Telegram.ChatId {
			findAndUpdateProblem(bot, update.Message)
		}
	}
}

func findAndUpdateProblem(bot *tg.BotAPI, msg *tg.Message) {
	for _, problem := range conf.Telegram.Problems {
		if problem.Id == msg.Text {
			bot.Send(replyToMsg(msg, "Started to update the problem"))
			problemId := problem.PId
			pid := &problemId
			if err := DownloadAndImportProblem(problem.Link, pid); err != nil {
				bot.Send(replyToMsg(msg, err.Error()))
			} else {
				bot.Send(replyToMsg(msg, "Finished"))
			}
			return
		}
	}
	bot.Send(replyToMsg(msg, "Problem not found"))
}

func replyToMsg(msgToRep *tg.Message, text string) tg.MessageConfig {
	msg := tg.NewMessage(msgToRep.Chat.ID, text)
	msg.ReplyToMessageID = msgToRep.MessageID
	return msg
}
