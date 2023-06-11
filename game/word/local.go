// generator.go: generates a random word from a dictionary

package word

import (
	"math/rand"
	"strconv"
	"strings"
	"time"
)

var (
	// go:embed resources/five_letter_words.txt
	fileContent string

	// Length is the length of the word to be guessed
	Length = 5
)

// localWordGenerator generates a word from one of the words in fileContent
type localWordGenerator struct {
	words []string
	rnd   *rand.Rand
}

func NewLocalGen() *localWordGenerator {
	g := localWordGenerator{
		rnd: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
	g.loadWords()
	return &g
}

func (g *localWordGenerator) loadWords() {
	g.words = strings.Split(fileContent, "\n")
}

func (g *localWordGenerator) Generate(length int) string {
	if length != Length {
		panic("only " + strconv.Itoa(Length) + " letter words are supported")
	}
	return g.words[g.rnd.Intn(len(g.words))]

}
