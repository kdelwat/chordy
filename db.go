package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"
)

type DBItem struct {
	Name               string
	Recalls            uint
	Ef                 float32
	Interval           uint
	ExerciseType       string
	ExerciseDefinition string
	LastRecalledAt     time.Time
}

type DB struct {
	Items []DBItem
}

// https://stackoverflow.com/a/22467409
func fileExists(path string) bool {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return err == nil
}

func DBOpen(path string) (*DB, error) {
	if !fileExists(path) {
		db := DB{Items: DefaultItems()}

		dbJson, err := json.Marshal(db)
		if err != nil {
			return nil, err
		}

		if err = ioutil.WriteFile(path, dbJson, 0644); err != nil {
			return nil, err
		}

		return &db, nil
	} else {
		dbJson, err := ioutil.ReadFile(path)
		if err != nil {
			return nil, err
		}

		var db DB
		if err = json.Unmarshal(dbJson, &db); err != nil {
			return nil, err
		}

		return &db, nil
	}
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
