package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	pool "github.com/bitleak/go-redis-pool/v2"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis"
	_ "github.com/go-sql-driver/mysql"
	"github.com/go-xorm/xorm"
)

var (
	dbGroup  *xorm.EngineGroup
	pools    *pool.Pool
	mqClient *redis.Client
	mutex    sync.Mutex
)

func init() {
	initRedis()
	initMysql()
	initMq()
}

func initMysql() {
	engine, err := xorm.NewEngine("mysql", fmt.Sprintf("test:test@tcp(127.0.0.1:3306)/%s?charset=utf8", "test"))
	if err != nil {
		log.Fatalln(err)
		return
	}
	engine.ShowSQL(true)
	dbGroup, err = xorm.NewEngineGroup(engine, []*xorm.Engine{engine, engine})
	if err != nil {
		log.Fatalln(err)
		return
	}
	log.Println("mysql 启动完毕")

}

func initRedis() {
	log.SetFlags(log.Llongfile | log.Lshortfile)
	var err error
	pools, err = pool.NewHA(&pool.HAConfig{
		Master: "127.0.0.1:6380",
		Slaves: []string{
			"127.0.0.1:6381",
			"127.0.0.1:6382",
		},
		Password:           "master123", // set master password
		ReadonlyPassword:   "master123", // use password if no set
		PollType:           pool.PollByWeight,
		AutoEjectHost:      true,
		ServerFailureLimit: 1,
		ServerRetryTimeout: 5 * time.Second,
		MinServerNum:       1,
	})

	if err != nil {
		log.Fatal(err, 222)
	}
	log.Println("redis 启动完毕")

}

func initMq() {
	mqClient = redis.NewClient(&redis.Options{
		Addr:     "127.0.0.1:6380",
		Password: "master123", // no password set
		DB:       0,           // use default DB
	})
	pong, err := mqClient.Ping().Result()
	if err == nil && pong != "" {
		log.Println("mq 启动完毕")
	}
}

func main() {

	r := gin.New()
	r.Use(gin.Logger())
	gin.SetMode("release")
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	r.GET("/get", GetArticle)
	r.GET("/list", FindArticles)
	r.POST("/create", CreateArticle)
	r.POST("/edit", EditArticle)
	r.POST("/del", DelArticle)

	go topicHandle()

	var port = flag.Int64("port", 8100, "端口")
	flag.Parse()
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", *port),
		Handler: r,
	}
	// 监听并启动服务
	err := server.ListenAndServe()
	if err != nil {
		log.Println(err)
	}

}

func GetArticle(c *gin.Context) {
	id, _ := c.GetQuery("id")
	idInt, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		log.Println(err)
	}
	if idInt != 0 {
		data := getArticle(idInt)
		c.JSON(200, gin.H{
			"data":    data,
			"message": "ok",
		})
		return
	}

	c.JSON(200, gin.H{
		"data": struct {
		}{},
		"message": "数据不存在",
	})
}

func DelArticle(c *gin.Context) {
	data := Article{}
	err := c.BindJSON(&data)
	if err != nil {
		log.Println(err)
	}
	if data.Id > 0 {
		row, err := dbGroup.Where("id=?", data.Id).Delete(&data)
		log.Println(row, err)
	}

	c.JSON(200, gin.H{
		"message": "ok",
	})
}

func EditArticle(c *gin.Context) {

	data := Article{}
	err := c.BindJSON(&data)
	if err != nil {
		log.Println(err)
	}

	oldData := getArticle(data.Id)
	var needUpdata bool
	if data.Content != oldData.Content {
		oldData.Content = data.Content
		needUpdata = true
	}

	if data.Email != oldData.Email {
		oldData.Email = data.Email
		needUpdata = true
	}

	if data.Author != oldData.Author {
		oldData.Author = data.Author
		needUpdata = true

	}
	message := "ok"
	if needUpdata {
		bs, _ := json.Marshal(data)
		mqClient.Publish("topic", string(bs))
	}
	c.JSON(200, gin.H{
		"message": message,
	})
}

func CreateArticle(c *gin.Context) {
	data := Article{}
	err := c.BindJSON(&data)
	if err != nil {
		log.Println(err)
	}
	createArticle(&data)
	c.JSON(200, gin.H{
		"message": "ok",
	})
}

func FindArticles(c *gin.Context) {
	id, _ := c.GetQuery("id")
	lastId, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		log.Println(err)
	}
	list := make([]*Article, 0, 10)
	err = dbGroup.Slave().Where("id > ?", lastId).Limit(10).Find(&list)
	if err != nil {
		log.Println(err)
	}

	c.JSON(200, gin.H{
		"data": list,
	})
}

// ——————————————————————————————————————————————model————————————————————————————————————————————————————————————————————————————————

// Article describes a arc
type Article struct {
	Id        int64     `xorm:"id" json:"id"`
	Content   string    `xorm:"content" sql:"content" json:"content"`
	Email     string    `xorm:"email" sql:"email" json:"email"`
	Author    string    `xorm:"author" sql:"author" json:"author"`
	Updated   time.Time `xorm:"update_by" json:"updated"`
	DeletedAt time.Time `xorm:"deleted" json:"_"`
}

func (n *Article) TableName() string {
	return "article"
}

// —————————————————————————————————————————————design——func———————————————————————————————————————————————————————————————————————————

func getArticle(id int64) (data *Article) {
	data = &Article{}
	ctx := context.Background()
	key := fmt.Sprintf("arts_%d", id)
	art := pools.Get(ctx, key).Val()
	if art != "" {
		err := json.Unmarshal([]byte(art), data)
		if err != nil {
			log.Println(err, 22)
		}
		return data
	}

	_, err := dbGroup.Slave().Where("id=?", id).Get(data)
	if err != nil {
		log.Println(err, 11)
	}
	bs, _ := json.Marshal(data)
	pools.Set(ctx, key, string(bs), time.Hour*24*3).Val()
	return data

}
func editArticle(data *Article) error {
	ctx := context.Background()
	key := fmt.Sprintf("arts_%d", data.Id)

	_, err := dbGroup.Master().Where("id=?", data.Id).Update(&Article{
		Content: data.Content,
		Email:   data.Email,
		Author:  data.Author,
	})
	if err != nil {
		log.Println(err)
	}
	bs, _ := json.Marshal(data)
	pools.Set(ctx, key, string(bs), time.Hour*24*3).Val()
	return err

}

func createArticle(data *Article) {
	bs, _ := json.Marshal(data)
	mqClient.Publish("topic", string(bs))
}

func articleToDb(art string) {
	data := &Article{}
	err := json.Unmarshal([]byte(art), data)
	if err != nil {
		log.Println(err)
	}
	isLock := lock(fmt.Sprintf("art_%d", data.Id))
	defer unLock(fmt.Sprintf("art_%d", data.Id))
	if isLock {
		if data.Id > 0 {
			err := editArticle(data)
			if err != nil {
				log.Println("失败更新db  id", err)
				return
			}
			log.Printf("成功更新db  id：%d", data.Id)
			return
		}
		row, err := dbGroup.Master().InsertOne(data)
		if err != nil {
			log.Println(err)
		}
		log.Printf("成功写入db：%d条", row)
	}

}

func topicHandle() {
	defer func() {
		if r := recover(); r != nil {
			log.Println("topicHandle Recovered in f", r)
		}
	}()

	key := fmt.Sprintf("liten_%d_%d", time.Now().Day(), time.Now().Minute()/10)
	ident, _ := pools.Get(context.TODO(), key).Int()
	pools.SetNX(context.TODO(), key, 1, time.Minute)
	if ident == 0 {
		println("启动消费mq成功")
		pubsub := mqClient.PSubscribe("topic")
		defer pubsub.Close()
		for msg := range pubsub.Channel() {
			log.Printf("channel=%s message=%s\n", msg.Channel, msg.Payload)
			articleToDb(msg.Payload)
		}
	}

}

func lock(key string) bool {
	var ctx = context.Background()
	mutex.Lock()
	defer mutex.Unlock()
	bools, err := pools.SetNX(ctx, key, 1, 10*time.Second).Result()
	if err != nil {
		log.Println(err.Error())
	}
	return bools
}
func unLock(key string) int64 {
	var ctx = context.Background()
	nums, err := pools.Del(ctx, key)
	if err != nil {
		log.Println(err.Error())
		return 0
	}
	return nums
}
