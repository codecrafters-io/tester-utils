package random

import (
	"math/big"
	"math/rand"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"
)

// rng is our package-level random number generator
var rng *rand.Rand

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
	var source rand.Source
	if seed := os.Getenv("CODECRAFTERS_RANDOM_SEED"); seed != "" {
		seedInt, err := strconv.Atoi(seed)
		if err != nil {
			panic(err)
		}
		source = rand.NewSource(int64(seedInt))
	} else {
		source = rand.NewSource(time.Now().UnixNano())
	}

	rng = rand.New(source)
}

// RandomInt returns a random integer between [min, max).
func RandomInt(min, max int) int {
	return rng.Intn(max-min) + min
}

// RandomInts returns an array of `count` unique random integers between [min, max).
// It panics if count is greater than the range of possible values.
func RandomInts(min, max int, count int) []int {
	randomInts := []int{}

	if count > max-min {
		panic("can't generate more unique random integers than the range of possible values")
	}

	for range count {
		randomInt := RandomInt(min, max)
		for slices.Contains(randomInts, randomInt) {
			randomInt = RandomInt(min, max)
		}
		randomInts = append(randomInts, randomInt)
	}

	return randomInts
}

// RandomFloat64 returns a random float64 number between [min, max)
func RandomFloat64(min, max float64) float64 {
	if max < min {
		panic("max boundary is less than min boundary")
	}

	if min == max {
		return min
	}

	// Generate random value in [0, 1) using big.Rat
	randomInZeroToOne := new(big.Rat).SetFloat64(rng.Float64())

	// Convert min and max to rational numbers
	bigMin := new(big.Rat).SetFloat64(min)
	bigMax := new(big.Rat).SetFloat64(max)

	// Calculate range: max - min
	diff := new(big.Rat).Sub(bigMax, bigMin)

	// Scale random value to range: [0, diff)
	scaledToRange := new(big.Rat).Mul(diff, randomInZeroToOne)

	// Translate random value to range: [min, max)
	result := new(big.Rat).Add(scaledToRange, bigMin)

	// Convert back to float64
	// 'exact' maybe be true or false, because float64 is not big enough to
	// represent every possible value of big.Rat, so we ignore it
	resultFloat, _ := result.Float64()
	return resultFloat
}

// RandomFloat64s returns an array of `count` random Float64 values between [min, max).
func RandomFloat64s(min, max float64, count int) []float64 {
	randomFloats := make([]float64, count)
	for i := range count {
		randomFloats[i] = RandomFloat64(min, max)
	}
	return randomFloats
}

// RandomWord returns a random word from the list of words.
func RandomWord() string {
	return randomWords[rng.Intn(len(randomWords))]
}

// RandomWords returns a random list of n words.
func RandomWords(n int) []string {
	return RandomElementsFromArray(randomWords, n)
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

func RandomElementFromArray[T any](arr []T) T {
	return RandomElementsFromArray(arr, 1)[0]
}

func RandomElementsFromArray[T any](arr []T, count int) []T {
	// Randomly selects `count` unique elements from the given array
	// and returns them in a new array.
	for count > len(arr) {
		// If we need more elements than the array has, we'll just append the array to itself repeatedly.
		arr = append(arr, arr...)
	}
	elements := make([]T, count)
	indices := rng.Perm(len(arr))[:count]
	for i, randIndex := range indices {
		elements[i] = arr[randIndex]
	}

	return elements
}

func ShuffleArray[T any](arr []T) []T {
	return RandomElementsFromArray(arr, len(arr))
}
