# mgun - an http load testing tool

## Basic configuration

    # The number of concurrent request processes
    concurrency: 2

    # Number of loops to run. 
    # The total numbero of calls will be 
    # loopcount * concurrency * number of urls
    loopcount: 100
    
    # No matter how high the concurrency a value for ratepersecond will force limit
    ratepersecond: 4
    
    # http or https
    # https will not check the certificate
    scheme: https
    
    # The host to query
    host: test.com
    
    # Client timeout in seconds
    timeout: 30
    
    # If -o parameter is given that will be used instead.
    # output: report.txt

    # variables can be used in header, request variable
    params:
        # regular variables are selected for each request and are not related in any way
        search:
            # if the value is an enumeration, then one of the enumeration values ​​will be substituted into the request
            languages: [php, java, c++, c, go, golang, js]
            structures: [for, while, class, if, else, case]
        agent:
            - Mozilla/5.0 (Macintosh; Intel Mac OS X 10_9_5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/38.0.2125.101 Safari/537.36 FirePHP/4Chrome
            - Mozilla/5.0 (Windows NT 6.3; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/37.0.2049.0 Safari/537.36
            - Mozilla/5.0 (compatible; MSIE 10.6; Windows NT 6.1; Trident/5.0; InfoPath.2; SLCC1; .NET CLR 3.0.4506.2152; .NET CLR 3.5.30729; .NET CLR 2.0.50727) 3gpp-gba UNTRUSTED/1.0
            - Mozilla/5.0 (Windows NT 5.1; rv:31.0) Gecko/20100101 Firefox/31.0
        say: hello world

        # global headers to be inserted in every script request, optional parameter
        session:
            - login: user1
            password: password1
            friendIds: [1, 2, 3, 4]
            - login: user2
            password: password2
            friendIds: [5, 6, 7, 8]

    # global headers to be inserted in every script request, optional parameter
    headers:
    Content-Type: text/html; charset=utf-8
    User-Agent: ${agent}
    X-Key-1: Value-1

    # List of requests, either random of in order
    requests:
        - RANDOM:
            - GET: /base/level/two?start=P0D&end=U365D&usr=bob
            - GET: /base/next/level/report?usr=bob
            - GET: /base/event-actions?date=2021-01-30&count=5&usr=bob
            - GET: /base/sub/group?date=2021-02-17&usr=bob
            - GET: /base/next?date=2021-02-17&count=5&usr=bob
            - GET: /base/next?date=2021-02-16&count=20&usr=bob
            - GET: /base/next/one?count=100&usr=bob&date=2021-02-17

        # this request headers, optional
        - GET: /soccer/event-actions-with-deleted?count=100&usr=xts&date=2021-02-17
        headers:
            # will overwrite the value of the global header to local
            X-Key-1: New-Value-1
            X-Key-2: Value-2

        - POST: /some/path/save
          params:
              friend_id: ${session.friendIds}
              say: ${say}

        # SYNC - in this group all requests will be executed in the order in which they are specified. 
        # SYNC can be both inside and outside RANDOM.
        - SYNC:
          - GET: /some/path?query=2
          - GET: /some/path/3?query=123
