package bitring

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBitRing_ShouldTrigger(t *testing.T) {
	t.Parallel()

	type mockEvent struct {
		index         int
		event         bool
		wantTriggered bool
	}

	tcs := []struct {
		name             string
		windowSize       int
		minConsecutive   int
		triggerThreshold float64
		events           []mockEvent
	}{
		{
			name:             "without trigger",
			windowSize:       4,
			minConsecutive:   2,
			triggerThreshold: 0.5,
			events: []mockEvent{
				{index: 1, event: false, wantTriggered: false},
				{index: 2, event: false, wantTriggered: false},
				{index: 3, event: false, wantTriggered: false},
				{index: 4, event: false, wantTriggered: false},
			},
		}, {
			name:             "over min consecutive",
			windowSize:       32,
			minConsecutive:   4,
			triggerThreshold: 1,
			events: []mockEvent{
				{index: 1, event: true, wantTriggered: false},
				{index: 2, event: true, wantTriggered: false},
				{index: 3, event: true, wantTriggered: false},
				{index: 4, event: true, wantTriggered: true},
			},
		}, {
			name:             "over trigger threshold",
			windowSize:       32,
			minConsecutive:   8,
			triggerThreshold: 0.5,
			events: []mockEvent{
				{index: 1, event: false, wantTriggered: false},
				{index: 2, event: false, wantTriggered: false},
				{index: 3, event: true, wantTriggered: false},
				{index: 4, event: true, wantTriggered: false},
				{index: 5, event: false, wantTriggered: false},
				{index: 6, event: true, wantTriggered: false},
				{index: 6, event: true, wantTriggered: true},
			},
		}, {
			name:             "over window size",
			windowSize:       8,
			minConsecutive:   3,
			triggerThreshold: 0.5,
			events: []mockEvent{
				{index: 1, event: false, wantTriggered: false},
				{index: 2, event: true, wantTriggered: false},
				{index: 3, event: false, wantTriggered: false},
				{index: 4, event: false, wantTriggered: false},
				{index: 5, event: true, wantTriggered: false},
				{index: 6, event: true, wantTriggered: false},
				{index: 7, event: false, wantTriggered: false},
				{index: 8, event: true, wantTriggered: false},
				{index: 9, event: true, wantTriggered: true},
				{index: 10, event: false, wantTriggered: false},
				{index: 11, event: true, wantTriggered: true},
				{index: 12, event: false, wantTriggered: true},
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			br := NewBitRing(tc.windowSize, tc.minConsecutive, tc.triggerThreshold)

			for _, evt := range tc.events {
				br.Add(evt.event)

				triggered := br.ShouldTrigger()
				assert.Equal(t, evt.wantTriggered, triggered, "index of event: %d", evt.index)
			}
		})
	}
}
