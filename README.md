# lambda-go-sam-s3

This is SAM template for lambda golang 1.x

## Instalation

1. Edit Globals and Parameters section in [template.yaml]

2. Build SAM

```
AWS_PROFILE=<AWS profile> AWS_DEFAULT_REGION=<AWS default region> S3_BUCKET=<SAM bucket name> make build
```

3. Deploy SAM

```
AWS_PROFILE=<AWS profile> AWS_DEFAULT_REGION=<AWS default region> S3_BUCKET=<SAM bucket name> STACK_NAME=<SAM stack name> make deploy
```


## About this SAM

1. Trigger S3 event `All object create events`

2. Launch Lambda function


### About Lambda function

1. Get ALB access log from S3

2. Parse ALB access log

3. Post datadog metric
