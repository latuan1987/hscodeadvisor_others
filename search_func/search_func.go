package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/lib/pq"
)

const (
	DB_USER     = "postgres"
	DB_PASSWORD = "tuandino"
	DB_NAME     = "postgres"
)

type hsdata struct {
	ID       int    `json:"id"`
	HSCODE   string `json:"hscode"`
	PRODNAME string `json:"productname"`
	PRODDESC string `json:"productdesc"`
}

func main() {
	dbinfo := fmt.Sprintf("user=%s password=%s dbname=%s sslmode=disable",
		DB_USER, DB_PASSWORD, DB_NAME)
	db, err := sql.Open("postgres", dbinfo)
	checkErr(err)
	defer db.Close()

	fmt.Println("# Querying")
	rows, err := db.Query("SELECT * FROM hscode WHERE id=100")
	checkErr(err)

	for rows.Next() {
		var id int
		var hscode string
		var productname string
		var productdesc string
		err = rows.Scan(&id, &hscode, &productname, &productdesc)
		checkErr(err)

		resHs := &hsdata{
			ID:       id,
			HSCODE:   hscode,
			PRODNAME: productname,
			PRODDESC: productdesc}

		res, _ := json.Marshal(resHs)
		fmt.Println(string(res))
	}
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}
