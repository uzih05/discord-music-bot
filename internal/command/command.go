package command

import "github.com/disgoorg/disgo/discord"

var (
	dmPerm = false

	ko = discord.LocaleKorean

	// HelpEntries는 /help에서 표시할 명령어 목록 (등록 순서대로)
	HelpEntries = []HelpEntry{
		{Command: "/play <검색어>", Korean: "/재생", Description: "노래를 재생합니다 (검색어 또는 URL)"},
		{Command: "/pause", Korean: "/일시정지", Description: "일시정지 또는 재개합니다"},
		{Command: "/skip", Korean: "/스킵", Description: "현재 곡을 스킵합니다"},
		{Command: "/stop", Korean: "/정지", Description: "재생을 중지하고 채널에서 나갑니다"},
		{Command: "/queue", Korean: "/대기열", Description: "현재 대기열을 표시합니다"},
		{Command: "/move <시작> <끝>", Korean: "/이동", Description: "대기열에서 곡 순서를 이동합니다"},
		{Command: "/remove <위치>", Korean: "/삭제", Description: "대기열에서 곡을 삭제합니다"},
		{Command: "/volume <0-100>", Korean: "/볼륨", Description: "볼륨을 조절합니다"},
		{Command: "/repeat <모드>", Korean: "/반복", Description: "반복 모드 (끄기 / 한 곡 / 전체)"},
		{Command: "/shuffle", Korean: "/셔플", Description: "대기열을 셔플합니다"},
		{Command: "/nowplaying", Korean: "/현재곡", Description: "현재 재생 중인 곡 정보"},
		{Command: "/help", Korean: "/도움말", Description: "이 도움말을 표시합니다"},
	}

	Commands = []discord.ApplicationCommandCreate{
		discord.SlashCommandCreate{
			Name:                     "play",
			NameLocalizations:        map[discord.Locale]string{ko: "재생"},
			Description:              "노래를 재생합니다 (검색어 또는 URL)",
			DescriptionLocalizations: map[discord.Locale]string{ko: "노래를 재생합니다 (검색어 또는 URL)"},
			DMPermission:             &dmPerm,
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionString{
					Name:                     "query",
					NameLocalizations:        map[discord.Locale]string{ko: "검색어"},
					Description:              "검색어 또는 YouTube URL",
					DescriptionLocalizations: map[discord.Locale]string{ko: "검색어 또는 YouTube URL"},
					Required:                 true,
				},
			},
		},
		discord.SlashCommandCreate{
			Name:                     "pause",
			NameLocalizations:        map[discord.Locale]string{ko: "일시정지"},
			Description:              "일시정지 또는 재개합니다",
			DescriptionLocalizations: map[discord.Locale]string{ko: "일시정지 또는 재개합니다"},
			DMPermission:             &dmPerm,
		},
		discord.SlashCommandCreate{
			Name:                     "skip",
			NameLocalizations:        map[discord.Locale]string{ko: "스킵"},
			Description:              "현재 곡을 스킵합니다",
			DescriptionLocalizations: map[discord.Locale]string{ko: "현재 곡을 스킵합니다"},
			DMPermission:             &dmPerm,
		},
		discord.SlashCommandCreate{
			Name:                     "stop",
			NameLocalizations:        map[discord.Locale]string{ko: "정지"},
			Description:              "재생을 중지하고 대기열을 초기화합니다",
			DescriptionLocalizations: map[discord.Locale]string{ko: "재생을 중지하고 대기열을 초기화합니다"},
			DMPermission:             &dmPerm,
		},
		discord.SlashCommandCreate{
			Name:                     "queue",
			NameLocalizations:        map[discord.Locale]string{ko: "대기열"},
			Description:              "현재 대기열을 표시합니다",
			DescriptionLocalizations: map[discord.Locale]string{ko: "현재 대기열을 표시합니다"},
			DMPermission:             &dmPerm,
		},
		discord.SlashCommandCreate{
			Name:                     "move",
			NameLocalizations:        map[discord.Locale]string{ko: "이동"},
			Description:              "대기열에서 곡 순서를 이동합니다",
			DescriptionLocalizations: map[discord.Locale]string{ko: "대기열에서 곡 순서를 이동합니다"},
			DMPermission:             &dmPerm,
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionInt{
					Name:                     "from",
					NameLocalizations:        map[discord.Locale]string{ko: "시작"},
					Description:              "이동할 곡의 번호",
					DescriptionLocalizations: map[discord.Locale]string{ko: "이동할 곡의 번호"},
					Required:                 true,
					MinValue:                 intPtr(1),
				},
				discord.ApplicationCommandOptionInt{
					Name:                     "to",
					NameLocalizations:        map[discord.Locale]string{ko: "끝"},
					Description:              "이동할 위치",
					DescriptionLocalizations: map[discord.Locale]string{ko: "이동할 위치"},
					Required:                 true,
					MinValue:                 intPtr(1),
				},
			},
		},
		discord.SlashCommandCreate{
			Name:                     "remove",
			NameLocalizations:        map[discord.Locale]string{ko: "삭제"},
			Description:              "대기열에서 곡을 삭제합니다",
			DescriptionLocalizations: map[discord.Locale]string{ko: "대기열에서 곡을 삭제합니다"},
			DMPermission:             &dmPerm,
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionInt{
					Name:                     "position",
					NameLocalizations:        map[discord.Locale]string{ko: "위치"},
					Description:              "삭제할 곡의 번호",
					DescriptionLocalizations: map[discord.Locale]string{ko: "삭제할 곡의 번호"},
					Required:                 true,
					MinValue:                 intPtr(1),
				},
			},
		},
		discord.SlashCommandCreate{
			Name:                     "volume",
			NameLocalizations:        map[discord.Locale]string{ko: "볼륨"},
			Description:              "볼륨을 조절합니다 (0-100)",
			DescriptionLocalizations: map[discord.Locale]string{ko: "볼륨을 조절합니다 (0-100)"},
			DMPermission:             &dmPerm,
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionInt{
					Name:                     "level",
					NameLocalizations:        map[discord.Locale]string{ko: "크기"},
					Description:              "볼륨 (0-100)",
					DescriptionLocalizations: map[discord.Locale]string{ko: "볼륨 (0-100)"},
					Required:                 true,
					MinValue:                 intPtr(0),
					MaxValue:                 intPtr(100),
				},
			},
		},
		discord.SlashCommandCreate{
			Name:                     "repeat",
			NameLocalizations:        map[discord.Locale]string{ko: "반복"},
			Description:              "반복 모드를 설정합니다",
			DescriptionLocalizations: map[discord.Locale]string{ko: "반복 모드를 설정합니다"},
			DMPermission:             &dmPerm,
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionString{
					Name:                     "mode",
					NameLocalizations:        map[discord.Locale]string{ko: "모드"},
					Description:              "반복 모드",
					DescriptionLocalizations: map[discord.Locale]string{ko: "반복 모드"},
					Required:                 true,
					Choices: []discord.ApplicationCommandOptionChoiceString{
						{Name: "끄기", Value: "off"},
						{Name: "한 곡 반복", Value: "one"},
						{Name: "전체 반복", Value: "all"},
					},
				},
			},
		},
		discord.SlashCommandCreate{
			Name:                     "shuffle",
			NameLocalizations:        map[discord.Locale]string{ko: "셔플"},
			Description:              "대기열을 셔플합니다",
			DescriptionLocalizations: map[discord.Locale]string{ko: "대기열을 셔플합니다"},
			DMPermission:             &dmPerm,
		},
		discord.SlashCommandCreate{
			Name:                     "nowplaying",
			NameLocalizations:        map[discord.Locale]string{ko: "현재곡"},
			Description:              "현재 재생 중인 곡 정보를 표시합니다",
			DescriptionLocalizations: map[discord.Locale]string{ko: "현재 재생 중인 곡 정보를 표시합니다"},
			DMPermission:             &dmPerm,
		},
		discord.SlashCommandCreate{
			Name:                     "help",
			NameLocalizations:        map[discord.Locale]string{ko: "도움말"},
			Description:              "명령어 도움말을 표시합니다",
			DescriptionLocalizations: map[discord.Locale]string{ko: "명령어 도움말을 표시합니다"},
			DMPermission:             &dmPerm,
		},
	}
)

type HelpEntry struct {
	Command     string
	Korean      string
	Description string
}

func intPtr(v int) *int {
	return &v
}
