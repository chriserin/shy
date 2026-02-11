package summary

import (
	"fmt"
	"sort"
	"time"

	"github.com/chris/shy/pkg/models"
)

// Bucket represents commands in an hour with time range
type Bucket struct {
	BucketSize    BucketSize
	BucketID      int // 0-23
	Commands      []models.Command
	FirstTime     int64            // Unix timestamp of first command
	LastTime      int64            // Unix timestamp of last command
	CommandCounts map[string]int64 // Map of command text to its number of executions
}

type BucketSize int

const (
	Hourly BucketSize = iota
	Daily
	Weekly
)

// GetHour extracts the hour (0-23) from a Unix timestamp
// Uses local timezone for hour calculation
func GetHour(timestamp int64) int {
	t := time.Unix(timestamp, 0)
	return t.Hour()
}

// BucketByHour groups commands by hour of the day (0-23)
// Returns a map of hour to HourBucket with commands and time range
func BucketBy(commands []models.Command, bucketSize BucketSize) map[int]*Bucket {
	buckets := make(map[int]*Bucket)

	bucketID := 0
	for _, cmd := range commands {

		switch bucketSize {
		case Hourly:
			// no change needed
			bucketID = GetHour(cmd.Timestamp)
		case Daily:
			t := time.Unix(cmd.Timestamp, 0)
			year, month, day := t.Date()
			midnight := time.Date(year, month, day, 0, 0, 0, 0, t.Location())
			bucketID = int(midnight.Unix())
		case Weekly:
			t := time.Unix(cmd.Timestamp, 0)
			_, week := t.ISOWeek()
			// Use year and week number to create a unique bucket ID
			bucketID = week
		}

		// Initialize bucket if it doesn't exist
		if buckets[bucketID] == nil {
			buckets[bucketID] = &Bucket{
				BucketSize: bucketSize,
				BucketID:   bucketID,
				Commands:   []models.Command{},
				FirstTime:  cmd.Timestamp,
				LastTime:   cmd.Timestamp,
			}
		}

		// Append command
		buckets[bucketID].Commands = append(buckets[bucketID].Commands, cmd)

		// Update time range
		if cmd.Timestamp < buckets[bucketID].FirstTime {
			buckets[bucketID].FirstTime = cmd.Timestamp
		}
		if cmd.Timestamp > buckets[bucketID].LastTime {
			buckets[bucketID].LastTime = cmd.Timestamp
		}
	}

	// count each commands exec time in the hour bucket
	for _, bucket := range buckets {
		bucket.CommandCounts = make(map[string]int64)
		for _, cmd := range bucket.Commands {
			bucket.CommandCounts[cmd.CommandText]++
		}
	}

	return buckets
}

// GetOrderedBuckets returns hours that have commands, sorted chronologically
func GetOrderedBuckets(buckets map[int]*Bucket) []int {
	ids := make([]int, 0, len(buckets))
	for _, bucket := range buckets {
		ids = append(ids, bucket.BucketID)
	}
	sort.Ints(ids)
	return ids
}

// FormatHour formats an hour as "8am", "2pm", "12pm", "12am"
func FormatHour(hour int) string {
	if hour == 0 {
		return "12am"
	} else if hour < 12 {
		return fmt.Sprintf("%dam", hour)
	} else if hour == 12 {
		return "12pm"
	} else {
		return fmt.Sprintf("%dpm", hour-12)
	}
}
