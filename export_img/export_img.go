package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"path/filepath"
)

type Data struct {
	Items []Item `xml:"Item"`
}

type Item struct {
	ImageURL string
	ItemName string
	Price    string
	Detail   TechnicalDetail `xml:"TechnicalDetail"`
}

type TechnicalDetail struct {
	ScreenSize    string `xml:"screensize"`
	Certification string `xml:"certification"`
}

func (i Item) String() string {
	return fmt.Sprintf("Item name is %s and image url is %s and technical details is %s & %s ", i.ItemName, i.ImageURL, i.Detail.ScreenSize, i.Detail.Certification)
}

func (d TechnicalDetail) String() string {
	return fmt.Sprintf("Screen Size: %s\r\nCertification: %s\r\n", d.ScreenSize, d.Certification)
}

func ExportImg(filePath string) bool {
	xmlFile, err := os.Open(filePath)
	if err != nil {
		fmt.Println(err.Error())
		return false
	}
	defer xmlFile.Close()
	
	// Get file name of XML File
	fileName := xmlFile.Name()
	
	// Create folder for exporting image
	folderName := strings.TrimSuffix(fileName, ".xml") + string(filepath.Separator)
	err2 := os.MkdirAll(folderName,0777)
	if err2 != nil {
		fmt.Println("err2")
		fmt.Println(err2.Error())
		return false
	}

	b, _ := ioutil.ReadAll(xmlFile)
	var d Data
	xml.Unmarshal(b, &d)

	for _, item := range d.Items {
		response, err := http.Get(item.ImageURL)
		if err != nil {
			fmt.Println(err.Error())
			return false
		}
		defer response.Body.Close()

		//open a file for writing
		imgNameTrim := strings.TrimPrefix(item.ImageURL, "https://")
		imgNameReplace := strings.Replace(imgNameTrim, "/", "_", -1)
		imgName := folderName + imgNameReplace

		file, err := os.Create(imgName)
		if err != nil {
			fmt.Println(err.Error())
			return false
		}
		// Use io.Copy to just dump the response body to the file. This supports huge files
		_, err = io.Copy(file, response.Body)
		if err != nil {
			fmt.Println(err.Error())
			return false
		}
		file.Close()
	}
	
	return true
}

func FindAllFiles(searchDir string) []string {
    fileList := []string{}
    err := filepath.Walk(searchDir, func(path string, f os.FileInfo, err error) error {
		if strings.Compare(filepath.Ext(path), ".xml") == 0 {
			fileList = append(fileList, path)
		}
        return nil
    })
	
	if err != nil {
		fmt.Println(err.Error())
	}
	
	return fileList
}

func main() {
	listFiles := FindAllFiles("./data/xml/alibaba")
    for _, file := range listFiles {
		// Goroutines
        ExportImg(file)
    }
}
