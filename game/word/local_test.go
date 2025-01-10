// generator.go: generates a random word from a dictionary

package word

import (
	"fmt"
	"math/rand/v2"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewLocalGen(t *testing.T) {
	got := NewLocalGen()
	assert.Greater(t, len(got.wordsArray), 0, "words should be loaded")
}

func Test_localWordGenerator_Generate(t *testing.T) {
	gen := NewLocalGen()
	words := [3]string{}
	for i := 0; i < len(words); i++ {
		words[i] = gen.Generate(Length)
	}
	if words[0] == words[1] && words[1] == words[2] {
		t.Errorf("localWordGenerator.Generate() = %v, %v, %v on 3 consecutive generations; should be unique",
			words[0], words[1], words[2])
	}
}

func Test_localWordGenerator_Validate(t *testing.T) {
	gen := NewLocalGen()
	correct, incorrect := 3, 3
	words := []string{}
	consonants := []rune{'B', 'C', 'D', 'F', 'G', 'H', 'J', 'K', 'L', 'M', 'N', 'P', 'Q', 'R', 'S', 'T', 'V', 'W', 'X', 'Y', 'Z'}
	badWord := func() string {
		rns := make([]rune, 0, Length)
		for range Length {
			rns = append(rns, consonants[rand.IntN(len(consonants))])
		}
		return string(rns)
	}
	for i := 0; i < correct; i++ {
		words = append(words, gen.Generate(Length))
	}
	for i := 0; i < incorrect; i++ {
		words = append(words, badWord())
	}

	for _, tt := range words[:correct] {
		t.Run(fmt.Sprintf("correct %s", tt), func(t *testing.T) {
			assert.True(t, gen.Validate(tt))
		})
	}

	for _, tt := range words[correct:] {
		t.Run(fmt.Sprintf("incorrect %s", tt), func(t *testing.T) {
			assert.False(t, gen.Validate(tt))
		})
	}
}
