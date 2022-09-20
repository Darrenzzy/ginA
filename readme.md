

# 服务设计


### 目标
* 使用gin框架开发文章管理服务的前端接口，通过micro客户端调用后端文章管理服务。
* 要求：日活1000万，峰值QPS：20000


### 设计思路

1. 网关：nginx作为入口，负载均衡后流量分发到多个服务节点上
2. go服务：启动三个节点，分别承担5000-8000 的QPS
3. 缓存：redis作为缓存，通过一主两从达到读写分离，
    单节点4GB规格redis可以抗16WQPS， 当前读写分离有效降低对写库的压力
4. 数据库：mysql作为数据落地，通过2个数据库做到主从读写分离，
    主写库负责数据更新，从库负责数据查询


### 重点设计方案

1. 削峰：创建文章和更新文章 都是用mq消息来通过发布/订阅消费方式,降低对服务本身请求积压影响，和响应速度慢影响。
 
2. 多场景读写分离，降低对写入数据的压力，更独立读取数据

3. 