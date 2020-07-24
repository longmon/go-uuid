# go-uuid
A UUID Generator

基于snowflake方案，增加了读取redis服务器时间，如果启动时间落后redis服务器时间太久便会启动失败，以防止服务时间回拨生成重复UUID。

使用方法：
