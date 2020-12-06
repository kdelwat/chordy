package main

import (
	"encoding/json"
	"fmt"
	"github.com/boltdb/bolt"
	"math/rand"
	"sort"
	"time"
)

var CardBucket = []byte("cards")
var MigrationBucket = []byte("migrations")

var Migrated = []byte{1}

type Card struct {
	Name               string
	Recalls            uint
	Ef                 float32
	Interval           uint
	ExerciseType       string
	ExerciseDefinition string
	LastRecalledAt     time.Time
}

func (self *Card) Key() []byte {
	return []byte(self.Name)
}

func (self *Card) Serialize() ([]byte, error) {
	return json.Marshal(self)
}

func DeserializeCard(data []byte) (Card, error) {
	var card Card
	err := json.Unmarshal(data, &card)
	return card, err
}

type DB struct {
	db *bolt.DB
}

func Connect(path string) (*DB, error) {
	db, err := bolt.Open(path, 0600, nil)
	if err != nil {
		return nil, err
	}

	err = db.Update(func(tx *bolt.Tx) error {
		cards, err := tx.CreateBucketIfNotExists(CardBucket)
		if err != nil {
			return err
		}

		migrations, err := tx.CreateBucketIfNotExists(MigrationBucket)
		if err != nil {
			return err
		}

		hasAddedDefaults := migrations.Get([]byte("defaults"))

		if hasAddedDefaults == nil {
			for _, card := range DefaultCards() {
				v, err := card.Serialize()
				if err != nil {
					return err
				}

				err = cards.Put(card.Key(), v)
				if err != nil {
					return err
				}
			}

			err := migrations.Put([]byte("defaults"), Migrated)
			if err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &DB{db: db}, nil
}

func (self *DB) Close() {
	self.db.Close()
}

func (self *DB) Upsert(card Card) error {
	return self.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(CardBucket)

		v, err := card.Serialize()
		if err != nil {
			return err
		}

		return b.Put(card.Key(), v)
	})
}

func min(a, b int) int {
	if a < b {
		return a
	} else {
		return b
	}
}

func (self *DB) GetCardsForToday() ([]Card, error) {
	// Read in all cards which have a next recall time before now (ready for review)
	eligibleCards := []Card{}
	now := time.Now()

	err := self.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(CardBucket)
		b.ForEach(func(k, v []byte) error {
			card, err := DeserializeCard(v)

			if err != nil {
				return err
			}

			if NextRecallTime(card).Before(now) {
				eligibleCards = append(eligibleCards, card)
			}

			return nil
		})
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Sort cards in ascending order of next recall time (oldest first)
	sort.Slice(eligibleCards, func(a, b int) bool {
		return NextRecallTime(eligibleCards[a]).Before(NextRecallTime(eligibleCards[b]))
	})

	// Take subset of cards for this session
	eligibleCards = eligibleCards[:min(MaxItemsPerDay, len(eligibleCards))]

	// Shuffle cards
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(eligibleCards), func(a, b int) {
		eligibleCards[a], eligibleCards[b] = eligibleCards[b], eligibleCards[a]
	})

	return eligibleCards, nil
}

func makeDefaultCard(name, exerciseType, exerciseDefinition string) Card {
	return Card{
		Name:               name,
		Recalls:            0,
		Ef:                 2.5,
		Interval:           0,
		ExerciseType:       exerciseType,
		ExerciseDefinition: exerciseDefinition,
	}
}

func makeDefaultCardWithChord(note, chordForm string) Card {
	return Card{
		Name:               fmt.Sprintf("%s%s (chord)", note, chordForm),
		Recalls:            0,
		Ef:                 2.5,
		Interval:           0,
		ExerciseType:       "chord",
		ExerciseDefinition: fmt.Sprintf("%s%s", note, chordForm),
	}
}

func makeDefaultCardWithScale(note, scaleForm string) Card {
	return Card{
		Name:               fmt.Sprintf("%s %s (scale)", note, scaleForm),
		Recalls:            0,
		Ef:                 2.5,
		Interval:           0,
		ExerciseType:       "scale",
		ExerciseDefinition: fmt.Sprintf("%s %s", note, scaleForm),
	}
}

func DefaultCards() []Card {
	cards := []Card{}

	notes := []string{
		"Ab",
		"A",
		"A#",
		"Bb",
		"B",
		"C",
		"C#",
		"Db",
		"D",
		"D#",
		"Eb",
		"E",
		"F",
		"F#",
		"Gb",
		"G",
		"G#",
	}

	for _, note := range notes {
		// Base note
		cards = append(cards, makeDefaultCard(fmt.Sprintf("%s (note)", note), "note", note))

		// Chords
		cards = append(cards, makeDefaultCardWithChord(note, "maj"))
		cards = append(cards, makeDefaultCardWithChord(note, "min"))
		cards = append(cards, makeDefaultCardWithChord(note, "non"))
		cards = append(cards, makeDefaultCardWithChord(note, "aug"))
		cards = append(cards, makeDefaultCardWithChord(note, "dim"))

		// Scales
		cards = append(cards, makeDefaultCardWithScale(note, "maj"))
	}

	return cards
}
