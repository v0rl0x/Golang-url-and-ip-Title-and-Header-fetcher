# Golang-url-and-ip-Title-and-Header-fetcher
Golang url and ip Title and Header fetcher to automatically fetch header data and title data, extensive to allow filtering of specific titles, banners, or both conditions.

## To install the program:

go build banner.go

go mod init banner

go mod tidy

## To use the program:

ips.txt | ./banner -p portnumber -o outputfile -t threadcount -b "headerdata/bannerdata" -title "titledata" -url "urldata"

NOTE: Url data will try to input a url such as http://1.1.1.1/db_backup/ for example -url "/db_backup/" to the IP or any other data. It is recommended to also put a title data in there or you will get false positives for example -title "Index of /db_backup" with the URL flag option.

Script will automatically output header data/banner data and title data if command line arg isn't specified.

The -url flag can handle multiple urls, for example -url urls.txt, one URL per line such as /db/

Now supports header values and header types. -hk "Content-Type" -hv "application/json"

Zmap can also be piped into the program.

Threads is the number of concurrect connections, for example if you put 5000 threads it will scan 5000 ips/urls consecutively.

NOTE: You do not need to input -b or -t field if you do not want too! You can use -b or -t alone.

## extract.go file information

Extract.go file will automatically extract the IP:PORT off the resulting file from the fetched data. It will output to extract-output.txt can be modified in the code for your specific naming needs.

go build extract.go

./extract filename
