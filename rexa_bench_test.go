package rexa

import (
	"regexp"
	"strings"
	"testing"
)

var benchInput = strings.Repeat("the quick brown fox jumps over the lazy dog ", 1000)
var benchInputDigits = "abc" + strings.Repeat("x", 10000) + "12345" + strings.Repeat("y", 10000)

func BenchmarkLiteral_Rexa(b *testing.B) {
	re := MustCompile("lazy dog")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		re.FindString(benchInput)
	}
}

func BenchmarkLiteral_Stdlib(b *testing.B) {
	re := regexp.MustCompile("lazy dog")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		re.FindString(benchInput)
	}
}

func BenchmarkLiteralLong_Rexa(b *testing.B) {
	re := MustCompile("jumps over the lazy")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		re.FindString(benchInput)
	}
}

func BenchmarkLiteralLong_Stdlib(b *testing.B) {
	re := regexp.MustCompile("jumps over the lazy")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		re.FindString(benchInput)
	}
}

func BenchmarkDigits_Rexa(b *testing.B) {
	re := MustCompile(`\d+`)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		re.FindString(benchInputDigits)
	}
}

func BenchmarkDigits_Stdlib(b *testing.B) {
	re := regexp.MustCompile(`\d+`)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		re.FindString(benchInputDigits)
	}
}

func BenchmarkEmail_Rexa(b *testing.B) {
	input := strings.Repeat("hello world ", 500) + "user@example.com" + strings.Repeat(" hello world", 500)
	re := MustCompile(`[\w.+-]+@[\w.-]+\.[\w.-]+`)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		re.FindString(input)
	}
}

func BenchmarkEmail_Stdlib(b *testing.B) {
	input := strings.Repeat("hello world ", 500) + "user@example.com" + strings.Repeat(" hello world", 500)
	re := regexp.MustCompile(`[\w.+-]+@[\w.-]+\.[\w.-]+`)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		re.FindString(input)
	}
}

func BenchmarkIPv4_Rexa(b *testing.B) {
	input := strings.Repeat("no match here ", 500) + "192.168.1.1" + strings.Repeat(" no match here", 500)
	re := MustCompile(`\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		re.FindString(input)
	}
}

func BenchmarkIPv4_Stdlib(b *testing.B) {
	input := strings.Repeat("no match here ", 500) + "192.168.1.1" + strings.Repeat(" no match here", 500)
	re := regexp.MustCompile(`\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		re.FindString(input)
	}
}

func BenchmarkMatch_Rexa(b *testing.B) {
	re := MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		re.MatchString("user@example.com")
	}
}

func BenchmarkMatch_Stdlib(b *testing.B) {
	re := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		re.MatchString("user@example.com")
	}
}
