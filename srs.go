package main

import (
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
