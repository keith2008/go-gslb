go-gslb:: generate
	go build
	
	
generate:
	rm -f templated_*.go
	go generate *_template.go
		