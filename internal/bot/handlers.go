package bot

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgolink/v3/disgolink"
	"github.com/disgoorg/disgolink/v3/lavalink"
	"github.com/disgoorg/snowflake/v2"
	"github.com/uzih05/discord-music-bot/internal/embed"
	"github.com/uzih05/discord-music-bot/internal/player"
	"github.com/uzih05/discord-music-bot/internal/search"
)

var urlPattern = regexp.MustCompile(`^https?://`)

func (b *Bot) respondEphemeral(event *events.ApplicationCommandInteractionCreate, content string) {
	_ = event.CreateMessage(discord.NewMessageCreateBuilder().
		SetContent(content).
		SetEphemeral(true).
		Build())
}

func (b *Bot) getVoiceChannelID(event *events.ApplicationCommandInteractionCreate) *discord.VoiceState {
	voiceState, ok := b.Client.Caches().VoiceState(*event.GuildID(), event.User().ID)
	if !ok {
		return nil
	}
	return &voiceState
}

func (b *Bot) handlePlay(event *events.ApplicationCommandInteractionCreate) {
	data := event.SlashCommandInteractionData()
	query := data.String("query")

	voiceState := b.getVoiceChannelID(event)
	if voiceState == nil {
		b.respondEphemeral(event, "먼저 음성 채널에 접속해주세요!")
		return
	}

	_ = event.DeferCreateMessage(true)

	isURL := urlPattern.MatchString(query)
	searchQuery := query
	if !isURL {
		searchQuery = lavalink.SearchTypeYouTube.Apply(query)
	}

	gp := b.GetOrCreatePlayer(*event.GuildID())
	b.deleteIdleMessage(gp)
	gp.CancelIdleTimer()
	gp.Mu.Lock()
	gp.TextChannelID = event.Channel().ID()
	gp.Mu.Unlock()

	ctx := context.TODO()

	if err := b.Client.UpdateVoiceState(ctx, *event.GuildID(), voiceState.ChannelID, false, false); err != nil {
		b.updateResponse(event, "음성 채널 연결 실패: "+err.Error())
		return
	}

	b.Lavalink.BestNode().LoadTracksHandler(ctx, searchQuery, disgolink.NewResultHandler(
		func(track lavalink.Track) {
			b.playOrQueue(event, gp, track)
		},
		func(playlist lavalink.Playlist) {
			b.handlePlaylist(event, gp, playlist)
		},
		func(tracks []lavalink.Track) {
			if len(tracks) == 0 {
				b.updateResponse(event, "검색 결과가 없습니다.")
				return
			}

			if isURL {
				b.playOrQueue(event, gp, tracks[0])
				return
			}

			ps := &search.PendingSearch{
				Tracks:    tracks,
				Page:      0,
				GuildID:   *event.GuildID(),
				ChannelID: event.Channel().ID(),
				UserID:    event.User().ID,
				CreatedAt: time.Now(),
			}

			e, components := embed.SearchResultsMessage(ps)
			msg, err := b.Client.Rest().UpdateInteractionResponse(event.ApplicationID(), event.Token(), discord.NewMessageUpdateBuilder().
				SetEmbeds(e).
				SetContainerComponents(components...).
				Build())
			if err != nil {
				slog.Error("검색 결과 전송 실패", "error", err)
				return
			}

			b.SearchCache.Set(msg.ID, ps)
		},
		func() {
			b.updateResponse(event, "검색 결과가 없습니다.")
		},
		func(err error) {
			slog.Error("트랙 로딩 실패", "error", err)
			b.updateResponse(event, "트랙 로딩 실패: "+err.Error())
		},
	))
}

func (b *Bot) playOrQueue(event *events.ApplicationCommandInteractionCreate, gp *player.GuildPlayer, track lavalink.Track) {
	ctx := context.TODO()
	p := b.Lavalink.ExistingPlayer(*event.GuildID())
	if p == nil {
		p = b.Lavalink.Player(*event.GuildID())
		_ = p.Update(ctx, lavalink.WithVolume(gp.Volume))
	}

	if p.Track() == nil {
		gp.SetCurrentTrack(&track)
		if err := p.Update(ctx, lavalink.WithTrack(track)); err != nil {
			b.updateResponse(event, "재생 실패: "+err.Error())
			return
		}
		b.updateResponse(event, fmt.Sprintf("**%s** 재생을 시작합니다!", track.Info.Title))
	} else {
		gp.Add(track)
		b.updateResponse(event, fmt.Sprintf("**%s** 을(를) 대기열에 추가했습니다. (대기열: %d곡)", track.Info.Title, gp.QueueLen()))
	}
}

func (b *Bot) handlePlaylist(event *events.ApplicationCommandInteractionCreate, gp *player.GuildPlayer, playlist lavalink.Playlist) {
	ctx := context.TODO()
	p := b.Lavalink.ExistingPlayer(*event.GuildID())
	if p == nil {
		p = b.Lavalink.Player(*event.GuildID())
		_ = p.Update(ctx, lavalink.WithVolume(gp.Volume))
	}

	tracks := playlist.Tracks
	if len(tracks) == 0 {
		b.updateResponse(event, "플레이리스트가 비어있습니다.")
		return
	}

	if p.Track() == nil {
		first := tracks[0]
		gp.SetCurrentTrack(&first)
		if err := p.Update(ctx, lavalink.WithTrack(first)); err != nil {
			b.updateResponse(event, "재생 실패: "+err.Error())
			return
		}
		gp.Add(tracks[1:]...)
	} else {
		gp.Add(tracks...)
	}
	b.updateResponse(event, fmt.Sprintf("플레이리스트 **%s**에서 %d곡을 추가했습니다.", playlist.Info.Name, len(tracks)))
}

func (b *Bot) handleComponentInteraction(event *events.ComponentInteractionCreate, customID string) {
	// Now Playing 버튼 처리
	if strings.HasPrefix(customID, "np_") {
		b.handleNPButton(event, customID)
		return
	}

	// 검색 결과 버튼 처리
	messageID := event.Message.ID

	ps := b.SearchCache.Get(messageID)
	if ps == nil {
		_ = event.CreateMessage(discord.NewMessageCreateBuilder().
			SetContent("검색 세션이 만료되었습니다. 다시 검색해주세요.").
			SetEphemeral(true).
			Build())
		return
	}

	if event.User().ID != ps.UserID {
		_ = event.CreateMessage(discord.NewMessageCreateBuilder().
			SetContent("이 검색은 다른 사용자의 것입니다.").
			SetEphemeral(true).
			Build())
		return
	}

	switch {
	case strings.HasPrefix(customID, "search_select:"):
		indexStr := strings.TrimPrefix(customID, "search_select:")
		index, err := strconv.Atoi(indexStr)
		if err != nil {
			return
		}

		globalIndex := ps.Page*search.PageSize + index
		if globalIndex >= len(ps.Tracks) {
			return
		}

		track := ps.Tracks[globalIndex]
		b.SearchCache.Delete(messageID)

		gp := b.GetOrCreatePlayer(ps.GuildID)
		ctx := context.TODO()
		p := b.Lavalink.ExistingPlayer(ps.GuildID)
		if p == nil {
			p = b.Lavalink.Player(ps.GuildID)
			_ = p.Update(ctx, lavalink.WithVolume(gp.Volume))
		}

		if p.Track() == nil {
			gp.SetCurrentTrack(&track)
			if err := p.Update(ctx, lavalink.WithTrack(track)); err != nil {
				_ = event.UpdateMessage(discord.NewMessageUpdateBuilder().
					SetContent("재생 실패: " + err.Error()).
					SetEmbeds().
					SetContainerComponents().
					Build())
				return
			}
			_ = event.UpdateMessage(discord.NewMessageUpdateBuilder().
				SetContent(fmt.Sprintf("**%s** 재생을 시작합니다!", track.Info.Title)).
				SetEmbeds().
				SetContainerComponents().
				Build())
		} else {
			gp.Add(track)
			_ = event.UpdateMessage(discord.NewMessageUpdateBuilder().
				SetContent(fmt.Sprintf("**%s** 을(를) 대기열에 추가했습니다. (대기열: %d곡)", track.Info.Title, gp.QueueLen())).
				SetEmbeds().
				SetContainerComponents().
				Build())
		}

	case customID == "search_prev":
		if ps.Page > 0 {
			ps.Page--
		}
		e, components := embed.SearchResultsMessage(ps)
		_ = event.UpdateMessage(discord.NewMessageUpdateBuilder().
			SetEmbeds(e).
			SetContainerComponents(components...).
			Build())

	case customID == "search_next":
		if ps.Page < ps.TotalPages()-1 {
			ps.Page++
		}
		e, components := embed.SearchResultsMessage(ps)
		_ = event.UpdateMessage(discord.NewMessageUpdateBuilder().
			SetEmbeds(e).
			SetContainerComponents(components...).
			Build())

	case customID == "search_cancel":
		b.SearchCache.Delete(messageID)
		_ = event.UpdateMessage(discord.NewMessageUpdateBuilder().
			SetContent("검색을 취소했습니다.").
			SetEmbeds().
			SetContainerComponents().
			Build())
	}
}

func (b *Bot) handlePause(event *events.ApplicationCommandInteractionCreate) {
	p := b.Lavalink.ExistingPlayer(*event.GuildID())
	if p == nil {
		b.respondEphemeral(event, "재생 중인 곡이 없습니다.")
		return
	}

	paused := !p.Paused()
	if err := p.Update(context.TODO(), lavalink.WithPaused(paused)); err != nil {
		b.respondEphemeral(event, "조작 실패: "+err.Error())
		return
	}

	if paused {
		b.respondEphemeral(event, "일시정지했습니다.")
	} else {
		b.respondEphemeral(event, "재생을 재개합니다.")
	}
}

func (b *Bot) handleSkip(event *events.ApplicationCommandInteractionCreate) {
	p := b.Lavalink.ExistingPlayer(*event.GuildID())
	if p == nil {
		b.respondEphemeral(event, "재생 중인 곡이 없습니다.")
		return
	}

	gp := b.GetOrCreatePlayer(*event.GuildID())
	nextTrack := gp.Next()
	if nextTrack == nil {
		_ = p.Update(context.TODO(), lavalink.WithNullTrack())
		b.respondEphemeral(event, "대기열이 비었습니다. 재생을 종료합니다.")
		return
	}

	if err := p.Update(context.TODO(), lavalink.WithTrack(*nextTrack)); err != nil {
		b.respondEphemeral(event, "스킵 실패: "+err.Error())
		return
	}
	b.respondEphemeral(event, fmt.Sprintf("스킵! 다음 곡: **%s**", nextTrack.Info.Title))
}

func (b *Bot) handleStop(event *events.ApplicationCommandInteractionCreate) {
	p := b.Lavalink.ExistingPlayer(*event.GuildID())
	if p != nil {
		_ = p.Update(context.TODO(), lavalink.WithNullTrack())
		b.Lavalink.RemovePlayer(*event.GuildID())
	}

	gp := b.GetOrCreatePlayer(*event.GuildID())
	b.deleteNowPlaying(gp)
	b.deleteIdleMessage(gp)
	gp.Clear()

	_ = b.Client.UpdateVoiceState(context.TODO(), *event.GuildID(), nil, false, false)
	b.respondEphemeral(event, "재생을 중지하고 음성 채널에서 나갔습니다.")
}

func (b *Bot) handleQueue(event *events.ApplicationCommandInteractionCreate) {
	gp := b.GetOrCreatePlayer(*event.GuildID())
	e := embed.QueueEmbed(gp)

	_ = event.CreateMessage(discord.NewMessageCreateBuilder().
		AddEmbeds(e).
		SetEphemeral(true).
		Build())
}

func (b *Bot) handleVolume(event *events.ApplicationCommandInteractionCreate) {
	data := event.SlashCommandInteractionData()
	level := data.Int("level")

	p := b.Lavalink.ExistingPlayer(*event.GuildID())
	if p == nil {
		b.respondEphemeral(event, "재생 중인 곡이 없습니다.")
		return
	}

	gp := b.GetOrCreatePlayer(*event.GuildID())
	gp.Mu.Lock()
	gp.Volume = level
	gp.Mu.Unlock()

	if err := p.Update(context.TODO(), lavalink.WithVolume(level)); err != nil {
		b.respondEphemeral(event, "볼륨 조절 실패: "+err.Error())
		return
	}
	b.respondEphemeral(event, fmt.Sprintf("볼륨을 **%d%%**로 설정했습니다.", level))
	b.updateNowPlayingEmbed(*event.GuildID())
}

func (b *Bot) handleRepeat(event *events.ApplicationCommandInteractionCreate) {
	data := event.SlashCommandInteractionData()
	mode := data.String("mode")

	gp := b.GetOrCreatePlayer(*event.GuildID())
	gp.Mu.Lock()
	switch mode {
	case "one":
		gp.Repeat = player.RepeatOne
	case "all":
		gp.Repeat = player.RepeatAll
	default:
		gp.Repeat = player.RepeatOff
	}
	repeatMode := gp.Repeat
	gp.Mu.Unlock()

	b.respondEphemeral(event, fmt.Sprintf("반복 모드: **%s**", repeatMode))
	b.updateNowPlayingEmbed(*event.GuildID())
}

func (b *Bot) handleShuffle(event *events.ApplicationCommandInteractionCreate) {
	gp := b.GetOrCreatePlayer(*event.GuildID())
	if gp.QueueLen() == 0 {
		b.respondEphemeral(event, "대기열이 비어있습니다.")
		return
	}

	gp.Shuffle()
	b.respondEphemeral(event, fmt.Sprintf("대기열 %d곡을 셔플했습니다!", gp.QueueLen()))
}

func (b *Bot) handleNowPlaying(event *events.ApplicationCommandInteractionCreate) {
	p := b.Lavalink.ExistingPlayer(*event.GuildID())
	if p == nil || p.Track() == nil {
		b.respondEphemeral(event, "재생 중인 곡이 없습니다.")
		return
	}

	gp := b.GetOrCreatePlayer(*event.GuildID())
	e := embed.NowPlayingEmbed(*p.Track(), gp, p.Position())

	_ = event.CreateMessage(discord.NewMessageCreateBuilder().
		AddEmbeds(e).
		SetEphemeral(true).
		Build())
}

func (b *Bot) handleNPButton(event *events.ComponentInteractionCreate, customID string) {
	guildID := *event.GuildID()
	gp := b.GetOrCreatePlayer(guildID)

	switch customID {
	case "np_voldown":
		gp.Mu.Lock()
		gp.Volume -= 10
		if gp.Volume < 0 {
			gp.Volume = 0
		}
		newVol := gp.Volume
		gp.Mu.Unlock()

		if p := b.Lavalink.ExistingPlayer(guildID); p != nil {
			_ = p.Update(context.TODO(), lavalink.WithVolume(newVol))
		}
		b.updateNPMessage(event, guildID)

	case "np_volup":
		gp.Mu.Lock()
		gp.Volume += 10
		if gp.Volume > 100 {
			gp.Volume = 100
		}
		newVol := gp.Volume
		gp.Mu.Unlock()

		if p := b.Lavalink.ExistingPlayer(guildID); p != nil {
			_ = p.Update(context.TODO(), lavalink.WithVolume(newVol))
		}
		b.updateNPMessage(event, guildID)

	case "np_skip":
		p := b.Lavalink.ExistingPlayer(guildID)
		if p == nil {
			_ = event.DeferUpdateMessage()
			return
		}

		nextTrack := gp.Next()
		if nextTrack == nil {
			_ = p.Update(context.TODO(), lavalink.WithNullTrack())
			_ = event.DeferUpdateMessage()
			return
		}

		_ = p.Update(context.TODO(), lavalink.WithTrack(*nextTrack))
		_ = event.DeferUpdateMessage()

	case "np_repeat":
		newMode := gp.NextRepeat()
		_ = newMode
		b.updateNPMessage(event, guildID)

	case "np_queue":
		e := embed.QueueEmbed(gp)
		_ = event.CreateMessage(discord.NewMessageCreateBuilder().
			AddEmbeds(e).
			SetEphemeral(true).
			Build())
	}
}

func (b *Bot) updateNPMessage(event *events.ComponentInteractionCreate, guildID snowflake.ID) {
	gp := b.GetOrCreatePlayer(guildID)
	p := b.Lavalink.ExistingPlayer(guildID)

	gp.Mu.Lock()
	track := gp.CurrentTrack
	gp.Mu.Unlock()

	if p == nil || track == nil {
		_ = event.DeferUpdateMessage()
		return
	}

	e := embed.NowPlayingEmbed(*track, gp, p.Position())
	buttons := embed.NowPlayingButtons(gp)
	_ = event.UpdateMessage(discord.NewMessageUpdateBuilder().
		SetEmbeds(e).
		SetContainerComponents(buttons...).
		Build())
}

func (b *Bot) handleHelp(event *events.ApplicationCommandInteractionCreate) {
	e := embed.HelpEmbed()
	_ = event.CreateMessage(discord.NewMessageCreateBuilder().
		AddEmbeds(e).
		SetEphemeral(true).
		Build())
}

func (b *Bot) updateResponse(event *events.ApplicationCommandInteractionCreate, content string) {
	_, err := b.Client.Rest().UpdateInteractionResponse(event.ApplicationID(), event.Token(), discord.NewMessageUpdateBuilder().
		SetContent(content).
		Build())
	if err != nil {
		slog.Error("응답 업데이트 실패", "error", err)
	}
}
