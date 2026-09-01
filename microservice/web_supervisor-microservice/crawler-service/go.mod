module github.com/totooicu/crawler-service

go 1.25.0

replace github.com/totooicu/go-mytool => ../../../MyTool

require (
	github.com/sirupsen/logrus v1.10.2
	github.com/totooicu/go-mytool v0.0.0-00010101000000-000000000000
)

require (
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/go-redis/redis/v8 v8.11.5 // indirect
	golang.org/x/sys v0.13.0 // indirect
)
