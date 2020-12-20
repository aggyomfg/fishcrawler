package main

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/geziyor/geziyor"
	"github.com/geziyor/geziyor/client"
	"github.com/iotdog/json2table/j2t"
	"golang.org/x/text/encoding/charmap"
)

var (
	searchQuery  string
	emailsTotal  string
	searchResult []fishSearchCard
)

type fishSearchCard struct {
	ID        int    `json:"id"`
	Date      string `json:"date"`
	Name      string `json:"name"`
	Phones    string `json:"phones"`
	Emails    string `json:"emails"`
	TextMatch string `json:"text"`
}

func main() {
	for true {
		fmt.Print("какую рыбу ищем? или напиши \"выход\": ")
		input := bufio.NewScanner(os.Stdin)
		input.Scan()
		switch {
		case input.Text() == "exit":
			os.Exit(0)
		case input.Text() == "выход":
			os.Exit(0)
		}
		searchQuery = input.Text()
		geziyor.NewGeziyor(&geziyor.Options{
			StartURLs: searchURLPages(8),
			ParseFunc: parseFishSearch,
		}).Start()
		toCSV()
		fmt.Println(strings.Replace(emailsTotal, "\n", ",", -1))
	}

}

func searchURLPages(pages int) (result []string) {
	for i := 1; i <= pages; i++ {
		result = append(result, "http://fishery.ru/board?page="+strconv.Itoa(i)+"?s="+url.QueryEscape(encodeWindows1251(searchQuery)))
	}
	return result
}

func parseFishSearch(g *geziyor.Geziyor, r *client.Response) {
	fmt.Printf("Query is: %s\n", searchQuery)
	findJSPath := "#content > div.tires > div"
	r.HTMLDoc.Find(findJSPath).Each(func(i int, s *goquery.Selection) {
		currentSearch := fishSearchCard{ID: i}
		findMails := s.Find("div.name > a").Text()
		currentSearch.Emails = getEMailFromRandomString(findMails)
		emailsTotal = strings.Join([]string{emailsTotal, currentSearch.Emails}, "\n")

		findText, _ := s.Find("div.text").Html()
		decodedText := decodeWindows1251(findText)
		reformatedText := strings.Replace(decodedText, "<br/>", "\n", -1)
		currentSearch.TextMatch = findSearchStringInText(reformatedText)

		findPhone, _ := s.Find("div.name").Html()
		decodePhone := decodeWindows1251(findPhone)
		reformatedPhone := strings.Replace(decodePhone, "<br/>", "\n", -1)

		currentSearch.Name = decodeWindows1251(s.Find("div.name > div.n").Text())

		currentSearch.Date = decodeWindows1251(s.Find("div.name > span > span").Text())[:16]

		currentSearch.Phones = findPhoneStringInText(reformatedPhone)

		searchResult = append(searchResult, currentSearch)
	})
}

func toHTMLTable() {
	jsonResult, err := json.Marshal(searchResult)
	if err != nil {
		fmt.Println(err)
		return
	}

	ok, html := j2t.JSON2HtmlTable(string(jsonResult), []string{}, []string{})
	if ok {
		fmt.Println(html)
	} else {
		fmt.Println("failed to convert json to html table")
	}
}

func toCSV() {
	csvFile, err := os.Create(fmt.Sprintf("./%s-%d.csv", searchQuery, time.Now().Unix()))

	if err != nil {
		fmt.Println(err)
	}
	defer csvFile.Close()

	writer := csv.NewWriter(csvFile)
	if err != nil {
		fmt.Println(err)
		return
	}
	headers := []string{"id", "date", "name", "phones", "email", "text"}
	writer.Write(headers)
	for _, searchRes := range searchResult {
		var row []string
		row = append(row, strconv.Itoa(searchRes.ID))
		row = append(row, searchRes.Date)
		row = append(row, searchRes.Name)
		row = append(row, searchRes.Phones)
		row = append(row, searchRes.Emails)
		row = append(row, searchRes.TextMatch)
		writer.Write(row)
	}
	writer.Flush()
}

func getEMailFromRandomString(possibleMail string) string {
	re := regexp.MustCompile("\\S+@\\S+\\.[^\r\n\t\f\v\\, ]+")
	return ddSliceToNLString(re.FindAllStringSubmatch(possibleMail, -1))
}

func ddSliceToNLString(ddSlice [][]string) string {
	var result []string
	for _, v := range ddSlice {
		for _, s := range v {
			result = append(result, s)
		}
	}
	return strings.Join(result, "\n")
}

func findSearchStringInText(str string) string {
	re := regexp.MustCompile("(?i).*" + searchQuery + ".*")
	if !re.MatchString(str) {
		splittedQuery := strings.Split(searchQuery, " ")[0]
		re = regexp.MustCompile("(?i).*" + splittedQuery + ".*")
	}
	return ddSliceToNLString(re.FindAllStringSubmatch(str, -1))
}

func findPhoneStringInText(str string) string {
	var phoneList []string
	re := regexp.MustCompile("Тел\\..*\\d+")
	matches := re.FindAllStringSubmatch(str, -1)
	for _, value := range matches {
		for _, subval := range value {
			csPhone := subval[9:]
			spPhone := strings.Split(csPhone, ",")
			for _, phone := range spPhone {
				phoneList = append(phoneList, phone)
			}

		}
	}
	return strings.Join(phoneList, "\n")
}

func encodeWindows1251(str string) string {
	enc := charmap.Windows1251.NewEncoder()
	out, _ := enc.String(str)
	return out
}

func decodeWindows1251(str string) string {
	dec := charmap.Windows1251.NewDecoder()
	out, _ := dec.String(str)
	return out
}
