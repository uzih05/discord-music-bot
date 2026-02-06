# Discord Music Bot

Go로 작성된 Discord 음악 봇. Lavalink를 사용하여 YouTube 음악을 실시간 스트리밍합니다.

## 기능

- YouTube 검색 및 URL 재생
- 검색 결과를 페이지 형태로 표시 (버튼으로 선택)
- Now Playing 임베드에 컨트롤 버튼 (볼륨, 스킵, 반복, 대기열)
- 대기열 관리, 셔플, 반복 모드 (한 곡 / 전체)
- 재생 진행도 바 자동 업데이트 (15초 간격)
- 곡 종료 후 3분 유휴 시 자동 퇴장
- 슬래시 커맨드 한국어 로컬라이제이션

## 기술 스택

| 구성 요소 | 사용 기술 |
|-----------|----------|
| 언어 | Go 1.24 |
| Discord API | [DisGo](https://github.com/disgoorg/disgo) |
| Lavalink 클라이언트 | [DisGoLink v3](https://github.com/disgoorg/disgolink) |
| 오디오 서버 | [Lavalink v4](https://github.com/lavalink-devs/Lavalink) (Docker) |
| YouTube 소스 | [youtube-source](https://github.com/lavalink-devs/youtube-source) 플러그인 |

## 요구 사항

- Go 1.24 이상
- Docker / Docker Compose
- Discord Bot 토큰 ([Discord Developer Portal](https://discord.com/developers/applications)에서 발급)
- Google 계정 (YouTube OAuth 인증용, 부계정 권장)

## 설치 및 실행

### 1. 프로젝트 클론

```bash
git clone https://github.com/uzih05/discord-music-bot.git
cd discord-music-bot
```

### 2. 환경 변수 설정

`.env.example`을 복사하여 `.env` 파일을 만들고 값을 채웁니다.

```bash
cp .env.example .env
```

```env
BOT_TOKEN=your_discord_bot_token
GUILD_ID=your_guild_id          # 선택사항. 비워두면 글로벌 커맨드로 등록
LAVALINK_HOST=localhost
LAVALINK_PORT=2333
LAVALINK_PASSWORD=youshallnotpass
```

- `GUILD_ID`를 지정하면 해당 서버에만 즉시 커맨드가 등록됩니다 (테스트용).
- 비워두면 글로벌 커맨드로 등록되며, 반영까지 최대 1시간 소요됩니다.

### 3. Lavalink 서버 설정

`lavalink/application.yml` 파일을 생성합니다. 아래는 예시입니다.

```yaml
server:
  port: 2333
  address: 0.0.0.0

lavalink:
  server:
    password: "youshallnotpass"
    sources:
      youtube: false
      bandcamp: true
      soundcloud: true
      twitch: true
      vimeo: true
      http: true
      local: false

  plugins:
    - dependency: "dev.lavalink.youtube:youtube-plugin:1.17.0"
      snapshot: false
      # 1.17.0에서 YouTube 재생이 안 될 경우 최신 스냅샷 사용:
      # - dependency: "dev.lavalink.youtube:youtube-plugin:SNAPSHOT_COMMIT_HASH"
      #   snapshot: true

plugins:
  youtube:
    enabled: true
    allowSearch: true
    allowDirectVideoIds: true
    allowDirectPlaylistIds: true
    clients:
      - MUSIC
      - WEB
      - ANDROID_VR
    oauth:
      enabled: true
      # refreshToken: "첫 인증 후 발급받은 토큰을 여기에 붙여넣기"

logging:
  level:
    root: INFO
    lavalink: INFO
```

### 4. Lavalink 실행

```bash
docker compose up -d
```

### 5. YouTube OAuth 인증 (Google 계정 연동)

YouTube의 재생 제한을 우회하기 위해 Google 계정 OAuth 인증이 필요합니다.
**부계정 사용을 강력히 권장합니다** (메인 계정에 영향이 갈 수 있음).

#### 5-1. 첫 실행 시 인증 코드 확인

Lavalink 컨테이너 로그를 확인합니다.

```bash
docker logs -f discord-music-bot-lavalink-1
```

로그에 아래와 같은 메시지가 출력됩니다.

```
OAUTH INTEGRATION: To give youtube-source access to your Google account,
go to https://www.google.com/device and enter code XXXX-XXX-XXXX
```

#### 5-2. Google 계정으로 승인

1. 브라우저에서 https://www.google.com/device 접속
2. 로그에 출력된 코드 (예: `XXXX-XXX-XXXX`) 입력
3. Google 계정으로 로그인 (부계정 권장)
4. "YouTube에서 내 데이터에 액세스하도록 허용" 승인

#### 5-3. Refresh Token 저장

인증 완료 후 Lavalink 로그에 `refreshToken`이 출력됩니다.

```
OAUTH INTEGRATION: Token retrieved successfully. Refresh token: 1//0eXXXXXXXXXXXXXX...
```

이 토큰을 `lavalink/application.yml`의 oauth 섹션에 저장합니다.

```yaml
plugins:
  youtube:
    oauth:
      enabled: true
      refreshToken: "로그에서_복사한_토큰"
```

저장 후 Lavalink를 재시작하면 이후 재인증 없이 자동으로 YouTube에 접근합니다.

```bash
docker compose restart
```

> Refresh Token은 만료되지 않으므로 한 번만 설정하면 됩니다.
> Google 계정에서 앱 액세스를 취소하면 토큰이 무효화되며 재인증이 필요합니다.

### 6. 봇 빌드 및 실행

```bash
go build -o music-bot .
./music-bot
```

### 7. Discord 봇 초대

[Discord Developer Portal](https://discord.com/developers/applications)에서 봇의 OAuth2 URL을 생성합니다.

- 스코프: `bot`, `applications.commands`
- 권한: Connect, Speak, Send Messages, Embed Links, Manage Messages

## 슬래시 커맨드

| 커맨드 | 한국어 | 설명 |
|--------|--------|------|
| `/play <query>` | `/재생` | 검색어 또는 URL로 노래 재생 |
| `/pause` | `/일시정지` | 일시정지 / 재개 |
| `/skip` | `/스킵` | 현재 곡 스킵 |
| `/stop` | `/정지` | 재생 중지 + 채널 퇴장 |
| `/queue` | `/대기열` | 대기열 표시 |
| `/volume <0-100>` | `/볼륨` | 볼륨 조절 |
| `/repeat <mode>` | `/반복` | 반복 모드 (끄기 / 한 곡 / 전체) |
| `/shuffle` | `/셔플` | 대기열 셔플 |
| `/nowplaying` | `/현재곡` | 현재 재생 곡 정보 |
| `/help` | `/도움말` | 명령어 도움말 표시 |

한국어 커맨드는 Discord 클라이언트 언어가 한국어일 때 자동으로 표시됩니다.

## Now Playing 컨트롤

노래 재생 시 채널에 임베드 메시지가 전송되며, 아래 버튼으로 조작할 수 있습니다.

| 버튼 | 기능 |
|------|------|
| 볼륨 -10 | 볼륨 10% 감소 |
| 스킵 | 다음 곡으로 넘기기 |
| 반복 | 반복 모드 순환 (끄기 > 한 곡 > 전체) |
| 볼륨 +10 | 볼륨 10% 증가 |
| 대기열 | 현재 대기열을 본인에게만 보이는 메시지로 표시 |

버튼은 같은 서버에 있는 누구나 사용할 수 있습니다.

## 프로젝트 구조

```
discord-music-bot/
├── main.go                      # 진입점
├── internal/
│   ├── bot/
│   │   ├── bot.go               # Bot 구조체, 초기화
│   │   ├── handlers.go          # 슬래시 커맨드 및 버튼 핸들러
│   │   └── events.go            # Discord/Lavalink 이벤트 처리
│   ├── player/
│   │   └── player.go            # 길드별 재생 상태 관리
│   ├── search/
│   │   └── search.go            # 검색 결과 캐싱
│   ├── command/
│   │   └── command.go           # 슬래시 커맨드 정의
│   └── embed/
│       └── embed.go             # Discord 임베드 생성
├── docker-compose.yml           # Lavalink Docker 설정
├── lavalink/
│   └── application.yml          # Lavalink 서버 설정
├── .env                         # 환경 변수 (gitignore)
└── .env.example                 # 환경 변수 템플릿
```

## 라이선스

MIT
