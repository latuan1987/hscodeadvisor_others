package main

import (
	"encoding/xml"
	"database/sql"
	//"encoding/json"
	"io/ioutil"
	"os"
	"fmt"
	"path/filepath"
	"strings"
	_ "github.com/lib/pq"
)

const (
	DB_USER     = "postgres"
	DB_PASSWORD = "tuandino"
	DB_NAME     = "postgres"
)

type jsonProdItem struct {
	ID       int64    `json:"id"`
	PRODNAME string `json:"productname"`
	HSCODE   string `json:"hscode"`
	PRODDESC string `json:"productdesc"`
}

type Data struct {
	ProductGroups []ProductGroup `xml:"productGroup"`

}

type ProductGroup struct {
	Products []Product `xml:"product"`
	ProductGroupName string `xml:"name,attr"`
}

type Product struct {
	HsCode string `xml:"hsCode"`
	Desc string `xml:"productDesc"`
}

func xmlParse(filePath string) []ProductGroup {
	xmlFile, err := os.Open(filePath)
	checkErr(err)
	defer xmlFile.Close()

	b, _ := ioutil.ReadAll(xmlFile)
	var d Data
	xml.Unmarshal(b, &d)

	return d.ProductGroups
}

func findAllFiles(searchDir string) []string {
    fileList := []string{}
    err := filepath.Walk(searchDir, func(path string, f os.FileInfo, err error) error {
		if strings.Compare(filepath.Ext(path), ".xml") == 0 {
			fileList = append(fileList, path)
		}
        return nil
    })
	checkErr(err)
	
	return fileList
}
/*
func dbSearchFunc(query string) {
	fmt.Println("# Querying")
	rows, err := db.Query("SELECT * FROM hscodeadvisor WHRER productname=$1", query)
	checkErr(err)

	for rows.Next() {
		var id int64
		var hscode string
		var productname string
		var productdesc string
		err = rows.Scan(&id, &productname, &hscode, &productdesc)
		checkErr(err)

		resHs := jsonProdItem{
			ID:       id,
			PRODNAME: productname,
			HSCODE:   hscode,
			PRODDESC: productdesc}

		res, _ := json.Marshal(resHs)
		fmt.Println(string(res))
	}
}

func dbProdTableInit( list []ProductGroup ) bool {
	fmt.Println("# Create table")
    _ ,err := db.Exec("CREATE TABLE IF NOT EXISTS hscodeadvisor (id integer PRIMARY KEY NOT NULL, productname character varying(100) NOT NULL, hscode character varying(100) NOT NULL, productdesc character varying(500) NOT NULL, created date)")
	checkErr(err)
	if err != nil {
		return false
	}
	
	fmt.Println("# Create table success")
	insStmt, err := db.Prepare("INSERT INTO hscodeadvisor(productname, hscode, productdesc) VALUES ($1, $2, $3)")
	checkErr(err)
	if err != nil {
		return false
	}

	fmt.Println("# Insert")
	for _, prodGrpItem := range list {
		for _, product := range prodGrpItem.Products {
			// Insert to table
			_, err := insStmt.Exec(prodGrpItem.ProductGroupName, product.HsCode, product.Desc)
			checkErr(err)
		}
	}

	return true
}
*/
func main() {
	// Connect to database
	fmt.Println("# Connect")
	dbinfo := fmt.Sprintf("user=%s password=%s dbname=%s sslmode=disable",
		DB_USER, DB_PASSWORD, DB_NAME)
	db, err := sql.Open("postgres", dbinfo)
	checkErr(err)
	defer db.Close()
	fmt.Println("# Connect success")

	// Get all xml file in specified folder
	var prodGrpList []ProductGroup;
	listFiles := findAllFiles("./data")
	fmt.Println(listFiles)
    for _, file := range listFiles {
		// Parsing xml
        prodGrpList = append(prodGrpList, xmlParse(file)...)
    }
	fmt.Println("# find file success")
	
	// Init table and insert value
	fmt.Println("# Create table")
    _,err2 := db.Exec("CREATE TABLE IF NOT EXISTS hscodeadvisor (id SERIAL PRIMARY KEY NOT NULL, productname character varying(100) NOT NULL, hscode character varying(100) NOT NULL, productdesc character varying(500) NOT NULL, created timestamp DEFAULT CURRENT_TIMESTAMP)")
	checkErr(err2)
	fmt.Println("# Create table success")

	insStmt, err := db.Prepare("INSERT INTO hscodeadvisor(productname, hscode, productdesc) VALUES ($1, $2, $3)")
	checkErr(err)

	fmt.Println("# Insert")
	for _, prodGrpItem := range prodGrpList {
		for _, product := range prodGrpItem.Products {
			// Insert to table
			_, err := insStmt.Exec(strings.TrimPrefix(prodGrpItem.ProductGroupName, " - Import Data"), product.HsCode, product.Desc)
			checkErr(err)
		}
	}
}

func checkErr(err error) {
	if err != nil {
		fmt.Println(err.Error())
	}
}
