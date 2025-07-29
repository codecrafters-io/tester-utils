package random

import (
	"math"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInit(t *testing.T) {
	t.Run("uses seed from environment variable", func(t *testing.T) {
		// Set a specific seed
		os.Setenv("CODECRAFTERS_RANDOM_SEED", "42")
		defer os.Unsetenv("CODECRAFTERS_RANDOM_SEED")

		Init()

		// Generate some values
		val1 := RandomInt(0, 100)
		val2 := RandomInt(0, 100)

		// Reset with the same seed
		os.Setenv("CODECRAFTERS_RANDOM_SEED", "42")
		Init()

		// Should get the same sequence
		assert.Equal(t, val1, RandomInt(0, 100))
		assert.Equal(t, val2, RandomInt(0, 100))
	})

	t.Run("works without seed environment variable", func(t *testing.T) {
		os.Unsetenv("CODECRAFTERS_RANDOM_SEED")
		Init()

		// Just ensure it doesn't panic
		RandomInt(0, 100)
	})
}

func TestRandomInt(t *testing.T) {
	os.Setenv("CODECRAFTERS_RANDOM_SEED", "123")
	defer os.Unsetenv("CODECRAFTERS_RANDOM_SEED")
	Init()

	t.Run("returns values within the range", func(t *testing.T) {
		min, max := 10, 20
		for range 100 {
			val := RandomInt(min, max)
			assert.GreaterOrEqual(t, val, min)
			assert.Less(t, val, max)
		}
	})

	t.Run("can produce values at min boundary", func(t *testing.T) {
		// Run enough times to likely hit the minimum
		found := false
		for range 1000 {
			if RandomInt(5, 10) == 5 {
				found = true
				break
			}
		}
		assert.True(t, found, "should be able to generate values at the min boundary")
	})

	t.Run("never produces values at max boundary", func(t *testing.T) {
		for range 1000 {
			assert.NotEqual(t, 10, RandomInt(5, 10))
		}
	})
}

func TestRandomUniqueInts(t *testing.T) {
	Init()

	t.Run("panics if count is greater than the range", func(t *testing.T) {
		assert.PanicsWithValue(t, "can't generate more unique random integers than the range of possible values", func() {
			RandomInts(1, 5, 5)
		})
	})

	t.Run("returns all possible values when count equals the range", func(t *testing.T) {
		result := RandomInts(0, 100, 100)
		expected := make([]int, 100)
		for i := range 100 {
			expected[i] = i
		}
		assert.ElementsMatch(t, expected, result)
	})

	t.Run("returns unique values", func(t *testing.T) {
		result := RandomInts(0, 50, 20)
		assert.Len(t, result, 20)

		// Check for uniqueness
		seen := make(map[int]bool)
		for _, val := range result {
			assert.False(t, seen[val], "values should be unique")
			seen[val] = true
		}
	})
}

func TestRandomFloat64(t *testing.T) {
	Init()

	t.Run("values are within range [min, max)", func(t *testing.T) {
		min, max := 0.0, 1.0
		for range 1000 {
			val := RandomFloat64(min, max)
			assert.GreaterOrEqual(t, val, min, "value should be greater than or equal to min boundary")
			assert.Less(t, val, max, "value should be less than max boundary")
		}
	})

	t.Run("panics if max is smaller than min", func(t *testing.T) {
		assert.PanicsWithValue(t, "max boundary is less than min boundary", func() {
			RandomFloat64(1.0, 0.0)
		})
	})

	t.Run("mean is within ±0.01 of expected in ≥95 of 100 runs (10k samples each)", func(t *testing.T) {
		min, max := 0.0, 1.0
		runs := 100
		samples := 10000
		tolerance := 0.01
		expectedMean := (min + max) / 2
		meansWithinTolerance := 0

		for range runs {
			var total float64
			for range samples {
				total += RandomFloat64(min, max)
			}
			mean := total / float64(samples)
			if math.Abs(mean-expectedMean) <= tolerance {
				meansWithinTolerance++
			}
		}

		assert.GreaterOrEqual(t, meansWithinTolerance, 95, "mean should be within ±0.01 of expected in ≥95 of 100 runs (10k samples each)")
	})
}

func TestRandomFloat64s(t *testing.T) {
	Init()

	t.Run("returns specified number of random Float64s", func(t *testing.T) {
		count := 100
		result := RandomFloat64s(0, 1.0, count)
		assert.Equal(t, len(result), count, "slice length should be equal to count")
	})
}

func TestRandomWord(t *testing.T) {
	os.Setenv("CODECRAFTERS_RANDOM_SEED", "42")
	defer os.Unsetenv("CODECRAFTERS_RANDOM_SEED")
	Init()

	t.Run("returns a word from the predefined list", func(t *testing.T) {
		word := RandomWord()
		assert.Contains(t, randomWords, word)
	})

	t.Run("can return different values on subsequent calls", func(t *testing.T) {
		// Reset with the seed
		os.Setenv("CODECRAFTERS_RANDOM_SEED", "77")
		Init()

		// Generate a bunch of words and verify we get at least 2 different ones
		seen := make(map[string]bool)
		for range 20 {
			seen[RandomWord()] = true
			if len(seen) >= 2 {
				break
			}
		}
		assert.GreaterOrEqual(t, len(seen), 2, "should be able to generate at least 2 different words")
	})
}

func TestRandomWords(t *testing.T) {
	os.Setenv("CODECRAFTERS_RANDOM_SEED", "12345")
	defer os.Unsetenv("CODECRAFTERS_RANDOM_SEED")
	Init()

	t.Run("returns the requested number of words", func(t *testing.T) {
		words := RandomWords(5)
		assert.Len(t, words, 5)

		for _, word := range words {
			assert.Contains(t, randomWords, word)
		}
	})

	t.Run("can return more words than in the original array", func(t *testing.T) {
		words := RandomWords(20)
		assert.Len(t, words, 20)
	})
}

func TestRandomString(t *testing.T) {
	os.Setenv("CODECRAFTERS_RANDOM_SEED", "987")
	defer os.Unsetenv("CODECRAFTERS_RANDOM_SEED")
	Init()

	t.Run("returns a space-separated string of words", func(t *testing.T) {
		str := RandomString()
		parts := strings.Split(str, " ")
		assert.Len(t, parts, 6)

		for _, word := range parts {
			assert.Contains(t, randomWords, word)
		}
	})
}

func TestRandomStrings(t *testing.T) {
	os.Setenv("CODECRAFTERS_RANDOM_SEED", "333")
	defer os.Unsetenv("CODECRAFTERS_RANDOM_SEED")
	Init()

	t.Run("returns the requested number of strings", func(t *testing.T) {
		strs := RandomStrings(3)
		assert.Len(t, strs, 3)

		for _, str := range strs {
			parts := strings.Split(str, " ")
			assert.Len(t, parts, 6)
		}
	})
}

func TestRandomElementFromArray(t *testing.T) {
	os.Setenv("CODECRAFTERS_RANDOM_SEED", "8675309")
	defer os.Unsetenv("CODECRAFTERS_RANDOM_SEED")
	Init()

	t.Run("returns an element from the array", func(t *testing.T) {
		array := []string{"a", "b", "c", "d", "e"}
		element := RandomElementFromArray(array)
		assert.Contains(t, array, element)
	})

	t.Run("works with different types", func(t *testing.T) {
		numbers := []int{1, 2, 3, 4, 5}
		number := RandomElementFromArray(numbers)
		assert.Contains(t, numbers, number)

		bools := []bool{true, false}
		boolean := RandomElementFromArray(bools)
		assert.Contains(t, bools, boolean)
	})
}

func TestRandomElementsFromArray(t *testing.T) {
	os.Setenv("CODECRAFTERS_RANDOM_SEED", "1111")
	defer os.Unsetenv("CODECRAFTERS_RANDOM_SEED")
	Init()

	t.Run("returns the requested number of elements", func(t *testing.T) {
		array := []string{"a", "b", "c", "d", "e"}
		elements := RandomElementsFromArray(array, 3)
		assert.Len(t, elements, 3)

		for _, element := range elements {
			assert.Contains(t, array, element)
		}
	})

	t.Run("handles requests larger than array size", func(t *testing.T) {
		array := []int{1, 2, 3}
		elements := RandomElementsFromArray(array, 10)
		assert.Len(t, elements, 10)

		for _, element := range elements {
			assert.Contains(t, array, element)
		}
	})

	t.Run("generates elements uniquely when possible", func(t *testing.T) {
		array := []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"}
		elements := RandomElementsFromArray(array, 5)

		// Check if all elements are unique by checking the length of a set
		uniqueElements := make(map[string]bool)
		for _, e := range elements {
			uniqueElements[e] = true
		}

		assert.Len(t, uniqueElements, 5, "should select unique elements when array is large enough")
	})
}

func TestSeededRandomInt(t *testing.T) {
	os.Setenv("CODECRAFTERS_RANDOM_SEED", "42")
	defer os.Unsetenv("CODECRAFTERS_RANDOM_SEED")
	Init()

	assert.Equal(t, RandomInt(1, 10), 9)
	assert.Equal(t, RandomInt(1, 100), 54)
}

func TestSeededRandomFloat64(t *testing.T) {
	os.Setenv("CODECRAFTERS_RANDOM_SEED", "42")
	defer os.Unsetenv("CODECRAFTERS_RANDOM_SEED")
	Init()

	assert.Equal(t, RandomFloat64(1, 100), 37.92980774361663)
	assert.Equal(t, RandomFloat64(1, 100), 7.534049182558273)
}

func TestSeededRandomString(t *testing.T) {
	os.Setenv("CODECRAFTERS_RANDOM_SEED", "42")
	defer os.Unsetenv("CODECRAFTERS_RANDOM_SEED")
	Init()

	assert.Equal(t, RandomString(), "strawberry pineapple raspberry blueberry banana orange")
}

func TestShuffleArray(t *testing.T) {
	os.Setenv("CODECRAFTERS_RANDOM_SEED", "12345")
	defer os.Unsetenv("CODECRAFTERS_RANDOM_SEED")
	Init()

	t.Run("returns all elements from the original array", func(t *testing.T) {
		original := []string{"a", "b", "c", "d", "e"}
		shuffled := ShuffleArray(original)

		assert.Len(t, shuffled, len(original))
		assert.ElementsMatch(t, original, shuffled)
	})

	t.Run("works with different types", func(t *testing.T) {
		numbers := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
		shuffled := ShuffleArray(numbers)

		assert.Len(t, shuffled, len(numbers))
		assert.ElementsMatch(t, numbers, shuffled)
	})

	t.Run("handles empty array", func(t *testing.T) {
		empty := []string{}
		shuffled := ShuffleArray(empty)

		assert.Len(t, shuffled, 0)
		assert.Equal(t, empty, shuffled)
	})

	t.Run("handles single element array", func(t *testing.T) {
		single := []string{"only"}
		shuffled := ShuffleArray(single)

		assert.Len(t, shuffled, 1)
		assert.Equal(t, single, shuffled)
	})

	t.Run("produces consistent results with same seed", func(t *testing.T) {
		original := []string{"x", "y", "z", "w", "v"}

		// First shuffle with seed
		os.Setenv("CODECRAFTERS_RANDOM_SEED", "999")
		Init()
		shuffled1 := ShuffleArray(original)

		// Second shuffle with same seed
		os.Setenv("CODECRAFTERS_RANDOM_SEED", "999")
		Init()
		shuffled2 := ShuffleArray(original)

		assert.Equal(t, shuffled1, shuffled2, "same seed should produce same shuffle")
	})
}
