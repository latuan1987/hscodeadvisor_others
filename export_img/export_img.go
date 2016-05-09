package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
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

func main() {
	xmlFile, err := os.Open("C:/Go/GOWorkspace/src/hscodeweb/export_img/data/xml/alibaba/Items_20_04_2016.xml")
	if err != nil {
		fmt.Println(err.Error())
	}
	defer xmlFile.Close()

	b, _ := ioutil.ReadAll(xmlFile)
	var d Data
	xml.Unmarshal(b, &d)

	for _, item := range d.Items {
		response, err := http.Get(item.ImageURL)
		if err != nil {
			fmt.Println(err.Error())
		}
		defer response.Body.Close()

		//open a file for writing
		fileNameTrim := strings.TrimPrefix(item.ImageURL, "https://")
		fileNameReplace := strings.Replace(fileNameTrim, "/", "_", -1)
		fileName := fmt.Sprintf("C:/Go/GOWorkspace/src/hscodeweb/export_img/data/export_img/%s", fileNameReplace)

		file, err := os.Create(fileName)
		if err != nil {
			fmt.Println(err.Error())
		}
		// Use io.Copy to just dump the response body to the file. This supports huge files
		_, err = io.Copy(file, response.Body)
		if err != nil {
			fmt.Println(err.Error())
		}
		file.Close()
	}
}
