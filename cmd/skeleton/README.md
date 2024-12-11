# How to check

Run this:
```
curl http://localhost:10443/api/ok -v
```

And expect this:
```
*   Trying 127.0.0.1:10443...
* Connected to localhost (127.0.0.1) port 10443 (#0)
> GET /api/ok HTTP/1.1
> Host: localhost:10443
> User-Agent: curl/8.0.1
> Accept: */*
> 
< HTTP/1.1 200 OK
< Date: Wed, 11 Dec 2024 02:00:35 GMT
< Content-Length: 0
< 
* Connection #0 to host localhost left intact
```