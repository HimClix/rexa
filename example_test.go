package rexa_test

import (
	"fmt"

	"github.com/himclix/rexa"
)

func ExampleMustCompile() {
	re := rexa.MustCompile(`\d+`)
	fmt.Println(re.FindString("abc123def"))
	// Output: 123
}

func ExampleRegexp_FindStringSubmatch() {
	re := rexa.MustCompile(`(\w+)@(\w+)\.(\w+)`)
	match := re.FindStringSubmatch("user@example.com")
	fmt.Println(match[0]) // full match
	fmt.Println(match[1]) // user
	fmt.Println(match[2]) // example
	fmt.Println(match[3]) // com
	// Output:
	// user@example.com
	// user
	// example
	// com
}

func ExampleRegexp_ReplaceAllString() {
	re := rexa.MustCompile(`\d+`)
	fmt.Println(re.ReplaceAllString("a1b22c333", "N"))
	// Output: aNbNcN
}

func ExampleRegexp_FindAllString() {
	re := rexa.MustCompile(`\d+`)
	fmt.Println(re.FindAllString("a1b22c333", -1))
	// Output: [1 22 333]
}

func ExampleRegexp_Split() {
	re := rexa.MustCompile(`\s+`)
	fmt.Println(re.Split("hello  world foo", -1))
	// Output: [hello world foo]
}

func ExampleCompileWithOptions() {
	re, _ := rexa.CompileWithOptions(`(\w+)\s+\1`, rexa.CompileOptions{
		BacktrackLimit: 100000,
	})
	fmt.Println(re.MatchString("hello hello"))
	fmt.Println(re.MatchString("hello world"))
	// Output:
	// true
	// false
}

func Example_lookahead() {
	re := rexa.MustCompile(`\w+(?=\d)`)
	fmt.Println(re.FindString("abc1"))
	// Output: abc
}

func Example_lookbehind() {
	re := rexa.MustCompile(`(?<=@)\w+`)
	fmt.Println(re.FindString("user@domain"))
	// Output: domain
}

func Example_namedGroups() {
	re := rexa.MustCompile(`(?P<year>\d{4})-(?P<month>\d{2})-(?P<day>\d{2})`)
	match := re.FindStringSubmatch("2026-06-01")
	fmt.Println(match[re.SubexpIndex("year")])
	fmt.Println(match[re.SubexpIndex("month")])
	fmt.Println(match[re.SubexpIndex("day")])
	// Output:
	// 2026
	// 06
	// 01
}
