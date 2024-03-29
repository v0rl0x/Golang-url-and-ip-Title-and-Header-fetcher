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
    urlPath      string
    port         string
    output       string
    threads      int
    client       *http.Client
)

func init() {
    flag.StringVar(&bannerString, "b", "", "string to search for within headers")
    flag.StringVar(&titleString, "title", "", "string to search for within the title")
    flag.StringVar(&urlPath, "url", "", "URL path to append to the IP")
    flag.StringVar(&port, "p", "80", "port number")
    flag.StringVar(&output, "o", "output.txt", "output file")
    flag.IntVar(&threads, "t", 1, "number of threads")

    tr := &http.Transport{
        TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
        MaxIdleConns:        100,
        IdleConnTimeout:     30 * time.Second,
        DisableCompression:  true,
    }
    client = &http.Client{Transport: tr, Timeout: time.Second * 10}
}

func getHeaders(ip string, port string, urlPath string) (*http.Response, string, string, string, error) {
    url := fmt.Sprintf("https://%s:%s%s", ip, port, urlPath)
    resp, headersBuilder, title, err := fetchHeaders(url)
    if err != nil || resp.StatusCode != http.StatusOK {
        url = fmt.Sprintf("http://%s:%s%s", ip, port, urlPath)
        resp, headersBuilder, title, err = fetchHeaders(url)
        if err != nil || resp.StatusCode != http.StatusOK {
            return nil, "", "", "", err
        }
    }
    headers := headersBuilder.String()
    return resp, url, headers, title, nil
}

func fetchHeaders(url string) (*http.Response, *strings.Builder, string, error) {
    resp, err := client.Get(url)
    if err != nil {
        return nil, nil, "", err
    }
    defer resp.Body.Close()

    headers := &strings.Builder{}
    headers.WriteString(resp.Proto)
    headers.WriteString("\n")

    for key, values := range resp.Header {
        for _, value := range values {
            headers.WriteString(key)
            headers.WriteString(": ")
            headers.WriteString(value)
            headers.WriteString("\n")
        }
    }
    headers.WriteString("\n")

    var title string
    tokenizer := html.NewTokenizer(resp.Body)
    for {
        tokenType := tokenizer.Next()
        switch tokenType {
        case html.ErrorToken:
            return resp, headers, title, nil
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
    sem := make(chan struct{}, threads)

    file, err := os.Create(output)
    if err != nil {
        fmt.Println("Error:", err)
        return
    }
    defer file.Close()

    writer := bufio.NewWriter(file)
    defer writer.Flush()

    scanner := bufio.NewScanner(os.Stdin)
    for scanner.Scan() {
        ip := scanner.Text()
        wg.Add(1)
        sem <- struct{}{}

        go func(ip string) {
            defer func() {
                <-sem
                wg.Done()
            }()

            resp, url, headers, title, err := getHeaders(ip, port, urlPath)
            if err != nil {
                fmt.Fprintf(os.Stderr, "Error fetching headers: %v\n", err)
                return
            }

            if resp != nil && resp.StatusCode == http.StatusOK {
                pageNotFound := contains(title, "Not Found") || contains(headers, "Not Found") || contains(title, "404") || contains(headers, "404")

                if !pageNotFound {
                    matchBanner := bannerString == "" || contains(headers, bannerString)
                    matchTitle := titleString == "" || contains(title, titleString)

                    if (matchBanner && matchTitle) || (bannerString == "" && titleString == "") {
                        outputString := fmt.Sprintf("%s:%s%s, %s\n%s\nTitle: %s\n\n", ip, port, urlPath, url, headers, title)
                        fmt.Fprint(writer, outputString)
                        writer.Flush()
                    }
                }
            }
        }(ip)
    }

    if err := scanner.Err(); err != nil {
        fmt.Fprintln(os.Stderr, "Error reading from input:", err)
    }

    wg.Wait()
}

func contains(s, substr string) bool {
    return strings.Contains(s, substr)
}
