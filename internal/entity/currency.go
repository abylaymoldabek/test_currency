package entity

import "encoding/xml"

type NBKData struct {
	XMLName    xml.Name   `xml:"rates"`
	Date       string     `xml:"date"`
	Currencies []Currency `xml:"item"`
}

type Currency struct {
	XMLName xml.Name `xml:"item" json:"-"`
	Name    string   `xml:"fullname"`
	Code    string   `xml:"title"`
	Rate    float64  `xml:"description"`
	Date    string
}
