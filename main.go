package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"time"
)

var (
	output     *string // 输出位置
	link       *string // 奶牛链接
	directLink *bool   // 是否只返回直链
	password   *string // 口令
)

func init() {
	output = flag.String("o", "", "输出位置")
	link = flag.String("l", "", "快传链接，和口令二选一")
	directLink = flag.Bool("d", false, "只返回链接，但不进行下载")
	// password = flag.String("p", "", "快传口令，和链接二选一") // TODO 目前只支持直链
}

func main() {
	flag.Parse()

	uniqueUrl := *password
	if password == nil || *password == "" {
		if link == nil || *link == "" {
			fmt.Println("链接不得为空")
			return
		}

		pattern := regexp.MustCompile(`https://cowtransfer.com/s/([a-zA-Z0-9]+)`)
		mathResult := pattern.FindStringSubmatch(*link)
		if len(mathResult) < 2 {
			fmt.Println("没有匹配到结果")
			return
		}

		uniqueUrl = mathResult[1]
	}

	if uniqueUrl == "" {
		fmt.Println("链接格式错误")
		return
	}

	shareInfo := "https://cowtransfer.com/core/api/transfer/share?uniqueUrl=" + uniqueUrl
	body, err := RequestGet("GET", shareInfo)
	if err != nil {
		fmt.Println(err)
		return
	}

	shareInfoMap := make(map[string]interface{})
	err = json.Unmarshal(body, &shareInfoMap)
	if err != nil {
		fmt.Println(err)
		return
	}

	if shareInfoMap["code"].(string) != "0000" {
		fmt.Println(shareInfoMap["message"])
		return
	}

	data := make(map[string]interface{})
	data = shareInfoMap["data"].(map[string]interface{})

	// TODO 详情
	// if info {
	//
	// }

	guid := data["guid"].(string)
	if guid == "" {
		fmt.Println("获取guid失败")
		return
	}

	title := data["transferName"].(string)

	firstFile := make(map[string]interface{})
	firstFile = data["firstFile"].(map[string]interface{})
	fileId := firstFile["id"].(string)

	queryUrl := "https://cowtransfer.com/core/api/transfer/share/download/links?password=&transferGuid=" + guid + "&title=" + title + "&fileId=" + fileId

	queryResult, err := RequestGet("GET", queryUrl)
	if err != nil {
		fmt.Println(err)
		return
	}

	queryResultMap := make(map[string]interface{})
	err = json.Unmarshal(queryResult, &queryResultMap)
	if err != nil {
		fmt.Println(err)
		return
	}

	if queryResultMap["code"].(string) != "0000" {
		fmt.Println(queryResultMap["message"])
		return
	}

	downloadArray := make([]interface{}, 0)
	downloadArray = queryResultMap["data"].([]interface{})
	if len(downloadArray) < 1 {
		fmt.Println("获取下载链接失败")
		return
	}

	downloadUrl := downloadArray[0].(string)

	if *directLink {
		fmt.Println(downloadUrl)
		return
	}

	path := ""

	// 开始下载到本地
	if output == nil || *output == "" {

		fileName := title
		fileInfo := make(map[string]interface{})
		fileInfo = firstFile["file_info"].(map[string]interface{})
		fileType := fileInfo["format"].(string)

		// 获取当前绝对路径
		dir, err := os.Getwd()
		if err != nil {
			fmt.Println(err)
			return
		}

		path = dir + "/" + fileName + "." + fileType
	} else {
		path = *output
	}

	// 绝对路径
	err = DownloadFile(path, downloadUrl)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("下载完成，路径：%s\n", path)
}

func RequestGet(method string, url string) ([]byte, error) {

	client := &http.Client{}
	req, err := http.NewRequest(method, url, nil)

	if err != nil {
		return []byte{}, err
	}
	req.Header.Add("Accept-Language", "zh-CN,zh;q=0.9")
	req.Header.Add("Referer", "https://cowtransfer.com/mobile")
	req.Header.Add("User-Agent", "Mozilla/5.0 (iPhone; CPU iPhone OS 13_2_3 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/13.0.3 Mobile/15E148 Safari/604.1")
	req.Header.Add("x-business-code", "COW_TRANSFER")
	req.Header.Add("x-channel-code", "COW_CN_WECHAT")

	res, err := client.Do(req)
	if err != nil {
		return []byte{}, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return []byte{}, err
	}

	return body, nil
}

// DownloadFile will download a url to a local file with progress and percentage.
func DownloadFile(filepath string, url string) error {
	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Get the data
	resp, err := RequestReader(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Get the content length from header
	size, err := strconv.Atoi(resp.Header.Get("Content-Length"))
	if err != nil {
		return err
	}

	// Create a progress reader
	progress := &ProgressReader{Reader: resp.Body, Total: int64(size)}

	// Write the body to file with progress
	_, err = io.Copy(out, progress)
	if err != nil {
		return err
	}

	return nil
}

// ProgressReader is a reader with progress.
type ProgressReader struct {
	Reader         io.Reader
	Total          int64
	Current        int64
	LastRecordTIme time.Time
	LastRecordSize int64
}

// Read reads from the underlying reader and updates the progress.
func (pr *ProgressReader) Read(p []byte) (n int, err error) {
	n, err = pr.Reader.Read(p)
	pr.Current += int64(n)
	pr.PrintProgress()
	return
}

// PrintProgress prints the download progress.
func (pr *ProgressReader) PrintProgress() {
	if time.Now().Unix()-pr.LastRecordTIme.Unix() < 1 {
		return
	}

	percent := float64(pr.Current) / float64(pr.Total) * 100
	speed := float64(pr.Current-pr.LastRecordSize) / time.Since(pr.LastRecordTIme).Seconds() / 1024 / 1024
	fmt.Printf("\r%.2f%% 下载中... %.2fMB/s", percent, speed)
	if pr.Current == pr.Total {
		fmt.Println()
	}

	pr.LastRecordTIme = time.Now()
	pr.LastRecordSize = pr.Current
}

func RequestReader(url string) (*http.Response, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)

	if err != nil {
		return nil, err
	}
	req.Header.Add("Accept-Language", "zh-CN,zh;q=0.9")
	req.Header.Add("Referer", "https://cowtransfer.com/mobile")
	req.Header.Add("User-Agent", "Mozilla/5.0 (iPhone; CPU iPhone OS 13_2_3 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/13.0.3 Mobile/15E148 Safari/604.1")
	req.Header.Add("x-business-code", "COW_TRANSFER")
	req.Header.Add("x-channel-code", "COW_CN_WECHAT")

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	return res, nil
}
