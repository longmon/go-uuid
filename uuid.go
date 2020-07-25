package uuid

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	etcdv3 "github.com/coreos/etcd/clientv3"
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

//UUID_TIMESTAMP_KEY 保存成员的时间截
const UUID_TIMESTAMP_KEY = "/UUID/timestamps"

//UUID_WORKER_ID_KEY 注册成员
const UUID_WORKER_ID_KEY = "/UUID/workers"

//UUID_START_MTS 程序开始时间
const UUID_START_MTS = 1595592154670

var initializeLock = &sync.Once{}
var uuid *UUID
var initErr error
var hasInitialized = false
var lastTime = time.Now()

var etcdClient *etcdv3.Client

func NewUUID(workerID int64) *UUID {
	if workerID > MaxWorkerID {
		log.Panicf("Worker ID take %d bit should less than %d", WorkIDBitCount, MaxWorkerID)
	}
	return &UUID{
		WorkerID: workerID << SeriesNOBitCount,
		SeriesNO: 1,
	}
}

//InitializeStandalone 初始化单机版UUID
func InitializeStandalone() error {
	if hasInitialized {
		return initErr
	}
	initializeLock.Do(func() {
		uuid = NewUUID(1)
		initErr = fmt.Errorf("UUID has been initialized standalone mode")
		hasInitialized = true
	})
	return nil
}

//InitializeDistributedWithEtcd 初始化分布式UUID生成器
//分布式UUID生成器就是在强制多个服务器之间保持时间相对统一
//从而保证所有生成器生成的ID是都是顺势递增，且不会重复
func InitializeDistributedWithEtcd(workerID int64, endpoints []string) error {
	if hasInitialized {
		return initErr
	}
	client, err := etcdv3.New(etcdv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 3 * time.Second,
	})
	if err != nil {
		return err
	}
	etcdClient = client
	initializeLock.Do(func() {

		uuid = NewUUID(workerID)

		judgeOtherTimestamps(uuid)

		getOrSaveWorkerID(uuid)

		initErr = fmt.Errorf("UUID has been initialized distributed mode with `InitializeDistributedWithEtcd` called")
		hasInitialized = true
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

func getOrSaveWorkerID(u *UUID) {
	sWorker := strconv.Itoa(int(u.WorkerID))
	key := fmt.Sprintf("%s/%s", UUID_WORKER_ID_KEY, sWorker)

	ctx0, ce0 := context.WithTimeout(context.TODO(), 3* time.Second)
	defer ce0()
	get, err := etcdClient.Get(ctx0, key)
	if err != nil {
		panic(err)
	}
	if get.Count >0 {
		log.Panicf("WokerID has been registered.Please use another one")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	grant, err := etcdClient.Grant(ctx, 5)
	if err != nil {
		panic(err)
	}
	ctx1, cancel1 := context.WithTimeout(context.TODO(), 3*time.Second)
	defer cancel1()
	_, err = etcdClient.Put(ctx1, key, sWorker, etcdv3.WithLease(grant.ID))
	if err != nil {
		panic(err)
	}
	_, err = etcdClient.KeepAlive(context.TODO(), grant.ID)
	if err != nil {
		panic(err)
	}
}

func judgeOtherTimestamps(u *UUID) {
	nts := time.Now().Unix()

	ctx, ce := context.WithTimeout(context.Background(), 3*time.Second)
	defer ce()

	rsp, err := etcdClient.Get(ctx, UUID_TIMESTAMP_KEY, etcdv3.WithPrefix())
	if err != nil {
		panic(err)
	}

	if rsp.Count > 0 {
		var total int64 = 0
		var n int64 = 0
		for _, kv := range rsp.Kvs {
			ts, e := strconv.ParseInt(string(kv.Value), 10, 64)
			if e != nil {
				continue
			}
			total += ts
			n++
		}
		if nts+5 < total/n {
			log.Panicf("local clock is much behind than the world")
		}
	}

	go func() {
		sWorker := strconv.Itoa(int(u.WorkerID))
		key := fmt.Sprintf("%s/%s", UUID_TIMESTAMP_KEY, sWorker)
		for {
			tsgrant, err := etcdClient.Grant(context.TODO(), 3)
			if err != nil {
				panic(err)
			}
			lease := etcdv3.WithLease(tsgrant.ID)
			now := fmt.Sprintf("%d", time.Now().Unix())
			_, err = etcdClient.Put(context.TODO(), key, now, lease)
			if err != nil {
				panic(err)
			}
			time.Sleep(time.Second * 3)
		}
	}()
}
