buf/generate:
	buf generate buf.build/alchematik/athanor

install/athanor:
	cd ../athanor && go install ./cmd/athanor && cd -

go/build:
	go build -o build/go/v0.0.1/translator ./cmd/translator

go/build/provider:
	go build -o build/provider/gcp/v0.0.1/provider ./example/provider

provider/generate: go/build install/athanor
	athanor provider generate ./example/schema/config.json

blueprint/reconcile: go/build install/athanor
	athanor blueprint reconcile ./example/consumer/config.json
