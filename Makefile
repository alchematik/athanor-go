buf/generate:
	buf generate buf.build/alchematik/athanor

install/athanor:
	cd ../athanor && go install ./cmd/athanor && cd -

go/build:
	go build -o build/go/v0.0.1/translator ./cmd/translator

provider/generate: go/build install/athanor
	athanor provider generate ./example/schema/config.json

