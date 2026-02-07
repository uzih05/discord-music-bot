package bot

import (
	"context"
	"log/slog"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgolink/v3/disgolink"
	"github.com/disgoorg/disgolink/v3/lavalink"
	"github.com/disgoorg/snowflake/v2"
	"github.com/uzih05/discord-music-bot/internal/embed"
	"github.com/uzih05/discord-music-bot/internal/player"
)

const idleTimeout = 3 * time.Minute

func (b *Bot) onVoiceStateUpdate(event *events.GuildVoiceStateUpdate) {
	if event.VoiceState.UserID != b.Client.ApplicationID() {
		return
	}
	b.Lavalink.OnVoiceStateUpdate(context.TODO(), event.VoiceState.GuildID, event.VoiceState.ChannelID, event.VoiceState.SessionID)

	if event.VoiceState.ChannelID == nil {
		b.mu.Lock()
		if gp, ok := b.Players[event.VoiceState.GuildID]; ok {
			gp.Clear()
		}
		b.mu.Unlock()
	}
}

func (b *Bot) onVoiceServerUpdate(event *events.VoiceServerUpdate) {
	b.Lavalink.OnVoiceServerUpdate(context.TODO(), event.GuildID, event.Token, *event.Endpoint)
}

func (b *Bot) onApplicationCommand(event *events.ApplicationCommandInteractionCreate) {
	data := event.SlashCommandInteractionData()
	switch data.CommandName() {
	case "play":
		b.handlePlay(event)
	case "pause":
		b.handlePause(event)
	case "skip":
		b.handleSkip(event)
	case "stop":
		b.handleStop(event)
	case "queue":
		b.handleQueue(event)
	case "move":
		b.handleMove(event)
	case "remove":
		b.handleRemove(event)
	case "volume":
		b.handleVolume(event)
	case "repeat":
		b.handleRepeat(event)
	case "shuffle":
		b.handleShuffle(event)
	case "nowplaying":
		b.handleNowPlaying(event)
	case "help":
		b.handleHelp(event)
	}
}

func (b *Bot) onComponentInteraction(event *events.ComponentInteractionCreate) {
	customID := event.Data.CustomID()
	b.handleComponentInteraction(event, customID)
}

func (b *Bot) onTrackStart(p disgolink.Player, event lavalink.TrackStartEvent) {
	guildID := p.GuildID()
	gp := b.GetOrCreatePlayer(guildID)

	gp.StopUpdateLoop()
	b.deleteIdleMessage(gp)
	gp.CancelIdleTimer()

	gp.Mu.Lock()
	channelID := gp.TextChannelID
	gp.Mu.Unlock()

	if channelID == 0 {
		return
	}

	e := embed.NowPlayingEmbed(event.Track, gp, p.Position())
	buttons := embed.NowPlayingButtons(gp)
	msg, err := b.Client.Rest().CreateMessage(channelID, discord.NewMessageCreateBuilder().
		AddEmbeds(e).
		AddContainerComponents(buttons...).
		Build())
	if err != nil {
		slog.Error("Now Playing 메시지 전송 실패", "error", err)
		return
	}

	gp.Mu.Lock()
	gp.NowPlayingMessageID = msg.ID
	gp.NowPlayingChannelID = channelID
	stopCh := make(chan struct{})
	gp.StopUpdateCh = stopCh
	gp.Mu.Unlock()

	go b.nowPlayingUpdateLoop(guildID, stopCh)
}

func (b *Bot) nowPlayingUpdateLoop(guildID snowflake.ID, stopCh chan struct{}) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-stopCh:
			return
		case <-ticker.C:
			b.updateNowPlayingEmbed(guildID)
		}
	}
}

func (b *Bot) updateNowPlayingEmbed(guildID snowflake.ID) {
	gp := b.GetOrCreatePlayer(guildID)

	gp.Mu.Lock()
	msgID := gp.NowPlayingMessageID
	chID := gp.NowPlayingChannelID
	track := gp.CurrentTrack
	gp.Mu.Unlock()

	if msgID == 0 || chID == 0 || track == nil {
		return
	}

	p := b.Lavalink.ExistingPlayer(guildID)
	if p == nil {
		return
	}

	e := embed.NowPlayingEmbed(*track, gp, p.Position())
	buttons := embed.NowPlayingButtons(gp)
	_, err := b.Client.Rest().UpdateMessage(chID, msgID, discord.NewMessageUpdateBuilder().
		SetEmbeds(e).
		SetContainerComponents(buttons...).
		Build())
	if err != nil {
		slog.Debug("Now Playing 업데이트 실패", "error", err)
	}
}

func (b *Bot) onTrackEnd(p disgolink.Player, event lavalink.TrackEndEvent) {
	guildID := p.GuildID()
	gp := b.GetOrCreatePlayer(guildID)

	b.deleteNowPlaying(gp)

	if !event.Reason.MayStartNext() {
		return
	}

	nextTrack := gp.Next()
	if nextTrack == nil {
		b.startIdleTimer(guildID, gp)
		return
	}

	if err := p.Update(context.TODO(), lavalink.WithTrack(*nextTrack)); err != nil {
		slog.Error("다음 곡 재생 실패", "error", err)
	}
}

func (b *Bot) onTrackException(p disgolink.Player, event lavalink.TrackExceptionEvent) {
	guildID := p.GuildID()
	gp := b.GetOrCreatePlayer(guildID)
	slog.Error("트랙 예외 발생", "guild", guildID, "error", event.Exception.Message)
	b.deleteNowPlaying(gp)
}

func (b *Bot) onTrackStuck(p disgolink.Player, event lavalink.TrackStuckEvent) {
	guildID := p.GuildID()
	gp := b.GetOrCreatePlayer(guildID)
	slog.Warn("트랙이 멈춤", "guild", guildID, "threshold", event.Threshold)
	b.deleteNowPlaying(gp)

	nextTrack := gp.Next()
	if nextTrack != nil {
		_ = p.Update(context.TODO(), lavalink.WithTrack(*nextTrack))
	}
}

func (b *Bot) deleteNowPlaying(gp *player.GuildPlayer) {
	gp.StopUpdateLoop()

	gp.Mu.Lock()
	msgID := gp.NowPlayingMessageID
	chID := gp.NowPlayingChannelID
	gp.NowPlayingMessageID = 0
	gp.NowPlayingChannelID = 0
	gp.Mu.Unlock()

	if msgID != 0 && chID != 0 {
		_ = b.Client.Rest().DeleteMessage(chID, msgID)
	}
}

func (b *Bot) startIdleTimer(guildID snowflake.ID, gp *player.GuildPlayer) {
	gp.Mu.Lock()
	channelID := gp.TextChannelID
	gp.Mu.Unlock()

	if channelID == 0 {
		return
	}

	e := embed.IdleEmbed()
	msg, err := b.Client.Rest().CreateMessage(channelID, discord.NewMessageCreateBuilder().
		AddEmbeds(e).
		Build())
	if err != nil {
		slog.Error("대기 중 메시지 전송 실패", "error", err)
		return
	}

	gp.Mu.Lock()
	gp.IdleMessageID = msg.ID
	gp.IdleChannelID = channelID
	gp.IdleTimer = time.AfterFunc(idleTimeout, func() {
		b.handleIdleTimeout(guildID)
	})
	gp.Mu.Unlock()
}

func (b *Bot) handleIdleTimeout(guildID snowflake.ID) {
	gp := b.GetOrCreatePlayer(guildID)

	b.deleteIdleMessage(gp)

	p := b.Lavalink.ExistingPlayer(guildID)
	if p != nil {
		_ = p.Update(context.TODO(), lavalink.WithNullTrack())
		b.Lavalink.RemovePlayer(guildID)
	}

	gp.Clear()
	_ = b.Client.UpdateVoiceState(context.TODO(), guildID, nil, false, false)
	slog.Info("유휴 타임아웃으로 자동 퇴장", "guild", guildID)
}

func (b *Bot) deleteIdleMessage(gp *player.GuildPlayer) {
	gp.Mu.Lock()
	msgID := gp.IdleMessageID
	chID := gp.IdleChannelID
	gp.IdleMessageID = 0
	gp.IdleChannelID = 0
	gp.Mu.Unlock()

	if msgID != 0 && chID != 0 {
		_ = b.Client.Rest().DeleteMessage(chID, msgID)
	}
}
