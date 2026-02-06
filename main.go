package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/uzih05/discord-music-bot/internal/bot"
)

func main() {
	_ = godotenv.Load()

	slog.Info("Discord Music Bot 시작 중...")

	token := os.Getenv("BOT_TOKEN")
	if token == "" {
		slog.Error("BOT_TOKEN 환경변수가 설정되지 않았습니다")
		os.Exit(1)
	}

	b, err := bot.NewBot(token)
	if err != nil {
		slog.Error("봇 생성 실패", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := b.Start(ctx); err != nil {
		slog.Error("봇 시작 실패", "error", err)
		os.Exit(1)
	}

	slog.Info("봇이 실행 중입니다. CTRL+C로 종료합니다.")

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	slog.Info("봇을 종료합니다...")
	b.Stop(ctx)
}
