package main

import (
	"encoding/json"
	"fmt"

	"github.com/gomodule/redigo/redis"
	"github.com/nitishm/go-rejson"
)

type DB struct {
	Conn        redis.Conn
	JsonHandler *rejson.Handler
}

// Name - student name
type Name struct {
	First  string `json:"first,omitempty"`
	Middle string `json:"middle,omitempty"`
	Last   string `json:"last,omitempty"`
}

// Student - student object
type Student struct {
	Name Name `json:"name,omitempty"`
	Rank int  `json:"rank,omitempty"`
}

func (db *DB) AddJsonStruct(key string, value interface{}) (res interface{}, err error) {
	res, err = db.JsonHandler.JSONSet(key, ".", value)
	if err != nil {
		fmt.Println(err)
	}
	return res, err
}

func (db *DB) GetJsonStruct(key string) (json []byte, err error) {
	res, err := db.JsonHandler.JSONGet(key, ".")
	if err != nil {
		fmt.Println(err)
	}
	json, err = redis.Bytes(res, err)
	if err != nil {
		fmt.Println(err)
	}
	return json, err
}

func ConnectDB(addr string) DB {
	conn, err := redis.Dial("tcp", addr)
	if err != nil {
		panic("failed to connect to the Redis server")
	}

	rh := rejson.NewReJSONHandler()
	rh.SetRedigoClient(conn)

	db := DB{
		Conn:        conn,
		JsonHandler: rh,
	}

	return db
}

func main() {
	var addr = "localhost:6379"
	db := ConnectDB(addr)

	defer db.Conn.Close()

	student := Student{
		Name: Name{
			"Fart",
			"S",
			"Pronto",
		},
		Rank: 2,
	}

	res, err := db.AddJsonStruct("student", student)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(res)

	resJson, err := db.GetJsonStruct("student")
	if err != nil {
		fmt.Println(err)
	}

	stu := Student{}
	err = json.Unmarshal(resJson, &stu)

	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(stu.Rank)

}
