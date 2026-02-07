# 대기열 관리 기능 구현 계획

## 추가할 명령어

| 영어 | 한국어 | 설명 |
|------|--------|------|
| `/move <from> <to>` | `/이동 <시작> <끝>` | 대기열에서 곡 순서 이동 |
| `/remove <position>` | `/삭제 <위치>` | 대기열에서 곡 삭제 |

## 사용 예시

```
/queue
> 1. 곡A
> 2. 곡B
> 3. 곡C
> 4. 곡D

/move 4 1      → 곡D를 1번으로 이동 (다음에 재생)
/remove 2      → 곡B 삭제
```

---

## 수정할 파일

### 1. `internal/player/player.go`

`QueueList` 함수 뒤에 추가:

```go
func (gp *GuildPlayer) Move(from, to int) (lavalink.Track, bool) {
	gp.Mu.Lock()
	defer gp.Mu.Unlock()

	if from < 1 || from > len(gp.Queue) || to < 1 || to > len(gp.Queue) {
		return lavalink.Track{}, false
	}

	fromIdx := from - 1
	toIdx := to - 1

	track := gp.Queue[fromIdx]
	gp.Queue = append(gp.Queue[:fromIdx], gp.Queue[fromIdx+1:]...)

	newQueue := make([]lavalink.Track, 0, len(gp.Queue)+1)
	newQueue = append(newQueue, gp.Queue[:toIdx]...)
	newQueue = append(newQueue, track)
	newQueue = append(newQueue, gp.Queue[toIdx:]...)
	gp.Queue = newQueue

	return track, true
}

func (gp *GuildPlayer) Remove(pos int) (lavalink.Track, bool) {
	gp.Mu.Lock()
	defer gp.Mu.Unlock()

	if pos < 1 || pos > len(gp.Queue) {
		return lavalink.Track{}, false
	}

	idx := pos - 1
	track := gp.Queue[idx]
	gp.Queue = append(gp.Queue[:idx], gp.Queue[idx+1:]...)

	return track, true
}
```

---

### 2. `internal/command/command.go`

#### HelpEntries에 추가 (queue 다음에):

```go
{Command: "/move <시작> <끝>", Korean: "/이동", Description: "대기열에서 곡 순서를 이동합니다"},
{Command: "/remove <위치>", Korean: "/삭제", Description: "대기열에서 곡을 삭제합니다"},
```

#### Commands에 추가:

```go
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
```

---

### 3. `internal/bot/handlers.go`

#### 핸들러 함수 추가:

```go
func (b *Bot) handleMove(event *events.ApplicationCommandInteractionCreate) {
	data := event.SlashCommandInteractionData()
	from := data.Int("from")
	to := data.Int("to")

	gp := b.GetOrCreatePlayer(*event.GuildID())

	if from == to {
		b.respondEphemeral(event, "같은 위치입니다.")
		return
	}

	track, ok := gp.Move(from, to)
	if !ok {
		b.respondEphemeral(event, "잘못된 위치입니다. /queue로 대기열을 확인하세요.")
		return
	}

	b.respondEphemeral(event, fmt.Sprintf("**%s**을(를) %d번에서 %d번으로 이동했습니다.", track.Info.Title, from, to))
}

func (b *Bot) handleRemove(event *events.ApplicationCommandInteractionCreate) {
	data := event.SlashCommandInteractionData()
	pos := data.Int("position")

	gp := b.GetOrCreatePlayer(*event.GuildID())

	track, ok := gp.Remove(pos)
	if !ok {
		b.respondEphemeral(event, "잘못된 위치입니다. /queue로 대기열을 확인하세요.")
		return
	}

	b.respondEphemeral(event, fmt.Sprintf("**%s**을(를) 대기열에서 삭제했습니다.", track.Info.Title))
}
```

---

### 4. `internal/bot/bot.go` (또는 명령어 라우팅 파일)

switch문에 케이스 추가:

```go
case "move", "이동":
	b.handleMove(event)
case "remove", "삭제":
	b.handleRemove(event)
```

---

## 빌드 및 테스트

```powershell
go build -buildvcs=false -o discord-music-bot.exe .
.\discord-music-bot.exe
```

Discord에서 테스트:
1. `/play`로 여러 곡 추가
2. `/queue`로 대기열 확인
3. `/move 3 1`로 3번 곡을 1번으로 이동
4. `/remove 2`로 2번 곡 삭제
5. `/queue`로 결과 확인
