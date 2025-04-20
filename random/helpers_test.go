package random

import (
	"fmt"
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
		for i := 0; i < 100; i++ {
			val := RandomInt(min, max)
			assert.GreaterOrEqual(t, val, min)
			assert.Less(t, val, max)
		}
	})

	t.Run("can produce values at min boundary", func(t *testing.T) {
		// Run enough times to likely hit the minimum
		found := false
		for i := 0; i < 1000; i++ {
			if RandomInt(5, 10) == 5 {
				found = true
				break
			}
		}
		assert.True(t, found, "should be able to generate values at the min boundary")
	})

	t.Run("never produces values at max boundary", func(t *testing.T) {
		for i := 0; i < 1000; i++ {
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
		// TODO
		// for i := range 100 {
		for i := 0; i < 100; i++ {
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
		for i := 0; i < 20; i++ {
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

func ExampleRandomInt() {
	os.Setenv("CODECRAFTERS_RANDOM_SEED", "42")
	defer os.Unsetenv("CODECRAFTERS_RANDOM_SEED")
	Init()

	fmt.Println(RandomInt(1, 10))
	fmt.Println(RandomInt(1, 100))
	// Output:
	// 9
	// 54
}

func ExampleRandomString() {
	os.Setenv("CODECRAFTERS_RANDOM_SEED", "42")
	defer os.Unsetenv("CODECRAFTERS_RANDOM_SEED")
	Init()

	fmt.Println(RandomString())
	// Output:
	// strawberry pineapple raspberry blueberry banana orange
}
