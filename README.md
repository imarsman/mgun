# M (achine) gun

This project is a fork from an excellent project of the same name. The flexible
configuration is excellent and really appreciated. This fork adds rate limiting,
a random delay, some slight enhancements to version information, the ability to
output request summaries to a file, a bit of modification of the results layout
to avoid line wrapping, and the ability to print out a sample configuration.

The gun terminology is interesting but I (Ian Marsman) am not as well versed in
firearm terminology as perhaps the original authors are. I have changed some of
the nomenclature for the various aspects of the request process for my own clarity.

** Mgun ** is an HTTP server load testing tool.

Mgun creates a specified number of concurrent concurrent sessions, then executes
HTTP requests and aggregates the results into a table. Parallelism depends on
the number of processor cores; the more cores, the higher the parallelism of
HTTP requests.

Mgun allows you to create GET, POST, PUT, DELETE requests.

The fundamental difference between mgun and other load testing tools is that
that it allows you to create a script from an arbitrary number of requests,
simulating real user behavior. For example, a script might look like:

1. Go to the main page of the site
2. Log in
3. Log into your personal account
4. Change information about yourself
5. Log out

Requests in such a script will be executed sequentially as if the user were
doing it.

A configuration file in YAML format is used to create scripts.

In addition to the sequence of requests, you can specify the timeout and headers
in the configuration file. Timeout and headers can be either global for all
requests, or private for each request.

# Fast start

Mgun is written in Go, so you need to [install Go]
(http://golang.org/doc/install) first.

### Launch

```
    $ ./bin/mgun -f example/config.yaml

    Server Hostname:       test.com
    Server Port:           80
    Concurrency Level:     2
    Rate per second:       4
    Random delay ms:       0
    Loop count:            10
    Timeout:               30 seconds
    Time taken for tests:  22 seconds
    Total requests:        80
    Complete requests:     80
    Failed requests:       0
    Availability:          100.00%
    Requests per second:   ~ 1.15
    Total transferred:     1.5 MB

    #   Request
        Compl     Fail.     Min/s     Max/s     Avg/s.    Avail%    Min/Ave/Max req/s.   Cont len    Total trans
    1.  GET /api/test1
        20        0         0.301     0.605     0.453     100.00    1 / ~ 1.11 / 2       19 kB       374 kB

    2.  GET /api/test2
        20        0         0.312     0.526     0.419     100.00    1 / ~ 1.11 / 2       102 B       2.0 kB

    3.  GET /apip/test3
        20        0         0.276     0.569     0.423     100.00    1 / ~ 1.25 / 2       19 kB       374 kB

    4.  GET /api/test4
        20        0         0.371     0.853     0.612     100.00    1 / ~ 1.11 / 2       37 kB       738 kB
```