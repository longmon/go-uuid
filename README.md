# go-uuid
A UUID Generator

基于snowflake方案。

#### 依赖
分布式模式需要etcd服务

单机模式则不需要

#### 使用方法：
```go
import "github.com/longmon/uuid"

//单机模式
uuid.InitializeStandalone()

//使用此模式能保证多个实例生产的UUID是顺势递增的
//workerID: 机器号，每个实例应该不同
uuid.InitializeDistributedWithEtcd(workerID: 1, endpoints:[]string{"localhost:6379"})

//生产一个int64 UUID
id := uuid.generate()

log.Printf("UUID is %v\n", id)

//output: x is 558412312092674
```