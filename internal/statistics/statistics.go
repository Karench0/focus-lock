package statistics

import (
	"time"

	"github.com/Karench0/focus-lock/internal/model"
)

// Aggregate uses the user's local calendar; no URLs or document contents are stored.
func Aggregate(records []model.Record, now time.Time) model.Statistics {
	s := model.Statistics{Days: []model.Day{}, History: []model.Record{}}
	days := map[string]int{}
	for _, r := range records {
		if r.Outcome == "active" {
			continue
		}
		s.Seconds += r.Seconds
		s.AppBlocks += r.AppBlocks
		s.SiteBlocks += r.SiteBlocks
		if r.Outcome == "completed" {
			s.Completed++
		} else {
			s.Interrupted++
		}
		at, err := time.Parse(time.RFC3339, r.StartedAt)
		if err == nil {
			at = at.In(now.Location())
			left := r.Seconds
			for left > 0 {
				next := time.Date(at.Year(), at.Month(), at.Day()+1, 0, 0, 0, 0, at.Location())
				part := int(next.Sub(at).Seconds())
				if part > left {
					part = left
				}
				if part <= 0 {
					break
				}
				days[at.Format("2006-01-02")] += part
				left -= part
				at = next
			}
		}
		if len(s.History) < 100 {
			s.History = append(s.History, r)
		}
	}
	for i := 29; i >= 0; i-- {
		d := now.AddDate(0, 0, -i).Format("2006-01-02")
		s.Days = append(s.Days, model.Day{Date: d, Seconds: days[d]})
	}
	return s
}
