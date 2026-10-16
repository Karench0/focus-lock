package statistics

import (
	"github.com/Karench0/focus-lock/internal/model"
	"testing"
	"time"
)

func TestLocalDaysSplitAtMidnight(t *testing.T) {
	now := time.Date(2026, 10, 16, 12, 0, 0, 0, time.FixedZone("test", 3*3600))
	rs := []model.Record{{StartedAt: "2026-10-15T23:59:30+03:00", Seconds: 90, Outcome: "completed", AppBlocks: 2, SiteBlocks: 3}, {Seconds: 10, Outcome: "active"}}
	s := Aggregate(rs, now)
	if s.Seconds != 90 || s.Completed != 1 || s.AppBlocks != 2 || s.SiteBlocks != 3 || s.Days[28].Seconds != 30 || s.Days[29].Seconds != 60 {
		t.Fatal(s)
	}
}
