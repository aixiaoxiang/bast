//Copyright 2018 The axx Authors. All rights reserved.
package ids

import (
	"testing"
	"time"
)

func Benchmark_ID(b *testing.B) {
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		ID()
	}

}

func Benchmark_Parallel_ID(t *testing.B) {
	// t.Logf("start-s")
	start := time.Now().UnixNano() / 1000000
	rn := 20000000
	cds := make(chan int64, rn)
	ds := make(map[int64]struct{}, rn)
	// var wg sync.WaitGroup
	// numProcs := 1 * runtime.GOMAXPROCS(0)
	// wg.Add(numProcs)
	t.ReportAllocs()
	t.ResetTimer()
	t.RunParallel(func(pb *testing.PB) {
		// defer wg.Done()
		for pb.Next() {
			id := ID()
			if id <= start {
				t.Errorf("error=%d", id)
				break
			}
			cds <- id
		}
	})
	go func() {
		// lg := len(cds)
		// t.Logf("finish-s=%d", lg)
		for index := 0; index < rn; index++ {
			id := <-cds
			_, ok := ds[id]
			if ok {
				t.Errorf("exist=%d,%d", id, len(ds))
				break
			}
			ds[id] = struct{}{}
		}
		// t.Logf("finish-e=%d", len(ds))
		// time.AfterFunc(5*time.Second, nil)
	}()
}
