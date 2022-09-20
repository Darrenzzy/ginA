package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
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
)

func init() {
	initRedis()
	initMysql()
	initMq()
}

func connStr(db string) string {
	conn := "test:test@"
	conn += "tcp(127.0.0.1:3306)"
	// %s:%s@tcp(%s:3306)/%s?charset=utf8
	return conn + "/" + db + "?charset=utf8"
}

func initMysql() {
	engine, err := xorm.NewEngine("mysql", connStr("test"))
	if err != nil {
		log.Fatalln(err)
		return
	}
	// defer engine.Close()
	engine.ShowSQL(true)
	dbGroup, err = xorm.NewEngineGroup(engine, []*xorm.Engine{engine, engine})
	if err != nil {
		log.Fatalln(err)
		return
	}
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

}

func initMq() {
	mqClient = redis.NewClient(&redis.Options{
		Addr:     "127.0.0.1:6380",
		Password: "master123", // no password set
		DB:       0,           // use default DB
	})
	pong, err := mqClient.Ping().Result()
	fmt.Println(pong, err)
}

func main() {
	r := gin.Default()
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
	server := &http.Server{
		Addr:    ":8100",
		Handler: r,
	}
	// 监听并启动服务
	err := server.ListenAndServe()
	if err != nil {
		fmt.Println(err)
	}

}

func GetArticle(c *gin.Context) {
	id, _ := c.GetQuery("id")
	idInt, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		fmt.Println(err)
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

	ctx := context.Background()

	s := pools.Set(ctx, "name", "barry", time.Second*60).String()
	fmt.Println(s)

	data := &Article{}
	row, err := dbGroup.Where("id=2").Delete(data)
	log.Println(row, err)

	c.JSON(200, gin.H{
		"message": "pong",
	})
}

func EditArticle(c *gin.Context) {

	data := Article{}
	err := c.BindJSON(&data)
	if err != nil {
		fmt.Println(err)
	}

	oldData := getArticle(data.Id)
	var needUpdata bool
	if data.Content != "" {
		oldData.Content = data.Content
		needUpdata = true
	}

	if data.Email != "" {
		oldData.Email = data.Email
		needUpdata = true

	}

	if data.Author != "" {
		oldData.Author = data.Author
		needUpdata = true

	}
	message := "ok"
	if needUpdata {
		err := editArticle(oldData)
		if err != nil {
			message = err.Error()
		}
	}
	// data:=getArticle(data.Id)
	c.JSON(200, gin.H{
		"message": message,
	})
}

func CreateArticle(c *gin.Context) {

	data := Article{}
	err := c.BindJSON(&data)
	if err != nil {
		fmt.Println(err)
	}
	createArticle(&data)
	c.JSON(200, gin.H{
		"message": "ok",
	})
}

func FindArticles(c *gin.Context) {
	id, _ := c.GetQuery("id")
	lastId, err := strconv.ParseInt(id, 10, 64)
	list := []*Article{}
	err = dbGroup.Slave().Where("id > ?", lastId).Limit(2).Find(&list)
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
	DeletedAt time.Time `xorm:"deleted"`
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
			fmt.Println(err, 22)
		}
		return data
	}

	_, err := dbGroup.Slave().Where("id=?", id).Get(data)
	if err != nil {
		fmt.Println(err, 11)
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
		fmt.Println(err)
	}
	bs, _ := json.Marshal(data)
	pools.Set(ctx, key, string(bs), time.Hour*24*3).Val()
	return err

}

func createArticle(data *Article) {
	bs, _ := json.Marshal(data)
	mqClient.Publish("topic", string(bs))
}

func createArticleToDb(art string) {
	data := &Article{}
	err := json.Unmarshal([]byte(art), data)
	if err != nil {
		fmt.Println(err)
	}
	row, err := dbGroup.Master().InsertOne(data)
	if err != nil {
		fmt.Println(err)
	}
	log.Printf("成功写入db：%d条", row)
}

func topicHandle() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("topicHandle Recovered in f", r)
		}
	}()
	println(4444)

	pubsub := mqClient.PSubscribe("topic")
	defer pubsub.Close()
	for msg := range pubsub.Channel() {
		fmt.Printf("channel=%s message=%s\n", msg.Channel, msg.Payload)
		createArticleToDb(msg.Payload)
	}

}
