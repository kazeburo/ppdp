# ppdp

TCP Proxy (relay) server with Dynamic upstream resolution, Some balancing algorism, Support Proxy Protocol and TCPDump like function

```
% ./ppdp -h
Usage:
  ppdp [OPTIONS]

Application Options:
  -v, --version                                       Show version
  -l, --listen=                                       address to bind (default: 0.0.0.0:3000)
      --upstream=                                     upstream server: upstream-server:port
      --proxy-connect-timeout=                        timeout of connection to upstream (default: 60s)
      --proxy-protocol                                use proxy-proto for listen
      --dump-tcp=                                     Dump TCP. 0 = disable, 1 = src to dest, 2 = both (default: 0)
      --dump-mysql-ping                               Dump mysql ping packet
      --max-connect-retry=                            number of max connection retry (default: 3)
      --balancing=[leastconn|iphash|fixed|remotehash] balancing mode connection to upstream. iphash: remote ip based, remotehash: remote ip + port based, fixed:
                                                      upstream host based (default: leastconn)

Help Options:
  -h, --help                                          Show this help message
 ```

Sample

```
$ ./ppdp --proxy-protocol  --upstream example.com:80 --proxy-connect-timeout 3s --dump-tcp 2

{"level":"info","ts":1551234616.309875,"caller":"ppdp/ppdp.go:70","msg":"Start listen","listen":"0.0.0.0:3000"}
{"level":"info","ts":1551234632.877315,"caller":"proxy/proxy.go:76","msg":"log","listener":"[::]:3000","remote-addr":"127.0.0.1:57915","status":"Connected"}
{"level":"info","ts":1551234633.1970098,"caller":"dumper/dumper.go:80","msg":"dump","listener":"[::]:3000","remote-addr":"127.0.0.1:57915","upstream":"93.184.216.34:80","direction":1,"hex":"47 45 54 20 2f 20 48 54 54 50 2f 31 2e 31 0d 0a 48 6f 73 74 3a 20 6c 6f 63 61 6c 68 6f 73 74 3a 33 30 30 31 0d 0a 55 73 65 72 2d 41 67 65 6e 74 3a 20 63 75 72 6c 2f 37 2e 35 34 2e 30 0d 0a 41 63 63 65 70 74 3a 20 2a 2f 2a 0d 0a 0d 0a","ascii":"GET / HTTP/1.1..Host: localhost:3001..User-Agent: curl/7.54.0..Accept: */*...."}
{"level":"info","ts":1551234633.197132,"caller":"dumper/dumper.go:80","msg":"dump","listener":"[::]:3000","remote-addr":"127.0.0.1:57915","upstream":"93.184.216.34:80","direction":2,"hex":"48 54 54 50 2f 31 2e 31 20 34 30 34 20 4e 6f 74 20 46 6f 75 6e 64 0d 0a 43 6f 6e 74 65 6e 74 2d 54 79 70 65 3a 20 74 65 78 74 2f 68 74 6d 6c 0d 0a 44 61 74 65 3a 20 57 65 64 2c 20 32 37 20 46 65 62 20 32 30 31 39 20 30 32 3a 33 30 3a 33 33 20 47 4d 54 0d 0a 53 65 72 76 65 72 3a 20 45 43 53 20 28 73 6a 63 2f 34 45 34 36 29 0d 0a 43 6f 6e 74 65 6e 74 2d 4c 65 6e 67 74 68 3a 20 33 34 35 0d 0a 0d 0a 3c 3f 78 6d 6c 20 76 65 72 73 69 6f 6e 3d 22 31 2e 30 22 20 65 6e 63 6f 64 69 6e 67 3d 22 69 73 6f 2d 38 38 35 39 2d 31 22 3f 3e 0a 3c 21 44 4f 43 54 59 50 45 20 68 74 6d 6c 20 50 55 42 4c 49 43 20 22 2d 2f 2f 57 33 43 2f 2f 44 54 44 20 58 48 54 4d 4c 20 31 2e 30 20 54 72 61 6e 73 69 74 69 6f 6e 61 6c 2f 2f 45 4e 22 0a 20 20 20 20 20 20 20 20 20 22 68 74 74 70 3a 2f 2f 77 77 77 2e 77 33 2e 6f 72 67 2f 54 52 2f 78 68 74 6d 6c 31 2f 44 54 44 2f 78 68 74 6d 6c 31 2d 74 72 61 6e 73 69 74 69 6f 6e 61 6c 2e 64 74 64 22 3e 0a 3c 68 74 6d 6c 20 78 6d 6c 6e 73 3d 22 68 74 74 70 3a 2f 2f 77 77 77 2e 77 33 2e 6f 72 67 2f 31 39 39 39 2f 78 68 74 6d 6c 22 20 78 6d 6c 3a 6c 61 6e 67 3d 22 65 6e 22 20 6c 61 6e 67 3d 22 65 6e 22 3e 0a 09 3c 68 65 61 64 3e 0a 09 09 3c 74 69 74 6c 65 3e 34 30 34 20 2d 20 4e 6f 74 20 46 6f 75 6e 64 3c 2f 74 69 74 6c 65 3e 0a 09 3c 2f 68 65 61 64 3e 0a 09 3c 62 6f 64 79 3e 0a 09 09 3c 68 31 3e 34 30 34 20 2d 20 4e 6f 74 20 46 6f 75 6e 64 3c 2f 68 31 3e 0a 09 3c 2f 62 6f 64 79 3e 0a 3c 2f 68 74 6d 6c 3e 0a","ascii":"HTTP/1.1 404 Not Found..Content-Type: text/html..Date: Wed, 27 Feb 2019 02:30:33 GMT..Server: ECS (sjc/4E46)..Content-Length: 345....<?xml version=\"1.0\" encoding=\"iso-8859-1\"?>.<!DOCTYPE html PUBLIC \"-//W3C//DTD XHTML 1.0 Transitional//EN\".         \"http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd\">.<html xmlns=\"http://www.w3.org/1999/xhtml\" xml:lang=\"en\" lang=\"en\">..<head>...<title>404 - Not Found</title>..</head>..<body>...<h1>404 - Not Found</h1>..</body>.</html>."}
{"level":"info","ts":1551234633.197245,"caller":"proxy/proxy.go:115","msg":"log","listener":"[::]:3000","remote-addr":"127.0.0.1:57915","upstream":"93.184.216.34:80","status":"Suceeded","read":78,"write":478}
```
