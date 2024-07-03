package random

import (
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"
)

var randomWords = []string{
	"apple",
	"orange",
	"banana",
	"pear",
	"grape",
	"pineapple",
	"mango",
	"strawberry",
	"raspberry",
	"blueberry",
}

// Init must be called at the start of every program.
//
// If CODECRAFTERS_RANDOM_SEED is set, it will be used to generate predictable random numbers.
func Init() {
	if seed := os.Getenv("CODECRAFTERS_RANDOM_SEED"); seed != "" {
		seedInt, err := strconv.Atoi(seed)
		if err != nil {
			panic(err)
		}

		rand.Seed(int64(seedInt))
	} else {
		rand.Seed(time.Now().UnixNano())
	}
}

// RandomInt returns a random integer between min and max.
func RandomInt(min, max int) int {
	return rand.Intn(max-min) + min
}

// RandomWord returns a random word from the list of words.
func RandomWord() string {
	return randomWords[rand.Intn(len(randomWords))]
}

// RandomWords returns a random list of n words.
func RandomWords(n int) []string {
	return shuffle(randomWords, n)
}

// RandomString returns a random string of 6 words.
func RandomString() string {
	return strings.Join(RandomWords(6), " ")
}

// RandomStrings returns a random list of n strings.
func RandomStrings(n int) []string {
	l := make([]string, n)

	for i := range l {
		l[i] = RandomString()
	}

	return l
}

func shuffle(vals []string, n int) []string {
	if n > len(vals) {
		panic("don't have so many words")
	}

	ret := make([]string, n)

	for i, randIndex := range rand.Perm(len(vals))[:n] {
		ret[i] = vals[randIndex]
	}

	return ret
}

func RandomSelection(count int, characters []string) []string {
	selectedCharacters := make([]string, count)
	for i := 0; i < count; i++ {
		selectedCharacters[i] = characters[rand.Intn(len(characters))]
	}
	return selectedCharacters
}

func RandomStringFromCharacters(totalLength int, characters []string) string {
	characterList := make([]string, totalLength)
	for i := 0; i < totalLength; i++ {
		chosen := characters[rand.Intn(len(characters))]
		if len(chosen) > 2 {
			chosen = " " + chosen + " "
		}
		characterList[i] = chosen
	}

	shuffledString := strings.Join(shuffle(characterList, totalLength), "")
	return shuffledString
}
