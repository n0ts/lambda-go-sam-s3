.DEFAULT_GOAL := help

AWS_REGION ?= us-east-1
export AWS_DEFAULT_REGION ?= $(AWS_REGION)

.PHONY: deps
deps:
	@go get -u github.com/aws/aws-lambda-go/events
	@go get -u github.com/aws/aws-lambda-go/lambda
	@go get -u github.com/aws/aws-sdk-go

.PHONY: build
build: deps # Build
	@GOOS=linux GOARCH=amd64 go build -o main main.go
	@zip main.zip main
	@sam package --output-template-file out.yaml --s3-bucket $(S3_BUCKET)

.PHONY: deploy
deploy: # Deploy
	@sam deploy --template-file out.yaml --stack-name $(STACK_NAME)

.PHONY: build_deploy
build_deploy: # Build and Deploy
	@$(MAKE) build
	@$(MAKE) deploy

.PHONY: destroy
destroy: # Destroy
	@aws cloudformation delete-stack --stack-name $(STACK_NAME)

.PHONY: clean
clean: # Clean
	@rm -f out.yaml main main.zip

.PHONY: help
help: # Show usage
	@echo 'Available targets are:'
	@grep -E '^[a-zA-Z_-]+:.*?# .*$$' $(MAKEFILE_LIST) | \
	awk 'BEGIN {FS = ":.*?# "}; {printf "  \033[36m%-8s\033[0m %s\n", $$1, $$2}'
