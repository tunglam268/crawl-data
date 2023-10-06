package main

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/xuri/excelize/v2"
	"log"
	"net/http"
	"sync"
	"time"
)

const DomainURL = `https://vnexpress.net`

func extractHrefInformationCarsFromLinkURL(url string, wg *sync.WaitGroup, ch chan<- string) {
	defer wg.Done()
	// Tạo yêu cầu HTTP đến trang web
	response, err := http.Get(url)
	if err != nil {
		log.Printf("Lỗi khi tải %s: %v", url, err)
		return
	}
	defer response.Body.Close()

	// Kiểm tra mã trạng thái của yêu cầu HTTP
	if response.StatusCode != 200 {
		log.Printf("Mã trạng thái không hợp lệ khi tải %s: %d", url, response.StatusCode)
		return
	}

	// Sử dụng goquery để phân tích HTML
	doc, err := goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		log.Printf("Lỗi khi phân tích HTML từ %s: %v", url, err)
		return
	}

	// Tìm tất cả các thẻ có class "btn-default.btn-thongso"
	doc.Find(".btn-default.btn-thongso").Each(func(index int, item *goquery.Selection) {
		// Lấy giá trị thuộc tính href của thẻ
		href, exists := item.Attr("href")
		fmt.Println(`Link: `, href)
		if exists {
			ch <- href
		}
	})
}

func readDataFromExcel(excelFilePath string) ([]string, error) {
	xlsx, err := excelize.OpenFile(excelFilePath)
	if err != nil {
		return nil, err
	}

	rows, err := xlsx.GetRows("Sheet1")
	if err != nil {
		return nil, err
	}

	data := []string{}

	for _, row := range rows {
		if len(row) > 0 {
			data = append(data, row[0])
		}
	}

	return data, nil
}

func extractHrefsFromDivs(url string) []string {
	response, err := http.Get(DomainURL + url)
	if err != nil {
		log.Printf("Lỗi khi tải %s: %v", url, err)
		return nil
	}

	doc, err := goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		log.Printf("Lỗi khi phân tích HTML: %v", err)
		return nil
	}

	hrefs := []string{}

	doc.Find("div.btn.sort.sort-version-on-pc a").Each(func(index int, item *goquery.Selection) {
		href, exists := item.Attr("data-link-version")
		if exists {
			hrefs = append(hrefs, DomainURL+href)
		}
	})

	return hrefs
}

func writeDataToExcel(data []string, excelFilePath string) error {
	// Tạo một tệp Excel mới
	xlsx := excelize.NewFile()

	// Tạo một sheet mới
	sheetName := "Sheet1"
	xlsx.NewSheet(sheetName)

	// Ghi dữ liệu từ mảng data vào sheet
	for i, value := range data {
		cellName := fmt.Sprintf("A%d", i+1)
		xlsx.SetCellValue(sheetName, cellName, value)
	}

	// Lưu tệp Excel
	err := xlsx.SaveAs(excelFilePath)
	if err != nil {
		return err
	}

	return nil
}

func main() {
	excelFilePath := "output.xlsx"
	data, err := readDataFromExcel(excelFilePath)
	if err != nil {
		log.Fatal(err)
	}

	var wg sync.WaitGroup
	ch := make(chan string, len(data))

	for _, url := range data {
		wg.Add(1)
		go extractHrefInformationCarsFromLinkURL(url, &wg, ch)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	var allResults []string
	fmt.Println("Unique HREFs:")
	uniqueHrefs := make(map[string]struct{})
	for href := range ch {
		if _, exists := uniqueHrefs[href]; !exists {
			uniqueHrefs[href] = struct{}{}
			time.Sleep(time.Second * 30)
			allResults = extractHrefsFromDivs(href)
		}
	}

	wg.Wait()

	// Lấy dữ liệu từ các kênh Goroutines và kết hợp vào slice allResults
	excelFilePathOut := "outputcars.xlsx"
	err = writeDataToExcel(allResults, excelFilePathOut)
	if err != nil {
		log.Fatal(err)
	}

}
