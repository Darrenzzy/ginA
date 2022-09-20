package test

import (
	"context"
	"fmt"
	"log"
	"testing"
	"time"

	pool "github.com/bitleak/go-redis-pool/v2"
)

// var clusterClient2 *redis.ClusterClient
var pools *pool.Pool

var ctx = context.Background()

func init() {
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
	log.Println(pools.Set(ctx, "foo", "bar", 0))
	log.Println(pools.Get(ctx, "foo"))
	log.Println(pools.Get(ctx, "www"))

}

// 验证上面是否拿到数据
func TestGetKey(t *testing.T) {

	if pools == nil {
		return
	}
	for i := 0; i < 10; i++ {
		fmt.Println(pools.Incr(ctx, "nums").Val())
	}
	fmt.Println(pools.Get(ctx, "nums"))

}
