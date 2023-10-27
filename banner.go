package main

import (
	"bufio"
	"crypto/tls"
	"flag"
	"fmt"
	"golang.org/x/net/html"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	bannerString string
	titleString  string
	port         string
	output       string
	threads      int
)

func init() {
	flag.StringVar(&bannerString, "b", "", "string to search for within headers")
	flag.StringVar(&titleString, "title", "", "string to search for within the title")
	flag.StringVar(&port, "p", "80", "port number")
	flag.StringVar(&output, "o", "output.txt", "output file")
	flag.IntVar(&threads, "t", 1, "number of threads")
}

func getHeaders(ip string, port string) (string, string, string, error) {
	url := fmt.Sprintf("https://%s:%s", ip, port)
	headers, title, err := fetchHeaders(url)
	if err != nil {
		url = fmt.Sprintf("http://%s:%s", ip, port)
		headers, title, err = fetchHeaders(url)
	}
	return url, headers, title, err
}

func fetchHeaders(url string) (string, string, error) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr, Timeout: time.Second * 3}
	resp, err := client.Get(url)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	headers := resp.Proto + "\n"
	for key, values := range resp.Header {
		for _, value := range values {
			headers += key + ": " + value + "\n"
		}
	}
	headers += "\n"

	var title string
	tokenizer := html.NewTokenizer(resp.Body)
	for {
		tokenType := tokenizer.Next()
		switch tokenType {
		case html.ErrorToken:
			return headers, title, nil
		case html.StartTagToken, html.SelfClosingTagToken:
			token := tokenizer.Token()
			if token.Data == "title" {
				tokenType = tokenizer.Next()
				if tokenType == html.TextToken {
					title = tokenizer.Token().Data
					title = strings.TrimSpace(title)
				}
			}
		}
	}
}

func main() {
	flag.Parse()

	var wg sync.WaitGroup
	sem := make(chan bool, threads)

	file, err := os.Create(output)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		ip := scanner.Text()
		sem <- true
		wg.Add(1)

		go func(ip string) {
			defer func() {
				<-sem
				wg.Done()
			}()

			url, headers, title, err := getHeaders(ip, port)
			if err == nil {
				matchBanner := bannerString == "" || contains(headers, bannerString)
				matchTitle := titleString == "" || contains(title, titleString)

				if matchBanner && matchTitle {
					outputString := fmt.Sprintf("%s:%s, %s\n%s\nTitle: %s\n\n", ip, port, url, headers, title)
					fmt.Fprintf(file, outputString)
				}
			}
		}(ip)
	}

	wg.Wait()
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}