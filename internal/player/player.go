package player

import (
	"math/rand/v2"
	"sync"
	"time"

	"github.com/disgoorg/disgolink/v3/lavalink"
	"github.com/disgoorg/snowflake/v2"
)

type RepeatMode int

const (
	RepeatOff RepeatMode = iota
	RepeatOne
	RepeatAll
)

func (r RepeatMode) String() string {
	switch r {
	case RepeatOne:
		return "한 곡 반복"
	case RepeatAll:
		return "전체 반복"
	default:
		return "끄기"
	}
}

type GuildPlayer struct {
	GuildID              snowflake.ID
	TextChannelID        snowflake.ID
	Queue                []lavalink.Track
	NowPlayingMessageID  snowflake.ID
	NowPlayingChannelID  snowflake.ID
	Repeat               RepeatMode
	CurrentTrack         *lavalink.Track
	Volume               int
	StopUpdateCh         chan struct{}
	IdleTimer            *time.Timer
	IdleMessageID        snowflake.ID
	IdleChannelID        snowflake.ID
	Mu                   sync.Mutex
}

func NewGuildPlayer(guildID snowflake.ID) *GuildPlayer {
	return &GuildPlayer{
		GuildID: guildID,
		Volume:  50,
	}
}

func (gp *GuildPlayer) Add(tracks ...lavalink.Track) {
	gp.Mu.Lock()
	defer gp.Mu.Unlock()
	gp.Queue = append(gp.Queue, tracks...)
}

func (gp *GuildPlayer) Next() *lavalink.Track {
	gp.Mu.Lock()
	defer gp.Mu.Unlock()

	if gp.Repeat == RepeatOne && gp.CurrentTrack != nil {
		return gp.CurrentTrack
	}

	if gp.Repeat == RepeatAll && gp.CurrentTrack != nil {
		gp.Queue = append(gp.Queue, *gp.CurrentTrack)
	}

	if len(gp.Queue) == 0 {
		gp.CurrentTrack = nil
		return nil
	}

	next := gp.Queue[0]
	gp.Queue = gp.Queue[1:]
	gp.CurrentTrack = &next
	return &next
}

func (gp *GuildPlayer) SetCurrentTrack(track *lavalink.Track) {
	gp.Mu.Lock()
	defer gp.Mu.Unlock()
	gp.CurrentTrack = track
}

func (gp *GuildPlayer) Shuffle() {
	gp.Mu.Lock()
	defer gp.Mu.Unlock()

	for i := len(gp.Queue) - 1; i > 0; i-- {
		j := rand.IntN(i + 1)
		gp.Queue[i], gp.Queue[j] = gp.Queue[j], gp.Queue[i]
	}
}

func (gp *GuildPlayer) StopUpdateLoop() {
	gp.Mu.Lock()
	defer gp.Mu.Unlock()
	if gp.StopUpdateCh != nil {
		close(gp.StopUpdateCh)
		gp.StopUpdateCh = nil
	}
}

func (gp *GuildPlayer) CancelIdleTimer() {
	gp.Mu.Lock()
	defer gp.Mu.Unlock()
	if gp.IdleTimer != nil {
		gp.IdleTimer.Stop()
		gp.IdleTimer = nil
	}
}

func (gp *GuildPlayer) Clear() {
	if gp.StopUpdateCh != nil {
		close(gp.StopUpdateCh)
		gp.StopUpdateCh = nil
	}
	gp.Mu.Lock()
	defer gp.Mu.Unlock()
	if gp.IdleTimer != nil {
		gp.IdleTimer.Stop()
		gp.IdleTimer = nil
	}
	gp.Queue = nil
	gp.CurrentTrack = nil
	gp.Repeat = RepeatOff
	gp.NowPlayingMessageID = 0
	gp.NowPlayingChannelID = 0
	gp.IdleMessageID = 0
	gp.IdleChannelID = 0
}

func (gp *GuildPlayer) NextRepeat() RepeatMode {
	gp.Mu.Lock()
	defer gp.Mu.Unlock()
	switch gp.Repeat {
	case RepeatOff:
		gp.Repeat = RepeatOne
	case RepeatOne:
		gp.Repeat = RepeatAll
	default:
		gp.Repeat = RepeatOff
	}
	return gp.Repeat
}

func (gp *GuildPlayer) QueueLen() int {
	gp.Mu.Lock()
	defer gp.Mu.Unlock()
	return len(gp.Queue)
}

func (gp *GuildPlayer) QueueList(max int) []lavalink.Track {
	gp.Mu.Lock()
	defer gp.Mu.Unlock()
	if len(gp.Queue) <= max {
		result := make([]lavalink.Track, len(gp.Queue))
		copy(result, gp.Queue)
		return result
	}
	result := make([]lavalink.Track, max)
	copy(result, gp.Queue[:max])
	return result
}

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
