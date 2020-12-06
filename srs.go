package main

import (
	"math"
	"time"
)

const MaxItemsPerDay = 10

func NextRecallTime(item DBItem) time.Time {
	return item.LastRecalledAt.Add(time.Hour * time.Duration(24*item.Interval))
}

// Implements the SuperMemo SM-2 algorithm
func RecalculateCard(card DBItem, difficulty uint) DBItem {
	card.LastRecalledAt = time.Now()

	if difficulty >= 3 {
		if card.Recalls == 0 {
			card.Interval = 1
		} else if card.Recalls == 1 {
			card.Interval = 6
		} else {
			card.Interval = uint(math.Ceil(float64(card.Interval) * float64(card.Ef)))
		}

		card.Ef = card.Ef - 0.8 + 0.28*float32(difficulty) - 0.02*float32(difficulty*difficulty)

		if card.Ef < 1.3 {
			card.Ef = 1.3
		}

		card.Recalls = card.Recalls + 1
	} else {
		card.Recalls = 0
		card.Interval = 1
	}

	return card
}
