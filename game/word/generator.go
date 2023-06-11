// generator.go: generates a random word from a dictionary

package word

type Generator interface {
	Generate(length int) string
}
