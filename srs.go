package main

import (
	"math"
	"math/rand"
	"sort"
	"time"
)

const MaxItemsPerDay = 10

// Note: will sort items in-place
func GetItemsForToday(items []DBItem) []int {
	now := time.Now()

	// Sort in ascending order of next recall time (oldest first)
	sort.Slice(items, func(a, b int) bool {
		return NextRecallTime(items[a]).Before(NextRecallTime(items[b]))
	})

	// Get items for recall today
	nToday := 0
	itemsToday := []int{}

	for i, item := range items {
		nextRecallTime := NextRecallTime(item)

		// Stop when no more items to review or daily limit reached
		if nextRecallTime.After(now) || nToday >= MaxItemsPerDay {
			break
		}

		itemsToday = append(itemsToday, i)
		nToday++
	}

	// Shuffle items
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(itemsToday), func(a, b int) {
		itemsToday[a], itemsToday[b] = itemsToday[b], itemsToday[a]
	})

	return itemsToday
}

func NextRecallTime(item DBItem) time.Time {
	return item.LastRecalledAt.Add(time.Hour * time.Duration(24*item.Interval))
}

// Implements the SuperMemo SM-2 algorithm
func RecalculateItem(q, recalls, interval uint, ef float32) (uint, uint, float32) {
	if q >= 3 {
		// Recalled correctly
		if recalls == 0 {
			interval = 1
		} else if recalls == 1 {
			interval = 6
		} else {
			interval = uint(math.Ceil(float64(interval) * float64(ef)))
		}

		ef = ef - 0.8 + 0.28*float32(q) - 0.02*float32(q*q)

		if ef < 1.3 {
			ef = 1.3
		}

		recalls++
	} else {
		recalls = 0
		interval = 1
	}

	return recalls, interval, ef
}
