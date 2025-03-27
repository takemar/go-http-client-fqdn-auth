# go-http-client-fqdn-auth

This Go module is a tiny program working with the nginx [`auth_request`](https://nginx.org/en/docs/http/ngx_http_auth_request_module.html) directive and performs client authentication based on its domain (FQDN).

The program receives subrequests from nginx and determines each time if the IP address of the client matches the one for the specified domain. Since it performs DNS resolution on every request, it can properly perform domain-based authentication even if DDNS is employed.

## Usage

First, build the module and place the binary in an arbitrary location.

```
$ go build .
$ sudo cp http-client-fqdn-auth /usr/local/bin
```

Then you can run the program. For example, to receive subrequests from nginx on port 8080 and permit clients from IP addresses of domains `client.example.com` and `client.example.org`:

```
http-client-fqdn-auth --port 8080 client.example.com client.example.org
```

It would be a good idea to create a systemd service file:

```
[Unit]
Description=HTTP client authentication based on domain

[Service]
ExecStart=/usr/local/bin/http-client-fqdn-auth --port 8080 client.example.com client.example.org
Restart=always

[Install]
WantedBy=multi-user.target
```

Also, set up `auth_request` subrequest in your nginx configuration file:

```
http {
    upstream auth {
        server 127.0.0.1:8080;
    }

    server {
        listen 80; ssl;
        server_name server.example.org;

        auth_request /auth;
        location /auth {
            internal;
            proxy_pass_request_body off;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_pass http://auth/;
        }
    }
}
```

It is very important to **set the `X-Forwarded-For` header properly**, as the program relies on this header. In addition, make sure that the `proxy_pass` directive ends with a `/` so that subrequests do not contain the path of the original request.

## Command line options

- `--port`, `-p`: The port that this program will listen on.
- `--listen-ip`: The IP address that this program will listen on. `127.0.0.1` will disable direct access from the outside.
- `--socket`, `-s`: The path of the UNIX domain socket that this program will listen on instead of the TCP port. Cannot be specified with `--port` and `--listen-ip` at the same time.
- `--trusted-proxy`: In case of a multiple (reverse) proxy arrangement, an IP address beyond the proxy specified in this option will be used as the basis for the decision. Note that it assumes that the IP addresses of the proxies are included in the `X-Forwarded-For` header in the correct order. Can be specified multiple times.

## Note

This program executes [`net.LookupHost`](https://pkg.go.dev/net#LookupHost) of Go each time receiving a request. To ensure good performance, it is recommended to configure a DNS resolver with caching capabilities (such as [systemd-resolved](https://www.freedesktop.org/software/systemd/man/latest/systemd-resolved.html)) on your server and make sure that Go is using it.
