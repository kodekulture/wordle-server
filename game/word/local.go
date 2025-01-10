// generator.go: generates a random word from a dictionary

package word

import (
	_ "embed"
	"math/rand/v2"
	"strconv"
	"strings"
)

var (
	//go:embed resources/five_letter_words.txt
	fileContent string

	// Length is the length of the word to be guessed
	Length = 5
)

// localWordGenerator generates a word from one of the words in fileContent
type localWordGenerator struct {
	wordsArray []string
	wordsMap   map[string]struct{}
}

func NewLocalGen() *localWordGenerator {
	g := localWordGenerator{
		wordsMap: make(map[string]struct{}),
	}
	g.loadWords()
	return &g
}

func (g *localWordGenerator) loadWords() {
	g.wordsArray = strings.Split(fileContent, "\n")
	for _, word := range g.wordsArray {
		g.wordsMap[word] = struct{}{}
	}
}

func (g *localWordGenerator) Generate(length int) string {
	if length != Length {
		panic("only " + strconv.Itoa(Length) + " letter words are supported")
	}
	return g.wordsArray[rand.IntN(len(g.wordsArray))]
}

func (g *localWordGenerator) Validate(guess string) bool {
	_, ok := g.wordsMap[guess]
	return ok
}
