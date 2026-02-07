# Lavalink 트랙 로딩 실패 해결 방법

## 증상
```
ERROR 트랙 로딩 실패 error="fault: Something went wrong while looking up the track."
```

---

## 원인 1: OAuth 토큰 만료

`refreshToken`이 오래되면 인증 실패가 발생합니다.

### 해결 방법
새 토큰 발급:
```bash
docker exec -it lavalink java -jar /opt/Lavalink/Lavalink.jar oauth
```
출력된 새 토큰으로 `lavalink/application.yml`의 `refreshToken` 값 업데이트

---

## 원인 2: YouTube Rate Limiting

YouTube가 서버 IP를 일시적으로 차단하는 경우입니다.

### 해결 방법
`lavalink/application.yml`에서 클라이언트 순서 변경 (덜 차단되는 클라이언트 우선):

```yaml
clients:
  - TVHTML5_SIMPLY_EMBEDDED_PLAYER
  - ANDROID_MUSIC
  - ANDROID_VR
  - WEB
```

---

## 원인 3: 특정 영상 제한

일부 영상은 지역 제한 또는 연령 제한이 있어 실패합니다. 이 경우 해당 영상은 재생 불가.

---

## 수정 후 Lavalink 재시작

```powershell
docker-compose restart lavalink
```
