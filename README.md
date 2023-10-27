# Golang-url-and-ip-Title-and-Header-fetcher
Golang url and ip Title and Header fetcher to automatically fetch header data and title data, extensive to allow filtering of specific titles, banners, or both conditions.

## To install the program:

go build banner.go

go mod init banner

go mod tidy

## To use the program:

ips.txt | ./banner -p portnumber -o outputfile -t threadcount -b "headerdata/bannerdata" -t "titledata"

Script will automatically output header data/banner data and title data if command line arg isn't specified.

Zmap can also be piped into the program.

Threads is the number of concurrect connections, for example if you put 5000 threads it will scan 5000 ips/urls consecutively.
