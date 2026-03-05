# Custom DNS-proxy server from scratch

DNS-proxy is developed according to RFC <a href="https://www.rfc-editor.org/rfc/rfc1035">1035</a>

<h2>How to use</h2>

Run in default way
```
make run 
```

Run with race checking
```
go run -race ./cmd/main.go 
```

Run via docker
```
make docker
```

<h2>How to test<h2>

```
go test -v ./...
```

<p> sudo apt install dig </p>

```
dig @127.0.0.1 -p 8530 {your-desire-domain.com}
```

<h2>What is not implemeted</h2>

<ul>
    <li><b>EDNS<b></li>
    <li><b>DoT<b></li>
    <li><b>DNSSEC</b></li>
    <li><b>Recursion searching<b></li>
    <li><b>Handling others NS</b></li>
</ul>