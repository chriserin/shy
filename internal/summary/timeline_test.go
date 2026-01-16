package summary

import (
	"testing"
	"time"

	"github.com/chris/shy/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetTimePeriod_Morning tests morning time period (6am-12pm)
func TestGetTimePeriod_Morning(t *testing.T) {
	// 6:00 AM
	ts := time.Date(2026, 1, 14, 6, 0, 0, 0, time.Local).Unix()
	assert.Equal(t, 0, GetTimePeriod(ts)) // Morning = 0

	// 11:59 AM
	ts = time.Date(2026, 1, 14, 11, 59, 59, 0, time.Local).Unix()
	assert.Equal(t, 0, GetTimePeriod(ts))

	// 9:30 AM
	ts = time.Date(2026, 1, 14, 9, 30, 0, 0, time.Local).Unix()
	assert.Equal(t, 0, GetTimePeriod(ts))
}

// TestGetTimePeriod_Afternoon tests afternoon time period (12pm-6pm)
func TestGetTimePeriod_Afternoon(t *testing.T) {
	// 12:00 PM
	ts := time.Date(2026, 1, 14, 12, 0, 0, 0, time.Local).Unix()
	assert.Equal(t, 1, GetTimePeriod(ts)) // Afternoon = 1

	// 5:59 PM
	ts = time.Date(2026, 1, 14, 17, 59, 59, 0, time.Local).Unix()
	assert.Equal(t, 1, GetTimePeriod(ts))

	// 2:30 PM
	ts = time.Date(2026, 1, 14, 14, 30, 0, 0, time.Local).Unix()
	assert.Equal(t, 1, GetTimePeriod(ts))
}

// TestGetTimePeriod_Evening tests evening time period (6pm-12am)
func TestGetTimePeriod_Evening(t *testing.T) {
	// 6:00 PM
	ts := time.Date(2026, 1, 14, 18, 0, 0, 0, time.Local).Unix()
	assert.Equal(t, 2, GetTimePeriod(ts)) // Evening = 2

	// 11:59 PM
	ts = time.Date(2026, 1, 14, 23, 59, 59, 0, time.Local).Unix()
	assert.Equal(t, 2, GetTimePeriod(ts))

	// 9:30 PM
	ts = time.Date(2026, 1, 14, 21, 30, 0, 0, time.Local).Unix()
	assert.Equal(t, 2, GetTimePeriod(ts))
}

// TestGetTimePeriod_Night tests night time period (12am-6am)
func TestGetTimePeriod_Night(t *testing.T) {
	// 12:00 AM
	ts := time.Date(2026, 1, 14, 0, 0, 0, 0, time.Local).Unix()
	assert.Equal(t, 3, GetTimePeriod(ts)) // Night = 3

	// 5:59 AM
	ts = time.Date(2026, 1, 14, 5, 59, 59, 0, time.Local).Unix()
	assert.Equal(t, 3, GetTimePeriod(ts))

	// 3:30 AM
	ts = time.Date(2026, 1, 14, 3, 30, 0, 0, time.Local).Unix()
	assert.Equal(t, 3, GetTimePeriod(ts))
}

// TestGetTimePeriod_Boundaries tests exact boundary times
func TestGetTimePeriod_Boundaries(t *testing.T) {
	// 5:59:59 AM - should be Night
	ts := time.Date(2026, 1, 14, 5, 59, 59, 0, time.Local).Unix()
	assert.Equal(t, 3, GetTimePeriod(ts))

	// 6:00:00 AM - should be Morning
	ts = time.Date(2026, 1, 14, 6, 0, 0, 0, time.Local).Unix()
	assert.Equal(t, 0, GetTimePeriod(ts))

	// 11:59:59 AM - should be Morning
	ts = time.Date(2026, 1, 14, 11, 59, 59, 0, time.Local).Unix()
	assert.Equal(t, 0, GetTimePeriod(ts))

	// 12:00:00 PM - should be Afternoon
	ts = time.Date(2026, 1, 14, 12, 0, 0, 0, time.Local).Unix()
	assert.Equal(t, 1, GetTimePeriod(ts))

	// 17:59:59 PM - should be Afternoon
	ts = time.Date(2026, 1, 14, 17, 59, 59, 0, time.Local).Unix()
	assert.Equal(t, 1, GetTimePeriod(ts))

	// 18:00:00 PM - should be Evening
	ts = time.Date(2026, 1, 14, 18, 0, 0, 0, time.Local).Unix()
	assert.Equal(t, 2, GetTimePeriod(ts))

	// 23:59:59 PM - should be Evening
	ts = time.Date(2026, 1, 14, 23, 59, 59, 0, time.Local).Unix()
	assert.Equal(t, 2, GetTimePeriod(ts))

	// 00:00:00 AM (next day) - should be Night
	ts = time.Date(2026, 1, 15, 0, 0, 0, 0, time.Local).Unix()
	assert.Equal(t, 3, GetTimePeriod(ts))
}

// TestBucketBy_Periodically_AllPeriods tests bucketing across all time periods
func TestBucketBy_Periodically_AllPeriods(t *testing.T) {
	// Given: commands distributed across all time periods
	commands := []models.Command{
		{
			CommandText: "git commit",
			Timestamp:   time.Date(2026, 1, 14, 2, 0, 0, 0, time.Local).Unix(),
		},
		{
			CommandText: "git status",
			Timestamp:   time.Date(2026, 1, 14, 8, 0, 0, 0, time.Local).Unix(),
		},
		{
			CommandText: "go test",
			Timestamp:   time.Date(2026, 1, 14, 13, 0, 0, 0, time.Local).Unix(),
		},
		{
			CommandText: "git push",
			Timestamp:   time.Date(2026, 1, 14, 19, 0, 0, 0, time.Local).Unix(),
		},
	}

	// When: bucketing by time period
	buckets := BucketBy(commands, Periodically)

	// Then: should have 4 buckets (0=Morning, 1=Afternoon, 2=Evening, 3=Night)
	require.Len(t, buckets, 4)

	// And: Night bucket (3) should have 1 command
	require.NotNil(t, buckets[3])
	assert.Len(t, buckets[3].Commands, 1)
	assert.Equal(t, "git commit", buckets[3].Commands[0].CommandText)

	// And: Morning bucket (0) should have 1 command
	require.NotNil(t, buckets[0])
	assert.Len(t, buckets[0].Commands, 1)
	assert.Equal(t, "git status", buckets[0].Commands[0].CommandText)

	// And: Afternoon bucket (1) should have 1 command
	require.NotNil(t, buckets[1])
	assert.Len(t, buckets[1].Commands, 1)
	assert.Equal(t, "go test", buckets[1].Commands[0].CommandText)

	// And: Evening bucket (2) should have 1 command
	require.NotNil(t, buckets[2])
	assert.Len(t, buckets[2].Commands, 1)
	assert.Equal(t, "git push", buckets[2].Commands[0].CommandText)
}

// TestBucketBy_Periodically_SinglePeriod tests bucketing in one time period
func TestBucketBy_Periodically_SinglePeriod(t *testing.T) {
	// Given: commands all in morning
	commands := []models.Command{
		{
			CommandText: "git status",
			Timestamp:   time.Date(2026, 1, 14, 8, 0, 0, 0, time.Local).Unix(),
		},
		{
			CommandText: "go build",
			Timestamp:   time.Date(2026, 1, 14, 9, 0, 0, 0, time.Local).Unix(),
		},
		{
			CommandText: "go test",
			Timestamp:   time.Date(2026, 1, 14, 10, 0, 0, 0, time.Local).Unix(),
		},
	}

	// When: bucketing by time period
	buckets := BucketBy(commands, Periodically)

	// Then: should have only 1 bucket
	require.Len(t, buckets, 1)

	// And: Morning bucket (0) should have all 3 commands
	require.NotNil(t, buckets[0])
	assert.Len(t, buckets[0].Commands, 3)
	assert.Equal(t, "git status", buckets[0].Commands[0].CommandText)
	assert.Equal(t, "go build", buckets[0].Commands[1].CommandText)
	assert.Equal(t, "go test", buckets[0].Commands[2].CommandText)
}

// TestBucketBy_Periodically_TimeRange tests first/last time tracking
func TestBucketBy_Periodically_TimeRange(t *testing.T) {
	// Given: commands with varying timestamps in morning
	firstTime := time.Date(2026, 1, 14, 8, 23, 15, 0, time.Local).Unix()
	middleTime := time.Date(2026, 1, 14, 9, 30, 0, 0, time.Local).Unix()
	lastTime := time.Date(2026, 1, 14, 11, 47, 0, 0, time.Local).Unix()

	commands := []models.Command{
		{
			CommandText: "git status",
			Timestamp:   firstTime,
		},
		{
			CommandText: "go build",
			Timestamp:   middleTime,
		},
		{
			CommandText: "go test",
			Timestamp:   lastTime,
		},
	}

	// When: bucketing by time period
	buckets := BucketBy(commands, Periodically)

	// Then: Morning bucket (0) should track correct time range
	require.NotNil(t, buckets[0])
	assert.Equal(t, firstTime, buckets[0].FirstTime)
	assert.Equal(t, lastTime, buckets[0].LastTime)
}

// TestBucketBy_Periodically_TimeRangeOutOfOrder tests time range with out-of-order commands
func TestBucketBy_Periodically_TimeRangeOutOfOrder(t *testing.T) {
	// Given: commands inserted out of chronological order
	firstTime := time.Date(2026, 1, 14, 8, 0, 0, 0, time.Local).Unix()
	middleTime := time.Date(2026, 1, 14, 9, 0, 0, 0, time.Local).Unix()
	lastTime := time.Date(2026, 1, 14, 10, 0, 0, 0, time.Local).Unix()

	commands := []models.Command{
		{
			CommandText: "middle",
			Timestamp:   middleTime,
		},
		{
			CommandText: "first",
			Timestamp:   firstTime,
		},
		{
			CommandText: "last",
			Timestamp:   lastTime,
		},
	}

	// When: bucketing by time period
	buckets := BucketBy(commands, Periodically)

	// Then: Morning bucket (0) should still track correct first/last
	require.NotNil(t, buckets[0])
	assert.Equal(t, firstTime, buckets[0].FirstTime)
	assert.Equal(t, lastTime, buckets[0].LastTime)
}

// TestBucketBy_EmptyCommands tests empty command list
func TestBucketBy_EmptyCommands(t *testing.T) {
	// Given: empty command list
	commands := []models.Command{}

	// When: bucketing by time period
	buckets := BucketBy(commands, Periodically)

	// Then: should have empty buckets map
	assert.Empty(t, buckets)
}

// TestBucketBy_Periodically_SingleCommand tests single command
func TestBucketBy_Periodically_SingleCommand(t *testing.T) {
	// Given: single command
	timestamp := time.Date(2026, 1, 14, 9, 0, 0, 0, time.Local).Unix()
	commands := []models.Command{
		{
			CommandText: "go build",
			Timestamp:   timestamp,
		},
	}

	// When: bucketing by time period
	buckets := BucketBy(commands, Periodically)

	// Then: should have one bucket with one command
	require.Len(t, buckets, 1)
	require.NotNil(t, buckets[0]) // Morning = 0
	assert.Len(t, buckets[0].Commands, 1)

	// And: first and last time should be the same
	assert.Equal(t, timestamp, buckets[0].FirstTime)
	assert.Equal(t, timestamp, buckets[0].LastTime)
}

// TestGetOrderedPeriods tests chronological period ordering
func TestGetOrderedPeriods(t *testing.T) {
	// When: getting ordered periods
	periods := GetOrderedPeriods()

	// Then: should be in chronological order
	require.Len(t, periods, 4)
	assert.Equal(t, Morning, periods[0])
	assert.Equal(t, Afternoon, periods[1])
	assert.Equal(t, Evening, periods[2])
	assert.Equal(t, Night, periods[3])
}

// TestGetHour tests extracting hour from timestamps
func TestGetHour(t *testing.T) {
	tests := []struct {
		name string
		hour int
		want int
	}{
		{"midnight", 0, 0},
		{"1am", 1, 1},
		{"noon", 12, 12},
		{"1pm", 13, 13},
		{"11pm", 23, 23},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := time.Date(2026, 1, 14, tt.hour, 30, 0, 0, time.Local).Unix()
			got := GetHour(ts)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestBucketBy_Hourly tests grouping commands by hour
func TestBucketBy_Hourly(t *testing.T) {
	commands := []models.Command{
		{
			CommandText: "cmd1",
			Timestamp:   time.Date(2026, 1, 14, 8, 15, 0, 0, time.Local).Unix(),
		},
		{
			CommandText: "cmd2",
			Timestamp:   time.Date(2026, 1, 14, 8, 45, 0, 0, time.Local).Unix(),
		},
		{
			CommandText: "cmd3",
			Timestamp:   time.Date(2026, 1, 14, 9, 10, 0, 0, time.Local).Unix(),
		},
		{
			CommandText: "cmd4",
			Timestamp:   time.Date(2026, 1, 14, 14, 30, 0, 0, time.Local).Unix(),
		},
	}

	buckets := BucketBy(commands, Hourly)

	// Should have 3 buckets (8, 9, 14)
	assert.Len(t, buckets, 3)

	// Hour 8 should have 2 commands
	assert.NotNil(t, buckets[8])
	assert.Equal(t, 2, len(buckets[8].Commands))
	assert.Equal(t, "cmd1", buckets[8].Commands[0].CommandText)
	assert.Equal(t, "cmd2", buckets[8].Commands[1].CommandText)

	// Hour 9 should have 1 command
	assert.NotNil(t, buckets[9])
	assert.Equal(t, 1, len(buckets[9].Commands))
	assert.Equal(t, "cmd3", buckets[9].Commands[0].CommandText)

	// Hour 14 should have 1 command
	assert.NotNil(t, buckets[14])
	assert.Equal(t, 1, len(buckets[14].Commands))
	assert.Equal(t, "cmd4", buckets[14].Commands[0].CommandText)
}

// TestGetOrderedBuckets tests getting bucket IDs in chronological order
func TestGetOrderedBuckets(t *testing.T) {
	commands := []models.Command{
		{Timestamp: time.Date(2026, 1, 14, 14, 0, 0, 0, time.Local).Unix()},
		{Timestamp: time.Date(2026, 1, 14, 8, 0, 0, 0, time.Local).Unix()},
		{Timestamp: time.Date(2026, 1, 14, 20, 0, 0, 0, time.Local).Unix()},
	}

	buckets := BucketBy(commands, Hourly)
	hours := GetOrderedBuckets(buckets)

	// Should be sorted: 8, 14, 20
	require.Len(t, hours, 3)
	assert.Equal(t, 8, hours[0])
	assert.Equal(t, 14, hours[1])
	assert.Equal(t, 20, hours[2])
}

// TestFormatHour tests hour formatting
func TestFormatHour(t *testing.T) {
	tests := []struct {
		hour int
		want string
	}{
		{0, "12am"},
		{1, "1am"},
		{11, "11am"},
		{12, "12pm"},
		{13, "1pm"},
		{23, "11pm"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := FormatHour(tt.hour)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestBucketBy_Hourly_TimeRange tests that first/last times are tracked correctly
func TestBucketBy_Hourly_TimeRange(t *testing.T) {
	commands := []models.Command{
		{
			CommandText: "cmd1",
			Timestamp:   time.Date(2026, 1, 14, 8, 10, 0, 0, time.Local).Unix(),
		},
		{
			CommandText: "cmd2",
			Timestamp:   time.Date(2026, 1, 14, 8, 50, 0, 0, time.Local).Unix(),
		},
		{
			CommandText: "cmd3",
			Timestamp:   time.Date(2026, 1, 14, 8, 30, 0, 0, time.Local).Unix(),
		},
	}

	buckets := BucketBy(commands, Hourly)

	bucket := buckets[8]
	assert.NotNil(t, bucket)

	// First time should be 8:10
	firstTime := time.Unix(bucket.FirstTime, 0)
	assert.Equal(t, 8, firstTime.Hour())
	assert.Equal(t, 10, firstTime.Minute())

	// Last time should be 8:50
	lastTime := time.Unix(bucket.LastTime, 0)
	assert.Equal(t, 8, lastTime.Hour())
	assert.Equal(t, 50, lastTime.Minute())
}

// TestBucketBy_Daily tests grouping commands by day
func TestBucketBy_Daily(t *testing.T) {
	// Given: commands from three different days
	day1Time1 := time.Date(2026, 1, 14, 8, 30, 0, 0, time.Local).Unix()
	day1Time2 := time.Date(2026, 1, 14, 14, 15, 0, 0, time.Local).Unix()
	day2Time := time.Date(2026, 1, 15, 10, 0, 0, 0, time.Local).Unix()
	day3Time := time.Date(2026, 1, 16, 16, 45, 0, 0, time.Local).Unix()

	commands := []models.Command{
		{CommandText: "day1-cmd1", Timestamp: day1Time1},
		{CommandText: "day1-cmd2", Timestamp: day1Time2},
		{CommandText: "day2-cmd", Timestamp: day2Time},
		{CommandText: "day3-cmd", Timestamp: day3Time},
	}

	// When: bucketing by day
	buckets := BucketBy(commands, Daily)

	// Then: should have 3 buckets
	assert.Len(t, buckets, 3)

	// And: bucket IDs should be midnight timestamps in local timezone
	day1Midnight := time.Date(2026, 1, 14, 0, 0, 0, 0, time.Local).Unix()
	day2Midnight := time.Date(2026, 1, 15, 0, 0, 0, 0, time.Local).Unix()
	day3Midnight := time.Date(2026, 1, 16, 0, 0, 0, 0, time.Local).Unix()

	// Verify day 1 bucket
	bucket1 := buckets[int(day1Midnight)]
	require.NotNil(t, bucket1)
	assert.Len(t, bucket1.Commands, 2)
	assert.Equal(t, "day1-cmd1", bucket1.Commands[0].CommandText)
	assert.Equal(t, "day1-cmd2", bucket1.Commands[1].CommandText)

	// Verify day 2 bucket
	bucket2 := buckets[int(day2Midnight)]
	require.NotNil(t, bucket2)
	assert.Len(t, bucket2.Commands, 1)
	assert.Equal(t, "day2-cmd", bucket2.Commands[0].CommandText)

	// Verify day 3 bucket
	bucket3 := buckets[int(day3Midnight)]
	require.NotNil(t, bucket3)
	assert.Len(t, bucket3.Commands, 1)
	assert.Equal(t, "day3-cmd", bucket3.Commands[0].CommandText)
}

// TestBucketBy_Daily_FormatsCorrectly tests that daily buckets format as dates
func TestBucketBy_Daily_FormatsCorrectly(t *testing.T) {
	// Given: commands from a specific day
	commands := []models.Command{
		{CommandText: "cmd", Timestamp: time.Date(2026, 1, 14, 15, 30, 0, 0, time.Local).Unix()},
	}

	// When: bucketing by day
	buckets := BucketBy(commands, Daily)

	// Then: should have 1 bucket
	require.Len(t, buckets, 1)

	// And: the bucket should format its label as a date
	for _, bucket := range buckets {
		label := bucket.FormatLabel()
		assert.Equal(t, "2026-01-14", label)
	}
}

// TestBucketBy_Daily_MidnightBoundary tests commands right at midnight boundaries
func TestBucketBy_Daily_MidnightBoundary(t *testing.T) {
	// Given: commands at midnight boundaries
	lastSecondDay1 := time.Date(2026, 1, 14, 23, 59, 59, 0, time.Local).Unix()
	firstSecondDay2 := time.Date(2026, 1, 15, 0, 0, 0, 0, time.Local).Unix()

	commands := []models.Command{
		{CommandText: "day1", Timestamp: lastSecondDay1},
		{CommandText: "day2", Timestamp: firstSecondDay2},
	}

	// When: bucketing by day
	buckets := BucketBy(commands, Daily)

	// Then: should have 2 separate buckets
	assert.Len(t, buckets, 2)

	day1Midnight := time.Date(2026, 1, 14, 0, 0, 0, 0, time.Local).Unix()
	day2Midnight := time.Date(2026, 1, 15, 0, 0, 0, 0, time.Local).Unix()

	// Verify they're in different buckets
	assert.NotNil(t, buckets[int(day1Midnight)])
	assert.Equal(t, "day1", buckets[int(day1Midnight)].Commands[0].CommandText)

	assert.NotNil(t, buckets[int(day2Midnight)])
	assert.Equal(t, "day2", buckets[int(day2Midnight)].Commands[0].CommandText)
}

// TestBucketBy_CommandCounts tests that command execution counts are tracked
func TestBucketBy_CommandCounts(t *testing.T) {
	// Given: repeated commands in same hour
	commands := []models.Command{
		{CommandText: "git status", Timestamp: time.Date(2026, 1, 14, 9, 10, 0, 0, time.Local).Unix()},
		{CommandText: "git status", Timestamp: time.Date(2026, 1, 14, 9, 15, 0, 0, time.Local).Unix()},
		{CommandText: "git status", Timestamp: time.Date(2026, 1, 14, 9, 20, 0, 0, time.Local).Unix()},
		{CommandText: "go test", Timestamp: time.Date(2026, 1, 14, 9, 25, 0, 0, time.Local).Unix()},
		{CommandText: "go test", Timestamp: time.Date(2026, 1, 14, 9, 30, 0, 0, time.Local).Unix()},
	}

	// When: bucketing by hour
	buckets := BucketBy(commands, Hourly)

	// Then: command counts should be tracked
	bucket := buckets[9]
	require.NotNil(t, bucket)
	require.NotNil(t, bucket.CommandCounts)

	assert.Equal(t, int64(3), bucket.CommandCounts["git status"])
	assert.Equal(t, int64(2), bucket.CommandCounts["go test"])
}
