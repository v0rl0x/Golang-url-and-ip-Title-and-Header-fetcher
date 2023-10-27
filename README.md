# Golang-url-and-ip-Title-and-Header-fetcher
Golang url and ip Title and Header fetcher to automatically fetch header data and title data, extensive to allow filtering of specific titles, banners, or both conditions.

## To install the program:

go build banner.go

## To use the program:

ips.txt | ./banner -p portnumber -o outputfile -t threadcount -b headerdata/bannerdata -t titledata

Script will automatically output header data/banner data and title data if command line arg isn't specified.

Zmap can also be piped into the program.
