go-gslb:: generate
	go build
	
	
generate:
	rm templated_*.go
	go generate *_template.go
		