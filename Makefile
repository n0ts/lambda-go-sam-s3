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
	@sam deploy --template-file out.yaml --stack-name $(STACK_NAME) \
	  --capabilities CAPABILITY_IAM

.PHONY: add_permission
add_permission: # Add Permission Lambda Function
	@aws --output table --region $(AWS_REGION) lambda add-permission \
	  --function-name $(shell aws --region $(AWS_REGION) --output text \
	                    cloudformation describe-stacks --stack-name $(STACK_NAME) \
	  --query 'Stacks[0].Outputs[?OutputKey==`S3GetFunctionName`].OutputValue') \
	  --statement-id s3-account --principal s3.amazonaws.com \
	  --action lambda:InvokeFunction \
	  --source-arn arn:aws:s3:::$(EXTERNAL_S3_BUCKET) \
	  --source-account $(EXTERNAL_S3_ACCOUNT_ID)

.PHONY: build_deploy
build_deploy: # Build and Deploy
	@$(MAKE) build
	@$(MAKE) deploy

.PHONY: destroy
destroy: # Destroy
	@aws cloudformation delete-stack --stack-name $(STACK_NAME)

.PHONY: fmt
fmt: # Fmt
	@go fmt *.go

.PHONY: clean
clean: # Clean
	@rm -f out.yaml main main.zip

.PHONY: help
help: # Show usage
	@echo 'Available targets are:'
	@grep -E '^[a-zA-Z_-]+:.*?# .*$$' $(MAKEFILE_LIST) | \
	awk 'BEGIN {FS = ":.*?# "}; {printf "  \033[36m%-8s\033[0m %s\n", $$1, $$2}'
