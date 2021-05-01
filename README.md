# M (achine) gun

** Mgun ** is an HTTP server load testing tool.

Mgun creates a specified number of concurrent concurrent sessions, then executes HTTP requests and aggregates the results into a table.
Parallelism depends on the number of processor cores; the more cores, the higher the parallelism of HTTP requests.

Mgun allows you to create GET, POST, PUT, DELETE requests.

The fundamental difference between mgun and other load testing tools is that
that it allows you to create a script from an arbitrary number of requests, simulating real user behavior.
For example, a script might look like:

1. Go to the main page of the site
2. Log in
3. Log into your personal account
4. Change information about yourself
5. Log out

Requests in such a script will be executed sequentially as if the user were doing it.

A configuration file in YAML format is used to create scripts.

In addition to the sequence of requests, you can specify the timeout and headers in the configuration file.
Timeout and headers can be either global for all requests, or private for each request.

# Fast start

Mgun is written in Go, so you need to [install Go] (http://golang.org/doc/install) first.

### Download and install from source

    cd /path/to/gopath
    export GOPATH=/path/to/gopath/
    export GOBIN=/path/to/gopath/bin/
    go get github.com/byorty/mgun
    go install src/github.com/byorty/mgun/mgun.go

### Launch

    ./bin/mgun -f example/config.yaml

    1000 / 1000 [=============================================================================================================================================================] 100.00 %
    
    Server Hostname:       example.com
    Server Port:           80
    Concurrency Level:     100
    Loop count:            1
    Timeout:               30 seconds
    Time taken for tests:  89 seconds
    Total requests:        1000
    Complete requests:     997
    Failed requests:       3
    Availability:          99.70%
    Requests per second:   ~ 2.99
    Total transferred:     183MB
    
    #    Request                            Compl.  Fail.  Min.     Max.      Avg.      Avail.   Min, avg, max req. per sec.  Content len.  Total trans.
    1.   GET /                              100     0      1.047s.  9.285s.   5.166s.   100.00%  1 / ~ 9.09 / 24              131KB         13MB
    2.   POST /signin                       100     0      0.831s.  8.277s.   4.554s.   100.00%  1 / ~ 5.88 / 16              72B           7.2KB
    3.   GET /basket/                       100     0      0.268s.  22.094s.  11.181s.  100.00%  1 / ~ 1.67 / 4               61KB          15MB
    4.   GET /orders/                       100     0      0.390s.  31.168s.  15.779s.  100.00%  1 / ~ 1.82 / 5               58KB          17MB
    5.   GET /shoes?category=${categories}  98      2      1.089s.  31.546s.  16.318s.  98.00%   1 / ~ 1.72 / 6               476KB         49MB
    6.   GET /shoes?search=${search.shoes}  100     0      1.580s.  17.007s.  9.293s.   100.00%  1 / ~ 1.67 / 5               136KB         16MB
    7.   GET /shoes?id=${shoes.ids}         100     0      0.659s.  14.873s.  7.766s.   100.00%  1 / ~ 1.92 / 6               202KB         20MB
    8.   GET /friends/                      99      1      0.289s.  30.010s.  15.149s.  99.00%   1 / ~ 1.96 / 6               61KB          17MB
    9.   POST /friends/say?id=${frind.ids}  100     0      0.535s.  25.595s.  13.065s.  100.00%  1 / ~ 1.96 / 7               240KB         16MB
    10.  GET /friend?id=${frind.ids}        100     0      2.534s.  17.051s.  9.792s.   100.00%  1 / ~ 2.22 / 6               132KB         19MB



