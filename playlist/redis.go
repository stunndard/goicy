package playlist

import (
	"fmt"

	"github.com/gomodule/redigo/redis"
	"github.com/nitishm/go-rejson"
)

type DB struct {
	Conn        redis.Conn
	JsonHandler *rejson.Handler
}

// SET One Object
func (db *DB) AddJsonStruct(key string, value interface{}) (res interface{}, err error) {
	res, err = db.JsonHandler.JSONSet(key, ".", value)
	if err != nil {
		fmt.Println(err)
	}
	return res, err
}

// GET One Object
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

// Update a value with json path
// example `JSON.SET foo .bar.baz `"{\"thing\": \"false\"}"`
func (db *DB) UpdateJsonStruct(key string, path string, value interface{}) (res interface{}, err error) {
	res, err = db.JsonHandler.JSONSet(key, path, value)
	if err != nil {
		fmt.Println(err)
	}
	return res, err
}

// Get value at path
func (db *DB) GetJsonStructPath(key string, path string) (res interface{}, err error) {
	res, err = db.JsonHandler.JSONGet(key, path)
	if err != nil {
		fmt.Println(err)
	}
	json, err := redis.Bytes(res, err)
	if err != nil {
		fmt.Println(err)
	}
	return json, err
}

// Constructor
func ConnectDB(addr string) (db DB, err error) {
	conn, err := redis.Dial("tcp", addr)
	if err != nil {
		fmt.Println("error connecting to db")
	}

	rh := rejson.NewReJSONHandler()
	rh.SetRedigoClient(conn)

	db = DB{
		Conn:        conn,
		JsonHandler: rh,
	}

	return db, err
}
