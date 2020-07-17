# DroppingServer

This is a small Golang application used to test proxy behaviours

The server implemented features:

- reply OK on a `GET /`
- reply "Hello World!" on a `GET /world`
- reply "Hello World!" on a `GET /slow` request, waiting 20s by default or use the `?delay=60` to the request to wait for 60s
- reply with bogus data on a `GET /hj` request. In this case, the server will send some trash on the line before the regular HTTP headers, leading to (usually) a 503 error on a proxy
- reply with a duplicate header `Transfer-Encoding: chunked` on a `GET /dup` 
- reply with a duplicate header `x-content-type-options: nosniff` on a `GET /dup2`

## Usage

You can use the Docker image prune/droppingserver:v0.0.1 directly
You can also deploy to Kubernetes using the dumb provided yaml (change the namesapce from `default` to suite your needs):

```bash
kubectl -n default apply -f deployment.yaml
```

## TODO

I need to add more scenarios here, like:
- having a large (like many Mb) reply from the server
- limit the server response to a low mbps to test a long connexion with few bits flowing through

## Demo

```bash
curl -vv 'http://localhost:8080/hj'
*   Trying ::1...
* TCP_NODELAY set
* Connected to localhost (::1) port 8080 (#0)
> GET /hj HTTP/1.1
> Host: localhost:8080
> User-Agent: curl/7.64.1
> Accept: */*
> 
* Closing connection 0
connexion hijacked by the server
```

