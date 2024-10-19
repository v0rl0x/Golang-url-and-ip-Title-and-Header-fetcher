package main

import (
    "bufio"
    "crypto/tls"
    "flag"
    "fmt"
    "golang.org/x/net/html"
    "net/http"
    "net/url"
    "os"
    "strings"
    "sync"
    "time"
)

var (
    bannerString  string
    titleString   string
    urlPath       string
    port          string
    output        string
    threads       int
    headerKey     string
    headerValue   string
    client        *http.Client
)

func init() {
    flag.StringVar(&bannerString, "b", "", "string to search for within headers")
    flag.StringVar(&titleString, "title", "", "string to search for within the title")
    flag.StringVar(&urlPath, "url", "", "URL path to append or file containing URL paths")
    flag.StringVar(&port, "p", "80", "port number (ignored if full URL is provided)")
    flag.StringVar(&output, "o", "output.txt", "output file")
    flag.IntVar(&threads, "t", 1, "number of threads")
    flag.StringVar(&headerKey, "hk", "", "header key to search for")
    flag.StringVar(&headerValue, "hv", "", "header value to search for")

    tr := &http.Transport{
        TLSClientConfig:    &tls.Config{InsecureSkipVerify: true},
        MaxIdleConns:       100,
        IdleConnTimeout:    30 * time.Second,
        DisableCompression: true,
    }
    client = &http.Client{Transport: tr, Timeout: time.Second * 30}
}

func getHeadersWithRetry(target string, port string, urlPath string, retries int) (*http.Response, string, string, string, error) {
    var resp *http.Response
    var finalURL, headers, title string
    var err error

    for i := 0; i < retries; i++ {
        resp, finalURL, headers, title, err = getHeaders(target, port, urlPath)
        if err == nil && resp != nil && resp.StatusCode == http.StatusOK {
            return resp, finalURL, headers, title, nil
        }
        time.Sleep(time.Duration(i) * time.Second)
    }

    return nil, "", "", "", fmt.Errorf("failed after %d retries: %v", retries, err)
}

func getHeaders(target string, port string, urlPath string) (*http.Response, string, string, string, error) {
    var finalURL string

    if isValidURL(target) {
        finalURL = fmt.Sprintf("%s%s", strings.TrimRight(target, "/"), urlPath)
    } else {
        finalURL = fmt.Sprintf("https://%s:%s%s", target, port, urlPath)
    }

    resp, headersBuilder, title, err := fetchHeaders(finalURL)
    if err != nil || resp == nil || resp.StatusCode != http.StatusOK {
        if isValidURL(target) {
            return nil, "", "", "", err
        }
        finalURL = fmt.Sprintf("http://%s:%s%s", target, port, urlPath)
        resp, headersBuilder, title, err = fetchHeaders(finalURL)
        if err != nil || resp == nil || resp.StatusCode != http.StatusOK {
            return nil, "", "", "", err
        }
    }

    headers := headersBuilder.String()
    return resp, finalURL, headers, title, nil
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

func isValidURL(str string) bool {
    _, err := url.ParseRequestURI(str)
    return err == nil
}

func isFile(path string) bool {
    info, err := os.Stat(path)
    return err == nil && !info.IsDir()
}

func readURLPathsFromFile(filePath string) ([]string, error) {
    var urls []string
    file, err := os.Open(filePath)
    if err != nil {
        return nil, err
    }
    defer file.Close()

    scanner := bufio.NewScanner(file)
    for scanner.Scan() {
        line := strings.TrimSpace(scanner.Text())
        if line != "" {
            urls = append(urls, line)
        }
    }
    if err := scanner.Err(); err != nil {
        return nil, err
    }
    return urls, nil
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
    var urlPaths []string

    if isFile(urlPath) {
        urlPaths, err = readURLPathsFromFile(urlPath)
        if err != nil {
            fmt.Fprintf(os.Stderr, "Error reading URL paths from file: %v\n", err)
            return
        }
    } else {
        urlPaths = []string{urlPath}
    }

    for scanner.Scan() {
        target := scanner.Text()
        for _, urlSuffix := range urlPaths {
            wg.Add(1)
            sem <- struct{}{}

            go func(target, urlSuffix string) {
                defer func() {
                    <-sem
                    wg.Done()
                }()

                resp, url, headers, title, err := getHeadersWithRetry(target, port, urlSuffix, 3)
                if err != nil {
                    fmt.Fprintf(os.Stderr, "Error fetching headers for %s: %v\n", target, err)
                    return
                }

                if resp != nil && resp.StatusCode == http.StatusOK {
                    pageNotFound := contains(title, "Not Found") || contains(headers, "Not Found") || contains(title, "404") || contains(headers, "404")

                    if !pageNotFound {
                        matchBanner := bannerString == "" || contains(headers, bannerString)
                        matchTitle := titleString == "" || contains(title, titleString)

                        matchHeader := true
                        if headerKey != "" && headerValue != "" {
                            headerVal := resp.Header.Get(headerKey)
                            matchHeader = strings.Contains(headerVal, headerValue)
                        }

                        if (matchBanner && matchTitle && matchHeader) || (bannerString == "" && titleString == "" && matchHeader) {
                            outputString := fmt.Sprintf("%s, %s\n%s\nTitle: %s\n\n", target, url, headers, title)
                            fmt.Fprint(writer, outputString)
                            writer.Flush()
                        }
                    }
                }
            }(target, urlSuffix)
        }
    }

    if err := scanner.Err(); err != nil {
        fmt.Fprintln(os.Stderr, "Error reading from input:", err)
    }

    wg.Wait()
}

func contains(s, substr string) bool {
    return strings.Contains(s, substr)
}
