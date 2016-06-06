package main

import (
	"database/sql"
	"encoding/xml"
	"flag"
	"fmt"
	"github.com/blevesearch/bleve"
	bleveHttp "github.com/blevesearch/bleve/http"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var db *sql.DB = nil
var xmlDir = flag.String("xmlDir", "data/", "xml directory")
var indexPath = flag.String("index", "hscode-search.bleve", "index path")
var indexName = flag.String("indexName", "ProductData", "index name")

type DataInfo struct {
	ID         int64     `json:"id,omitempty"`
	DATE       time.Time `json:"Date,omitempty"`
	CATEGORY   string    `json:"Category,omitempty"`
	PRODDESC   string    `json:"ProductDescription,omitempty"`
	PICTURE    string    `json:"Picture,omitempty"`
	HSCODE     string    `json:"WCOHSCode,omitempty"`
	COUNTRY    string    `json:"Country,omitempty"`
	TARIFFCODE string    `json:"NationalTariffCode,omitempty"`
	EXPLAIN    string    `json:"ExplanationSheet,omitempty"`
	VOTE       string    `json:"Vote,omitempty"`
}

type ImportData struct {
	ProductGroups []ProductGroup `xml:"productGroup,omitempty"` // Viet Name Trade
	Items         []Item         `xml:"Item,omitempty"`         // Alibaba
}

type ProductGroup struct {
	ProductGroupName string    `xml:"name,attr"`
	Products         []Product `xml:"product,omitempty"`
}

type Product struct {
	HsCode string `xml:"hsCode,omitempty"`
	Desc   string `xml:"productDesc,omitempty"`
}

type Item struct {
	ImageURL string          `xml:"ImageURL,omitempty"`
	ItemName string          `xml:"ItemName,omitempty"`
	FOBPrice string          `xml:"FOBPrice,omitempty"`
	Detail   TechnicalDetail `xml:"TechnicalDetail,omitempty"`
}

type TechnicalDetail struct {
	ScreenSize    string `xml:"screensize"`
	Certification string `xml:"certification"`
}

func xmlParse(filePath string) ImportData {
	xmlFile, err := os.Open(filePath)
	if err != nil {
		log.Fatalf("Error xmlParse: %q", err)
	}
	defer xmlFile.Close()

	b, _ := ioutil.ReadAll(xmlFile)
	var d ImportData
	xml.Unmarshal(b, &d)

	return d
}

func findAllFiles(searchDir string) []string {
	fileList := []string{}
	err := filepath.Walk(searchDir, func(path string, f os.FileInfo, err error) error {
		if (strings.Compare(filepath.Ext(path), ".xml") == 0) && (strings.Contains(path, "_done") == false) {
			fileList = append(fileList, path)
		}
		return nil
	})
	if err != nil {
		log.Fatalf("Error findAllFiles: %q", err)
	}

	return fileList
}

func indexData(db *sql.DB, i bleve.Index) error {

	// Get all xml file in specified folder
	var importDataList []ImportData
	listFiles := findAllFiles(*xmlDir)
	for _, file := range listFiles {
		// Parsing xml
		importDataList = append(importDataList, xmlParse(file))

		// Rename file
		extension := filepath.Ext(file)
		basename := file[0 : len(file)-len(extension)]
		os.Rename(file, basename+"_done.xml")
	}

	// Insert data to table and make indexing
	for _, importDataItem := range importDataList {
		// Viet Name Trade data
		for _, productGroups := range importDataItem.ProductGroups {
			for _, productItem := range productGroups.Products {
				// Make data info
				dataInfo := DataInfo{
					CATEGORY: productGroups.ProductGroupName,
					HSCODE:   productItem.HsCode,
					PRODDESC: productItem.Desc}

				// Insert to data base
				var lastID int64
				if err := db.QueryRow("INSERT INTO Products (Category, ProductDescription, WCOHSCode) VALUES ($1,$2,$3) RETURNING ID", productGroups.ProductGroupName, productItem.Desc, productItem.HsCode).Scan(&lastID); err != nil {
					log.Fatal(err)
					return err
				}

				// Index
				if err := i.Index(string(lastID), dataInfo); err != nil {
					log.Fatal(err)
					return err
				}
			}
		}
		// Alibaba data
		for _, item := range importDataItem.Items {
			// Make data info
			dataInfo := DataInfo{
				PRODDESC: item.ItemName,
				PICTURE:  item.ImageURL}

			// Insert to data base
			var lastID int64
			if err := db.QueryRow("INSERT INTO Products (ProductDescription, Picture) VALUES ($1,$2) RETURNING ID", item.ItemName, item.ImageURL).Scan(&lastID); err != nil {
				log.Fatal(err)
				return err
			}

			// Index
			if err := i.Index(string(lastID), dataInfo); err != nil {
				log.Fatal(err)
				return err
			}
		}
	}

	return nil
}

func searchIndex(c *gin.Context) {
	// Get passed parameter
	queryString := c.Query("query") // shortcut for c.Request.URL.Query().Get("query")

	// Get index
	index := bleveHttp.IndexByName(*indexName)
	if index == nil {
		log.Printf("index null!!!")
		return
	}

	// Query data
	// We are looking to an product data with some string which match with dotGo
	query := bleve.NewMatchQuery(queryString)
	searchRequest := bleve.NewSearchRequest(query)
	searchResult, err := index.Search(searchRequest)
	if err != nil {
		c.String(http.StatusInternalServerError, fmt.Sprintf("Search: %q", err))
		return
	}

	if searchResult.Total < 0 {
		c.JSON(http.StatusOK, fmt.Sprintf("No data found!"))
		return
	}

	// Output data
	for _, hit := range searchResult.Hits {
		// Write JSON data to response body
		c.JSON(http.StatusOK, &hit)
		if id, err := strconv.ParseInt(hit.ID, 10, 64); err == nil {
			// Query data
			rows, err := db.Query("SELECT * FROM Products WHERE ID=$1", id)
			if err != nil {
				c.String(http.StatusInternalServerError, fmt.Sprintf("Query data: %q", err))
				return
			}

			for rows.Next() {
				var id int64
				var date time.Time
				var category string
				var proddesc string
				var picture string
				var hscode string
				var country string
				var tariffcode string
				var explain string
				var vote string

				if rows.Scan(&id, &date, &category, &proddesc, &picture, &hscode, &country, &tariffcode, &explain, &vote); err != nil {
					c.String(http.StatusInternalServerError, fmt.Sprintf("Error scanning: %q", err))
				} else {
					resHs := &DataInfo{
						ID:         id,
						DATE:       date,
						CATEGORY:   category,
						PRODDESC:   proddesc,
						PICTURE:    picture,
						HSCODE:     hscode,
						COUNTRY:    country,
						TARIFFCODE: tariffcode,
						EXPLAIN:    explain,
						VOTE:       vote}

					// Write JSON data to response body
					c.JSON(http.StatusOK, resHs)
				}
			}
			// Close
			rows.Close()
		}
	}
}

func main() {
	port := os.Getenv("PORT")

	if port == "" {
		log.Fatal("$PORT must be set")
	}

	flag.Parse()

	db, err := sql.Open("postgres", "postgres://oypadbnkrpjfds:VbrZNoYFYbbP-xz3G8CqNeKHGd@ec2-54-225-244-221.compute-1.amazonaws.com:5432/d7ipnov3t204cm")
	if err != nil {
		log.Fatalf("Error opening database: %q", err)
	}

	var CREATE_TABLE string = "CREATE TABLE IF NOT EXISTS Products (ID SERIAL PRIMARY KEY NOT NULL, Date timestamp DEFAULT CURRENT_TIMESTAMP, Category text, ProductDescription text, Picture text, WCOHSCode integer, Country text, NationalTariffCode integer, ExplanationSheet text, Vote text)"

	if _, err := db.Exec(CREATE_TABLE); err != nil {
		panic(err)
	}

	// open the index
	dataIndex, err := bleve.Open(*indexPath)
	if err == bleve.ErrorIndexPathDoesNotExist {
		log.Printf("Creating new index...")
		// create a new mapping file and create a new index
		indexMapping := bleve.NewIndexMapping()
		dataIndex, err = bleve.New(*indexPath, indexMapping)
		if err != nil {
			log.Fatal(err)
		}

		// index data in the background
		go func() {
			err = indexData(db, dataIndex)
			if err != nil {
				log.Fatal(err)
			}
		}()
	} else if err != nil {
		log.Fatal(err)
	} else {
		log.Printf("Opening existing index...")
	}

	router := gin.Default()

	// add the API
	bleveHttp.RegisterIndexName(*indexName, dataIndex)

	// Query string parameters are parsed using the existing underlying request object.
	// The request responds to a url matching:  /search?query=computer
	router.GET("/search", searchIndex)

	router.Run(":" + port)
}
