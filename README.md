# go-uuid
A UUID Generator

基于snowflake方案，增加了读取redis服务器时间，如果启动时间落后redis服务器时间太久便会启动失败，以防止服务时间回拨生成重复UUID。

使用方法：
```go
import "github.com/longmon/uuid"

//单机模式
uuid.InitializeStandalone()

//分布式，使用redis验证时间
//多个实例应该使用时间同步的redis集群或单例
//使用此模式能保证多个实例生产的UUID是顺势递增的
uuid.InitializeDistributedWithRedis("localhost:6379")

//生产一个int64 UUID
id := uuid.generate()

log.Printf("UUID is %v\n", id)
```