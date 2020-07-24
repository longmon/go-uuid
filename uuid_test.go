package uuid

import (
	"testing"
	"time"
)

func TestUUIDDistributed(t *testing.T) {
	InitializeDistributedWithRedis("192.168.79.128:6379")
	var lst int64
	for i := 0; i < 10000; i++ {
		x := Generate()
		if x < lst {
			t.Fatalf("i is %d\ntest failed! the next uuid is less prev, next is %v, prev is %v %b\n", i, x, lst, x)
			return
		}
		lst = x
		time.Sleep(time.Millisecond)
	}
}

func TestUUIDStandalone(t *testing.T) {
	InitializeStandalone()
	var lst int64
	for i := 0; i < 10000; i++ {
		x := Generate()
		if x < lst {
			t.Fatalf("i is %d\ntest failed! the next uuid is less prev, next is %v, prev is %v %b\n", i, x, lst, x)
			return
		}
		lst = x
		time.Sleep(time.Millisecond)
	}
}

func TestUUIDQPS(t *testing.T) {
	InitializeStandalone()
	var amount int64 = 100000000
	var i int64 = 0
	st := time.Now()
	for ; i < amount; i++ {
		Generate()
	}
	used := time.Now().Sub(st)
	t.Logf("Generate 100M UUIDs used time %v, QPS is %f\n", used, float64(amount)/used.Seconds())
}
