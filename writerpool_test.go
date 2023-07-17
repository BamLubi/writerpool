package writerpool

import (
	"os"
	"sync"
	"testing"
)

func TestWriteFile(t *testing.T) {
	wp := New()

	if err := wp.WriteStringWithDate("20230714", "1", "20230714-data-123456\n"); err != nil {
		t.Error(err)
	}

	if err := wp.Close(); err != nil {
		t.Error(err)
	}
}

func BenchmarkWriteFile(b *testing.B) {
	fwp := New()

	var wg sync.WaitGroup
	for i := 0; i < b.N; i++ {
		wg.Add(1)
		go func(num int) {
			defer wg.Done()
			err := fwp.WriteStringWithDate("20230714", "2", "20230714-data-123456\n")
			if err != nil {
				b.Error(err)
			}
		}(i)
	}
	wg.Wait()

	if err := fwp.Close(); err != nil {
		b.Error(err)
	}
}

func BenchmarkFileWriteString(b *testing.B) {
	file, _ := os.OpenFile("data/20230714/3.txt", os.O_CREATE|os.O_RDWR, 0666)
	defer file.Close()

	var wg sync.WaitGroup
	for i := 0; i < b.N; i++ {
		wg.Add(1)
		go func(num int) {
			defer wg.Done()
			file.WriteString("20230714-data-123456\n")
		}(i)
	}
	wg.Wait()
}

// BenchmarkWriteFile-8			1939741		604.6 ns/op		105 B/op	5 allocs/op
// BenchmarkFileWriteString-8	339056		3306 ns/op		270 B/op	3 allocs/op
