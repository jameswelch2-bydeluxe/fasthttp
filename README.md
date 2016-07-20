# Fast HTTP Getter

### Build
The library is a go package, which can be built using the following command:

	go install github.com/jameswelch2-bydeluxe/fasthttp

### Import
The package can be imported into a go source file using the following line:

	import "github.com/jameswelch2-bydeluxe/fasthttp"
	
	
### Command line
Package includes a command line tool for testing, and automation outside of the GO envionment. It is located in:

	github.com/jameswelch2-bydeluxe/fasthttp/cmd/fasthttp-get/
	
The command can be invoked with a simple syntax:

	fasthttp-get [-threads <threadcount:1>] url filename

## go-getter wrapper
Package includes a wrapper driver for Hashicorp go-getter. It can be found at:

	github.com/jameswelch2-bydeluxe/fasthttp/go-getter/get_fasthttp.go

### Roadmap

- v0.2 (July/August)
	- Importable library supporting concurrent connections
	- go-getter driver wrapping this library

- Asynchronous API?
- Support for TCP SACK to reduce packet chatter?