package word

import (
	"database/sql"
	"encoding/json"
	"strings"
)

// LetterStatus is an enum type for the Status of a letter in a word guess
type (
	LetterStatus   int
	LetterStatuses []LetterStatus
)

func (s LetterStatuses) Ints() []int {
	ints := make([]int, len(s))
	for i, v := range s {
		ints[i] = int(v)
	}
	return ints
}

const (
	Unknown   LetterStatus = iota // The letter has not been played
	Incorrect                     // The letter is not in the word to be guessed
	Exists                        // The letter is in the word but in the wrong position
	Correct                       // The letter is in the word and in the correct position
)

// Word contains a map of letters to their Status
// and the time this word was played
// for example the word 'WEIRD' would have the following
// Letters mapping
//
// W -> Incorrect
// E -> Correct
// I -> Incorrect
// R -> Exists
// D -> Incorrect
type Word struct {
	Word     string
	PlayedAt sql.NullTime
	Stats    LetterStatuses
}

func New(word string) Word {
	stats := make([]LetterStatus, len(word))
	return Word{strings.ToUpper(word), sql.NullTime{}, stats}
}

func (w Word) Runes() []rune {
	return []rune(w.Word)
}

// Correct returns true if the word is correct
func (w Word) Correct() bool {
	if w.Word == "" || len(w.Stats) == 0 {
		return false
	}
	for _, c := range w.Stats {
		if c != Correct {
			return false
		}
	}
	return true
}

// group returns the number of correct and existing letters
func (w Word) group() (c, a int) {
	for _, s := range w.Stats {
		switch s {
		case Correct:
			c++
		case Exists:
			a++
		default:
			continue
		}
	}
	return
}

// GreaterThan compares `w` with `other` returning true if `w` ranks higher than `other` otherwise false.
// This function is similar to the `Less` function of the `sort.Interface` interface
func (w Word) GreaterThan(other Word) bool {
	thisCorrect, thisExist := w.group()
	itCorrect, itExist := other.group()

	if thisCorrect != itCorrect {
		return thisCorrect > itCorrect
	}
	if thisExist != itExist {
		return thisExist > itExist
	}

	if !w.PlayedAt.Valid {
		return false
	}

	return w.PlayedAt.Time.Before(other.PlayedAt.Time)
}

// Check compares the word to the correct word
// sets the LetterStatus of each letter of `w` *Word
// and returns LetterStatus of each letter of Word accordingly
// Space Complexity: O(n)
// Time Complexity: O(n)
func (w *Word) Check(correctWord Word) []LetterStatus {
	correctRunes := correctWord.Runes()
	instanceRunes := w.Runes()

	wordStatus := make([]LetterStatus, len(instanceRunes))
	for key := range instanceRunes {
		wordStatus[key] = Incorrect
	}

	// check if the lengths match
	if len(instanceRunes) != len(correctRunes) {
		return wordStatus
	}

	// make a dict of the correct letters
	dict := make(map[rune]int)
	for _, v := range correctRunes {
		dict[v] += 1
	}

	// first parse the correct letters
	for i, v := range instanceRunes {
		if v == correctRunes[i] {
			wordStatus[i] = Correct
			dict[v] -= 1
		}
	}

	// parse the letters that have wrong positions
	for i, value := range instanceRunes {
		if wordStatus[i] == Correct {
			continue
		}
		if cnt, ok := dict[value]; ok && cnt > 0 {
			wordStatus[i] = Exists

			dict[value] -= 1
		}
	}
	w.Stats = wordStatus
	return wordStatus
}

func (w *Word) String() string {
	return w.Word
}

func (w Word) MarshalBinary() ([]byte, error) {
	return json.Marshal(w)
}
