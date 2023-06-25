// generator.go: generates a random word from a dictionary

package word

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewLocalGen(t *testing.T) {
	got := NewLocalGen()
	assert.Greater(t, len(got.words), 0, "words should be loaded")
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
