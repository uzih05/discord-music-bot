package search

import (
	"sync"
	"time"

	"github.com/disgoorg/disgolink/v3/lavalink"
	"github.com/disgoorg/snowflake/v2"
)

const PageSize = 5

type PendingSearch struct {
	Tracks    []lavalink.Track
	Page      int
	GuildID   snowflake.ID
	ChannelID snowflake.ID
	UserID    snowflake.ID
	CreatedAt time.Time
}

func (ps *PendingSearch) TotalPages() int {
	pages := len(ps.Tracks) / PageSize
	if len(ps.Tracks)%PageSize != 0 {
		pages++
	}
	return pages
}

func (ps *PendingSearch) PageTracks() []lavalink.Track {
	start := ps.Page * PageSize
	end := start + PageSize
	if end > len(ps.Tracks) {
		end = len(ps.Tracks)
	}
	if start >= len(ps.Tracks) {
		return nil
	}
	return ps.Tracks[start:end]
}

type Cache struct {
	searches map[snowflake.ID]*PendingSearch
	mu       sync.Mutex
}

func NewCache() *Cache {
	return &Cache{
		searches: make(map[snowflake.ID]*PendingSearch),
	}
}

func (sc *Cache) Set(messageID snowflake.ID, s *PendingSearch) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	now := time.Now()
	for id, existing := range sc.searches {
		if now.Sub(existing.CreatedAt) > 5*time.Minute {
			delete(sc.searches, id)
		}
	}

	sc.searches[messageID] = s
}

func (sc *Cache) Get(messageID snowflake.ID) *PendingSearch {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	return sc.searches[messageID]
}

func (sc *Cache) Delete(messageID snowflake.ID) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	delete(sc.searches, messageID)
}
