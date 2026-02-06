package bot

import (
	"context"
	"log/slog"
	"os"
	"sync"

	"github.com/disgoorg/disgo"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/cache"
	"github.com/disgoorg/disgo/gateway"
	"github.com/disgoorg/disgolink/v3/disgolink"
	"github.com/disgoorg/snowflake/v2"
	"github.com/uzih05/discord-music-bot/internal/command"
	"github.com/uzih05/discord-music-bot/internal/player"
	"github.com/uzih05/discord-music-bot/internal/search"
)

type Bot struct {
	Client      bot.Client
	Lavalink    disgolink.Client
	Players     map[snowflake.ID]*player.GuildPlayer
	SearchCache *search.Cache
	mu          sync.Mutex
}

func NewBot(token string) (*Bot, error) {
	b := &Bot{
		Players:     make(map[snowflake.ID]*player.GuildPlayer),
		SearchCache: search.NewCache(),
	}

	client, err := disgo.New(token,
		bot.WithGatewayConfigOpts(
			gateway.WithIntents(
				gateway.IntentGuilds,
				gateway.IntentGuildVoiceStates,
			),
		),
		bot.WithCacheConfigOpts(
			cache.WithCaches(cache.FlagVoiceStates),
		),
		bot.WithEventListenerFunc(b.onApplicationCommand),
		bot.WithEventListenerFunc(b.onComponentInteraction),
		bot.WithEventListenerFunc(b.onVoiceStateUpdate),
		bot.WithEventListenerFunc(b.onVoiceServerUpdate),
	)
	if err != nil {
		return nil, err
	}

	b.Client = client
	b.Lavalink = disgolink.New(client.ApplicationID(),
		disgolink.WithListenerFunc(b.onTrackStart),
		disgolink.WithListenerFunc(b.onTrackEnd),
		disgolink.WithListenerFunc(b.onTrackException),
		disgolink.WithListenerFunc(b.onTrackStuck),
	)

	return b, nil
}

func (b *Bot) Start(ctx context.Context) error {
	if err := b.registerLavalinkNodes(ctx); err != nil {
		return err
	}

	guildID := os.Getenv("GUILD_ID")
	if guildID != "" {
		id, err := snowflake.Parse(guildID)
		if err != nil {
			slog.Warn("GUILD_ID 파싱 실패, 글로벌 커맨드로 등록합니다", "error", err)
			if _, err := b.Client.Rest().SetGlobalCommands(b.Client.ApplicationID(), command.Commands); err != nil {
				slog.Error("글로벌 커맨드 등록 실패", "error", err)
			}
		} else {
			if _, err := b.Client.Rest().SetGuildCommands(b.Client.ApplicationID(), id, command.Commands); err != nil {
				slog.Error("길드 커맨드 등록 실패", "error", err)
			} else {
				slog.Info("길드 커맨드 등록 완료", "guild_id", id)
			}
		}
	} else {
		if _, err := b.Client.Rest().SetGlobalCommands(b.Client.ApplicationID(), command.Commands); err != nil {
			slog.Error("글로벌 커맨드 등록 실패", "error", err)
		} else {
			slog.Info("글로벌 커맨드 등록 완료")
		}
	}

	return b.Client.OpenGateway(ctx)
}

func (b *Bot) Stop(ctx context.Context) {
	b.Lavalink.Close()
	b.Client.Close(ctx)
}

func (b *Bot) registerLavalinkNodes(ctx context.Context) error {
	host := os.Getenv("LAVALINK_HOST")
	if host == "" {
		host = "localhost"
	}
	port := os.Getenv("LAVALINK_PORT")
	if port == "" {
		port = "2333"
	}
	password := os.Getenv("LAVALINK_PASSWORD")
	if password == "" {
		password = "youshallnotpass"
	}

	node, err := b.Lavalink.AddNode(ctx, disgolink.NodeConfig{
		Name:     "main",
		Address:  host + ":" + port,
		Password: password,
		Secure:   false,
	})
	if err != nil {
		return err
	}

	slog.Info("Lavalink 노드 연결 완료", "name", node.Config().Name)
	return nil
}

func (b *Bot) GetOrCreatePlayer(guildID snowflake.ID) *player.GuildPlayer {
	b.mu.Lock()
	defer b.mu.Unlock()

	if gp, ok := b.Players[guildID]; ok {
		return gp
	}

	gp := player.NewGuildPlayer(guildID)
	b.Players[guildID] = gp
	return gp
}
