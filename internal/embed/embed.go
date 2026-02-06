package embed

import (
	"fmt"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgolink/v3/lavalink"
	"github.com/uzih05/discord-music-bot/internal/command"
	"github.com/uzih05/discord-music-bot/internal/player"
	"github.com/uzih05/discord-music-bot/internal/search"
)

const Color = 0x1DB954

func FormatDuration(d lavalink.Duration) string {
	dur := time.Duration(d) * time.Millisecond
	minutes := int(dur.Minutes())
	seconds := int(dur.Seconds()) % 60
	if minutes >= 60 {
		hours := minutes / 60
		minutes = minutes % 60
		return fmt.Sprintf("%d:%02d:%02d", hours, minutes, seconds)
	}
	return fmt.Sprintf("%d:%02d", minutes, seconds)
}

func NowPlayingEmbed(track lavalink.Track, gp *player.GuildPlayer, position lavalink.Duration) discord.Embed {
	gp.Mu.Lock()
	repeatMode := gp.Repeat
	volume := gp.Volume
	queueLen := len(gp.Queue)
	gp.Mu.Unlock()

	builder := discord.NewEmbedBuilder().
		SetTitle("Now Playing").
		SetColor(Color)

	description := fmt.Sprintf("**[%s](%s)**", track.Info.Title, *track.Info.URI)
	if track.Info.Author != "" {
		description += fmt.Sprintf("\n%s", track.Info.Author)
	}

	if track.Info.IsStream {
		description += "\n\n`LIVE`"
	} else {
		posStr := FormatDuration(position)
		totalStr := FormatDuration(track.Info.Length)
		bar := progressBar(position, track.Info.Length, 16)
		description += fmt.Sprintf("\n\n%s\n`%s / %s`", bar, posStr, totalStr)
	}

	builder.SetDescription(description)

	if track.Info.ArtworkURL != nil && *track.Info.ArtworkURL != "" {
		builder.SetThumbnail(*track.Info.ArtworkURL)
	}

	builder.AddField("볼륨", fmt.Sprintf("%d%%", volume), true)
	builder.AddField("반복", repeatMode.String(), true)
	builder.AddField("대기열", fmt.Sprintf("%d곡", queueLen), true)

	return builder.Build()
}

func progressBar(position, total lavalink.Duration, length int) string {
	if total <= 0 {
		return ""
	}

	filled := int(float64(position) / float64(total) * float64(length))
	if filled > length {
		filled = length
	}
	if filled < 0 {
		filled = 0
	}

	bar := ""
	for i := 0; i < length; i++ {
		if i == filled {
			bar += "●"
		} else if i < filled {
			bar += "▬"
		} else {
			bar += "━"
		}
	}
	return bar
}

func QueueEmbed(gp *player.GuildPlayer) discord.Embed {
	gp.Mu.Lock()
	currentTrack := gp.CurrentTrack
	queueLen := len(gp.Queue)
	repeatMode := gp.Repeat
	gp.Mu.Unlock()

	builder := discord.NewEmbedBuilder().
		SetTitle("대기열").
		SetColor(Color)

	description := ""

	if currentTrack != nil {
		description += fmt.Sprintf("**현재 재생:** [%s](%s) `%s`\n\n",
			currentTrack.Info.Title,
			*currentTrack.Info.URI,
			FormatDuration(currentTrack.Info.Length))
	} else {
		description += "현재 재생 중인 곡이 없습니다.\n\n"
	}

	if queueLen == 0 {
		description += "대기열이 비어있습니다."
	} else {
		tracks := gp.QueueList(10)
		for i, track := range tracks {
			duration := FormatDuration(track.Info.Length)
			if track.Info.IsStream {
				duration = "LIVE"
			}
			description += fmt.Sprintf("`%d.` [%s](%s) `%s`\n",
				i+1, track.Info.Title, *track.Info.URI, duration)
		}
		if queueLen > 10 {
			description += fmt.Sprintf("\n... 외 %d곡", queueLen-10)
		}
	}

	builder.SetDescription(description)
	builder.SetFooterText(fmt.Sprintf("총 %d곡 | 반복: %s", queueLen, repeatMode))

	return builder.Build()
}

func SearchResultsMessage(ps *search.PendingSearch) (discord.Embed, []discord.ContainerComponent) {
	tracks := ps.PageTracks()

	builder := discord.NewEmbedBuilder().
		SetTitle("검색 결과").
		SetColor(0xFF6B6B).
		SetFooterText(fmt.Sprintf("페이지 %d/%d | 총 %d개", ps.Page+1, ps.TotalPages(), len(ps.Tracks)))

	description := ""
	for i, track := range tracks {
		duration := FormatDuration(track.Info.Length)
		if track.Info.IsStream {
			duration = "LIVE"
		}
		description += fmt.Sprintf("`%d.` **%s**\n%s · `%s`\n\n",
			ps.Page*search.PageSize+i+1,
			track.Info.Title,
			track.Info.Author,
			duration)
	}
	builder.SetDescription(description)

	if len(tracks) > 0 {
		first := tracks[0]
		if first.Info.ArtworkURL != nil && *first.Info.ArtworkURL != "" {
			builder.SetThumbnail(*first.Info.ArtworkURL)
		}
	}

	var selectButtons []discord.InteractiveComponent
	for i := range tracks {
		selectButtons = append(selectButtons, discord.NewPrimaryButton(
			fmt.Sprintf("%d", ps.Page*search.PageSize+i+1),
			fmt.Sprintf("search_select:%d", i),
		))
	}

	prevDisabled := ps.Page == 0
	nextDisabled := ps.Page >= ps.TotalPages()-1

	navButtons := []discord.InteractiveComponent{
		discord.NewSecondaryButton("◀ 이전", "search_prev").WithDisabled(prevDisabled),
		discord.NewSecondaryButton("다음 ▶", "search_next").WithDisabled(nextDisabled),
		discord.NewDangerButton("취소", "search_cancel"),
	}

	components := []discord.ContainerComponent{
		discord.NewActionRow(selectButtons...),
		discord.NewActionRow(navButtons...),
	}

	return builder.Build(), components
}

func NowPlayingButtons(gp *player.GuildPlayer) []discord.ContainerComponent {
	gp.Mu.Lock()
	volume := gp.Volume
	repeatMode := gp.Repeat
	gp.Mu.Unlock()

	var repeatLabel string
	switch repeatMode {
	case player.RepeatOne:
		repeatLabel = "🔂 한 곡"
	case player.RepeatAll:
		repeatLabel = "🔁 전체"
	default:
		repeatLabel = "🔁 끄기"
	}

	buttons := []discord.InteractiveComponent{
		discord.NewSecondaryButton("🔉 -10", "np_voldown").WithDisabled(volume <= 0),
		discord.NewSecondaryButton("⏭ 스킵", "np_skip"),
		discord.NewSecondaryButton(repeatLabel, "np_repeat"),
		discord.NewSecondaryButton("🔊 +10", "np_volup").WithDisabled(volume >= 100),
		discord.NewSecondaryButton("📜 대기열", "np_queue"),
	}

	return []discord.ContainerComponent{
		discord.NewActionRow(buttons...),
	}
}

func IdleEmbed() discord.Embed {
	return discord.NewEmbedBuilder().
		SetTitle("⏸ 대기 중").
		SetDescription("재생 중인 곡이 없습니다.\n3분 후 자동으로 퇴장합니다.\n\n`/play` 로 노래를 틀어주세요.").
		SetColor(0x808080).
		Build()
}

func HelpEmbed() discord.Embed {
	builder := discord.NewEmbedBuilder().
		SetTitle("명령어 도움말").
		SetColor(Color)

	description := ""
	for _, entry := range command.HelpEntries {
		description += fmt.Sprintf("`%s` (`%s`)\n%s\n\n", entry.Command, entry.Korean, entry.Description)
	}

	description += "---\n"
	description += "Now Playing 메시지의 버튼으로도 볼륨, 스킵, 반복, 대기열을 조작할 수 있습니다."

	builder.SetDescription(description)
	return builder.Build()
}
