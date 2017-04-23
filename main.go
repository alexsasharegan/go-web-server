package main

import (
	"database/sql"
	"encoding/json"
	"encoding/xml"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"html/template"
	"io/ioutil"
	"net/http"
	"net/url"
)

type Page struct {
	Name     string
	DbStatus bool
}

type SearchResult struct {
	Title  string `xml:"title,attr"`
	Author string `xml:"author,attr"`
	Year   string `xml:"hyr,attr"`
	ID     string `xml:"owi,attr"`
}

type ClassifyBookResponse struct {
	BookData struct {
		Title  string `xml:"title,attr"`
		Author string `xml:"author,attr"`
		ID     string `xml:"owi,attr"`
	} `xml:"work"`
	Classification struct {
		MostPopular string `xml:"sfa,attr"`
	} `xml:"recommendations>ddc>mostPopular"`
}

const classifyAPIBase = "http://classify.oclc.org/classify2/Classify"

func findBook(id string) (ClassifyBookResponse, error) {
	var cRes ClassifyBookResponse
	body, err := classifyAPI(classifyAPIBase + "?summary=tre&owi=" + url.QueryEscape(id))

	if errorIsPresent(err) {
		return ClassifyBookResponse{}, err
	}

	err = xml.Unmarshal(body, &cRes)

	return cRes, err
}

func main() {
	templates := template.Must(template.ParseFiles("templates/index.html"))

	db, _ := sql.Open("sqlite3", "dev.db")

	http.HandleFunc("/", func(res http.ResponseWriter, req *http.Request) {
		p := Page{Name: "Gopher"}
		name := req.FormValue("name")

		if name != "" {
			p.Name = name
		}

		p.DbStatus = db.Ping() == nil

		err := templates.ExecuteTemplate(res, "index.html", p)

		if errorIsPresent(err) {
			http.Error(res, err.Error(), http.StatusInternalServerError)
		}
	})

	http.HandleFunc("/search", func(res http.ResponseWriter, req *http.Request) {
		var results []SearchResult
		var err error

		results, err = search(req.FormValue("search"))

		if errorIsPresent(err) {
			http.Error(res, err.Error(), http.StatusInternalServerError)
		}

		encoder := json.NewEncoder(res)
		err = encoder.Encode(results)

		if errorIsPresent(err) {
			http.Error(res, err.Error(), http.StatusInternalServerError)
		}
	})

	http.HandleFunc("/books/add", func(res http.ResponseWriter, req *http.Request) {
		var book ClassifyBookResponse
		var err error

		book, err = findBook(req.FormValue("id"))

		if errorIsPresent(err) {
			http.Error(res, err.Error(), http.StatusInternalServerError)
		}

		err = db.Ping()

		if errorIsPresent(err) {
			http.Error(res, err.Error(), http.StatusInternalServerError)
		}

		_, err = db.Exec("INSERT INTO books (pk, title, author, id, classification) VALUES (?,?,?,?,?)",
			nil, book.BookData.Title, book.BookData.Author, book.BookData.ID, book.Classification.MostPopular)

		if errorIsPresent(err) {
			http.Error(res, err.Error(), http.StatusInternalServerError)
		}
	})

	bindListenErr := http.ListenAndServe(":8080", nil)

	if errorIsPresent(bindListenErr) {
		fmt.Println(bindListenErr.Error())
	} else {
		fmt.Println("Listening on port 8080")
	}

}

type ClassifySearchResponse struct {
	Results []SearchResult `xml:"works>work"`
}

func search(query string) ([]SearchResult, error) {
	res, err := classifyAPI(classifyAPIBase + "?summary=true&title=" + url.QueryEscape(query))

	if errorIsPresent(err) {
		return []SearchResult{}, err
	}

	var cRes ClassifySearchResponse
	err = xml.Unmarshal(res, &cRes)

	return cRes.Results, err
}

func classifyAPI(url string) ([]byte, error) {
	res, err := http.Get(url)

	if errorIsPresent(err) {
		return []byte{}, err
	}

	defer res.Body.Close()
	var body []byte
	body, err = ioutil.ReadAll(res.Body)

	if errorIsPresent(err) {
		return []byte{}, err
	}

	return body, nil
}

func errorIsNil(err error) bool {
	return err == nil
}

func errorIsPresent(err error) bool {
	return !errorIsNil(err)
}
