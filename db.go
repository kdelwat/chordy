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

type DBItem struct {
	Name               string
	Recalls            uint
	Ef                 float32
	Interval           uint
	ExerciseType       string
	ExerciseDefinition string
	LastRecalledAt     time.Time
}

func (self *DBItem) Key() []byte {
	return []byte(self.Name)
}

func (self *DBItem) Serialize() ([]byte, error) {
	return json.Marshal(self)
}

func DeserializeCard(data []byte) (DBItem, error) {
	var card DBItem
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
			for _, item := range DefaultItems() {
				v, err := item.Serialize()
				if err != nil {
					return err
				}

				err = cards.Put(item.Key(), v)
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

func (self *DB) Upsert(item DBItem) error {
	return self.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(CardBucket)

		v, err := item.Serialize()
		if err != nil {
			return err
		}

		return b.Put(item.Key(), v)
	})
}

func min(a, b int) int {
	if a < b {
		return a
	} else {
		return b
	}
}

func (self *DB) GetCardsForToday() ([]DBItem, error) {
	// Read in all cards which have a next recall time before now (ready for review)
	eligibleCards := []DBItem{}
	now := time.Now()

	err := self.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(CardBucket)
		b.ForEach(func(k, v []byte) error {
			item, err := DeserializeCard(v)

			if err != nil {
				return err
			}

			if NextRecallTime(item).Before(now) {
				eligibleCards = append(eligibleCards, item)
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

func makeDefaultItem(name, exerciseType, exerciseDefinition string) DBItem {
	return DBItem{
		Name:               name,
		Recalls:            0,
		Ef:                 2.5,
		Interval:           0,
		ExerciseType:       exerciseType,
		ExerciseDefinition: exerciseDefinition,
	}
}

func DefaultItems() []DBItem {
	items := []DBItem{}

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
		items = append(items, makeDefaultItem(fmt.Sprintf("%s (note)", note), "note", note))

		// Chords
		items = append(items, makeDefaultItem(fmt.Sprintf("%smaj (chord)", note), "chord", note))
		items = append(items, makeDefaultItem(fmt.Sprintf("%smin (chord)", note), "chord", note))
		items = append(items, makeDefaultItem(fmt.Sprintf("%snon (chord)", note), "chord", note))
	}

	return items
}
