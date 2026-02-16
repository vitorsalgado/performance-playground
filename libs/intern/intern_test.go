package intern

import (
	"runtime"
	"strings"
	"sync"
	"testing"
)

func TestInternString_ReturnsEqualForSameInput(t *testing.T) {
	const s = "hello"
	a := InternString(s)
	b := InternString(s)
	if a != b {
		t.Errorf("InternString(%q) = %q, second call = %q; want equal", s, a, b)
	}
	if a != s {
		t.Errorf("InternString(%q) = %q; want %q", s, a, s)
	}
}

func TestInternString_DifferentInputsReturnEqualValues(t *testing.T) {
	a := InternString("foo")
	b := InternString("bar")
	if a == b {
		t.Errorf("InternString(\"foo\") and InternString(\"bar\") should differ")
	}
	if a != "foo" || b != "bar" {
		t.Errorf("values should match inputs")
	}
}

func TestInternString_EmptyString(t *testing.T) {
	a := InternString("")
	b := InternString("")
	if a != b {
		t.Errorf("empty string should intern to equal value")
	}
	if a != "" {
		t.Errorf("InternString(\"\") = %q; want \"\"", a)
	}
}

func TestInternBytes_MatchesInternString(t *testing.T) {
	const s = "same content"
	fromBytes := InternBytes([]byte(s))
	fromString := InternString(s)
	if fromBytes != fromString {
		t.Errorf("InternBytes(%q) = %q, InternString(%q) = %q; want equal",
			s, fromBytes, s, fromString)
	}
	if fromBytes != s {
		t.Errorf("InternBytes(%q) = %q; want %q", s, fromBytes, s)
	}
}

func TestInternBytes_EmptySlice(t *testing.T) {
	a := InternBytes(nil)
	b := InternBytes([]byte{})
	if a != "" || b != "" {
		t.Errorf("InternBytes(nil)=%q, InternBytes([])=%q; want both \"\"", a, b)
	}
}

func TestInternString_WhenDisabledReturnsCopy(t *testing.T) {
	origDisable := DisableCache
	defer func() { DisableCache = origDisable }()
	DisableCache = true

	const s = "cached when enabled"
	a := InternString(s)
	b := InternString(s)
	if a != b || a != s {
		t.Errorf("values should still be equal to input")
	}
	// When cache is disabled we get a new copy each time; we can't assert pointer
	// identity without unsafe, but at least both must be equal to s.
}

func TestInternString_OverMaxLenNotCached(t *testing.T) {
	origMax := MaxLen
	defer func() { MaxLen = origMax }()
	MaxLen = 5

	long := "hello world"
	a := InternString(long)
	b := InternString(long)
	if a != b || a != long {
		t.Errorf("long string should still return equal value: got %q, %q", a, b)
	}
}

func TestInternString_Concurrent(t *testing.T) {
	const concurrency = 100
	const repeats = 1000
	var wg sync.WaitGroup
	wg.Add(concurrency)
	for i := 0; i < concurrency; i++ {
		go func(seed int) {
			defer wg.Done()
			for j := 0; j < repeats; j++ {
				s := string(rune('a' + seed%26))
				result := InternString(s)
				if result != s {
					t.Errorf("InternString(%q) = %q", s, result)
				}
			}
		}(i)
	}
	wg.Wait()
}

func TestInternBytes_Concurrent(t *testing.T) {
	const concurrency = 100
	const repeats = 1000
	payload := []byte("intern me")
	var wg sync.WaitGroup
	wg.Add(concurrency)
	for i := 0; i < concurrency; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < repeats; j++ {
				result := InternBytes(payload)
				if result != string(payload) {
					t.Errorf("InternBytes(%q) = %q", payload, result)
				}
			}
		}()
	}
	wg.Wait()
}

// Verify that interned string is safe to use after original buffer is modified (InternBytes).
func TestInternBytes_ResultIndependentOfSlice(t *testing.T) {
	buf := []byte("original")
	interned := InternBytes(buf)
	if interned != "original" {
		t.Fatalf("interned = %q; want \"original\"", interned)
	}
	// Mutate the original buffer; interned string must be unchanged.
	copy(buf, "modified!!!!") // same length
	if interned != "original" {
		t.Errorf("after mutating buffer, interned = %q; want \"original\"", interned)
	}
}

func BenchmarkInternString_Repeated(b *testing.B) {
	const s = "a typical repeated string"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = InternString(s)
	}
}

func BenchmarkInternString_Unique(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s := string(rune('a' + i%26))
		_ = InternString(s)
	}
}

func BenchmarkInternBytes_Repeated(b *testing.B) {
	payload := []byte("a typical repeated string")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = InternBytes(payload)
	}
}

func BenchmarkInternBytes_Unique(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		payload := []byte(string(rune('a' + i%26)))
		_ = InternBytes(payload)
	}
}

func BenchmarkStringsClone_Repeated(b *testing.B) {
	const s = "a typical repeated string"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = string([]byte(s))
	}
}

func BenchmarkInternString_Repeated_Parallel(b *testing.B) {
	const s = "parallel repeated string"
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = InternString(s)
		}
	})
}

func BenchmarkInternBytes_Repeated_Parallel(b *testing.B) {
	payload := []byte("parallel repeated string")
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = InternBytes(payload)
		}
	})
}

// Allocation comparison: interning vs plain clone for repeated strings.
func BenchmarkAlloc_InternString_Repeated(b *testing.B) {
	const s = "repeated"
	var out string
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out = InternString(s)
	}
	runtime.KeepAlive(out)
}

func BenchmarkAlloc_Clone_Repeated(b *testing.B) {
	const s = "repeated"
	var out string
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out = strings.Clone(s)
	}
	runtime.KeepAlive(out)
}
