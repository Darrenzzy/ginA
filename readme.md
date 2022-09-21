

# 服务设计


### 目标
1. 使用gin框架开发文章管理服务的前端接口，通过micro客户端调用后端文章管理服务。
2. 要求：日活1000万，峰值QPS：20000


### 设计思路

1. 网关：nginx作为入口，负载均衡后流量分发到多个服务节点上

2. go服务：启动三个节点，分别承担5000-8000 的QPS

3. 缓存：redis作为缓存，通过一主两从达到读写分离，
    根据单节点4GB规格redis可以抗16WQPS， 当前读写分离有效降低对写库的压力  [阿里云测试](https://help.aliyun.com/document_detail/100453.html)

4. 数据库：mysql作为数据落地，通过2个数据库做到主从读写分离，
    主写库负责数据更新，从库负责数据查询 , 根据阿里云mysql 2核4 GB可以单节点可以抗1.2w写入并发 [阿里云测试](https://help.aliyun.com/document_detail/53638.htm?spm=a2c4g.11186623.0.0.738865d7fwc33o#concept-8031) 



### 重点设计方案

1. 削峰：创建文章和更新文章 都是用mq消息来通过发布/订阅消费方式,降低对服务本身请求积压影响，和响应速度慢的影响。
 
2. 多场景读写分离，降低对写入数据的压力，更独立读取数据。

3. 服务启动多节点 承担并发压力。

4. 查询DB，添加索引，保证条件语句命中索引。这里对 (创建时间,发布内容，主键id) 做联合索引。

5. 分布式并发锁，保证DB数据原子性操作。

### 启动流程

```bash

# 网关
1. nginx -s reload
# 缓存
2. cd /Users/darren/go/src/person-go/docker-practise/redis-cluster && docker-compose up
# 数据库
3. cd /Users/darren/go/src/person-go/docker/mysql && docker-compose up
# 程序服务
4. cd /Users/darren/go/src/ginA && ./ginA --port=8100 && ./ginA --port=8200 && ./ginA --port=8300


```

### 由于时间问题待完成项的思考

1. 对于数据参数的校验 
2. 对于通过中间件 对脏数据的识别 
3. 
  