package uuid

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	redis "github.com/go-redis/redis/v8"
)

//UUID 唯一ID生产器
type UUID struct {
	WorkerID int64
	SeriesNO int64
}

//SeriesNOLength 序列号容量
const SeriesNOLength int64 = 0b1111111111111111

//SeriesNOBitCount 序列号占用位数
const SeriesNOBitCount int = 13

//MaxWorkerID 机器码容量
const MaxWorkerID int64 = 0b1111111111

//WorkIDBitCount 机器号占用位数
const WorkIDBitCount int = 10

var a int64 = 0b100111010101110001100000000000000000001000010110011111111111110

//UUID_START_MTS 程序开始时间
const UUID_START_MTS = 1595592154670

var initializeLock = &sync.Once{}
var uuid *UUID
var initErr error
var hasInitialized = false
var lastTime = time.Now()

//InitializeStandalone 初始化单机版UUID
func InitializeStandalone() error {
	if hasInitialized {
		return initErr
	}
	initializeLock.Do(func() {
		now := time.Now()
		uuid = &UUID{
			WorkerID: (now.UnixNano() % MaxWorkerID) << SeriesNOBitCount,
			SeriesNO: 1,
		}
		initErr = fmt.Errorf("UUID has been initialized standalone mode")
		hasInitialized = true
	})
	return nil
}

//InitializeDistributedWithRedis 初始化分布式UUID生成器
//分布式UUID生成器就是在强制多个服务器之间保持时间统一
//从而保证所有生成器生成的ID是都是顺势递增，且不会重复
func InitializeDistributedWithRedis(endpoints string) error {
	if hasInitialized {
		return initErr
	}
	initializeLock.Do(func() {
		now := time.Now()
		uuid = &UUID{
			WorkerID: (now.UnixNano() % MaxWorkerID) << SeriesNOBitCount,
			SeriesNO: 1,
		}
		initErr = fmt.Errorf("UUID has been initialized distributed mode")
		hasInitialized = true
		//判断当前时间是否比服务器时间落后
		assertRedisTS(endpoints)
	})
	return nil
}

//Generate 生产一个ID
func Generate() int64 {
	now := time.Now()
	if now.Unix() < lastTime.Unix() {
		panic("local lock goes back")
	}
	lastTime = now
	atomic.CompareAndSwapInt64(&uuid.SeriesNO, uuid.SeriesNO, (uuid.SeriesNO+1)%SeriesNOLength)
	ts := now.UnixNano() / 1e6
	return ((ts - UUID_START_MTS) << (SeriesNOBitCount + WorkIDBitCount)) | uuid.WorkerID | uuid.SeriesNO
}

func assertRedisTS(endpoints string) {
	rdb := redis.NewClient(&redis.Options{
		Addr: endpoints,
		DB:   0,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	rs := rdb.Time(ctx)
	t, e := rs.Result()
	if e != nil {
		panic(e)
	}
	sub := time.Now().Sub(t)
	if sub < -1*time.Second {
		log.Panicf("Local timestamp is behind remote redis server %v\n", sub)
	}
}
