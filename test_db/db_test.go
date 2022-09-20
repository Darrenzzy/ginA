package tests

import (
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/go-xorm/xorm"
)

func connStr(db string) string {
	conn := "test:test@"
	conn += "tcp(127.0.0.1:3306)"
	// %s:%s@tcp(%s:3306)/%s?charset=utf8
	return conn + "/" + db + "?charset=utf8"
}

// var engineGroup
func TestMysql(t *testing.T) {

	engine, err := xorm.NewEngine("mysql", connStr("test"))
	if err != nil {
		t.Error(err)
		return
	}
	defer engine.Close()

	engineGroup, err := xorm.NewEngineGroup(engine, []*xorm.Engine{engine, engine})

	if err != nil {
		t.Error(err)
		return
	}

	engine.ShowSQL(true)
	list := []User{}
	db := engineGroup.Master().Table("article")
	err1 := db.SQL("select * from article").Find(&list)
	t.Log(err1)
	t.Log(list)

	db2 := engineGroup.Slave().Table("article")
	err2 := db2.SQL("select * from article").Find(&list)
	t.Log(err2)
	t.Log(list)

}

// User describes a user
type User struct {
	Id      int       `xorm:"id"`
	Content string    `xorm:"content" sql:"content"`
	Updated time.Time `xorm:"update_by"`
}

func (n *User) TableName() string {
	return "article"
}
