# lambda-go-sam-s3

This is SAM template for lambda golang 1.x

## Instalation

1. Edit Globals and Parameters section in [template.yaml]

2. Build SAM

```
AWS_PROFILE=<AWS profile> AWS_DEFAULT_REGION=<AWS default region> \
  S3_BUCKET=<SAM bucket name> make build
```

3. Deploy SAM

```
AWS_PROFILE=<AWS profile> AWS_DEFAULT_REGION=<AWS default region> \
  S3_BUCKET=<SAM S3 bucket name> STACK_NAME=<SAM cfn stack name> make deploy
```

4. Add Lambda Permission

```
EXTERNAL_S3_BUCKET=<External S3 Bucket> EXTERNAL_S3_ACCOUNT_ID=<External S3 Account ID> \
  AWS_PROFILE=<AWS profile> AWS_REGION=<AWS default region> \
  S3_BUCKET=<SAM S3 bucket name> STACK_NAME=<STAM cfn stack name> add_permission
```


## About this SAM

1. Trigger S3 event `All object create events`

2. Launch Lambda function


### About Lambda function

1. Get ALB access log from S3 bucket

2. Parse  ALB access logs the service login url

3. Post datadog metric - metric name: `test.metric.login`
