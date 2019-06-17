package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

// regexpAlb - Regular expression for ALB access log format
// https://docs.aws.amazon.com/elasticloadbalancing/latest/application/load-balancer-access-logs.html#access-log-entry-format
const regexpAlb = `(?P<type>.*?)\s(?P<timestamp>.*?)\s(?P<elb>.*?)\s(?P<client_port>.*?)\s(?P<target_port>.*?)\s(?P<request_processing_time>.*?)\s(?P<target_processing_time>.*?)\s(?P<response_processing_time>.*?)\s(?P<elb_status_code>.*?)\s(?P<target_status_code>.*?)\s(?P<received_bytes>.*?)\s(?P<send_bytes>.*?)\s"(?P<request>.*?)"\s"(?P<user_agent>.*?)"\s(?P<ssl_cipher>.*?)\s(?P<ssl_procotol>.*?)\s(?P<target_group_arn>.*?)\s"(?P<trace_id>.*?)"\s"(?P<domain_name>.*?)"\s"(?P<chosen_cert_an>.*?)"\s(?P<matched_rule_priority>.*?)\s(?P<request_creation_time>.*?)\s"(?P<actions_executed>.*?)"\s"(?P<redirect_url>.*?)"\s"(?P<error_reason>.*?)"`

// regexpURL - Regular expression for service url
const regexpURL = "^https://(.*):[^/]*/([^/]*)/?([^/]*)"

// ddMetricName - Datadog post metric name
const ddMetricName = "test.metric"

// ddParam - Datadog API parameter
type ddParam struct {
	Metric string      `json:"metric"`
	Points [1][2]int64 `json:"points"`
	Type   string      `json:"type"`
	Host   string      `json:"host"`
	Tags   []string    `json:"tags"`
}

// ddParams - Datadog API parameters
type ddParams struct {
	Series []ddParam `json:"series"`
}

// ReadGzFile - Read gzipped file
func ReadGzFile(filename string) ([]byte, error) {
	fi, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer fi.Close()

	fz, err := gzip.NewReader(fi)
	if err != nil {
		return nil, err
	}
	defer fz.Close()

	s, err := ioutil.ReadAll(fz)
	if err != nil {
		return nil, err
	}
	return s, nil
}

// Groupmap - Mapping group name
func Groupmap(s string, r *regexp.Regexp) map[string]string {
	values := r.FindStringSubmatch(s)
	keys := r.SubexpNames()

	d := make(map[string]string)
	for i := 1; i < len(keys); i++ {
		d[keys[i]] = values[i]
	}
	return d
}

// Tag - Return tag string
func Tag(k string, v string) string {
	return fmt.Sprintf("%s:%s", k, v)
}

// PostMetric - Post metric to datadog
func PostMetric(m string, t int64, c string) (int, []byte, error) {
	ddParams := ddParams{Series: []ddParam{
		{
			Metric: m,
			Points: [1][2]int64{
				{t, int64(1)},
			},
			Type: "count",
			Host: os.Getenv("DD_HOST"),
			Tags: []string{
				Tag("company", c),
			},
		},
	}}
	jsonValue, _ := json.Marshal(ddParams)
	url := fmt.Sprintf("https://api.datadoghq.com/api/v1/series?api_key=%s",
		os.Getenv("DD_API_KEY"))
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, errClient := client.Do(req)
	if errClient != nil {
		return 500, nil, errClient
	}
	defer resp.Body.Close()

	body, errReadAll := ioutil.ReadAll(resp.Body)
	if errReadAll != errReadAll {
		return 500, nil, errReadAll
	}

	return resp.StatusCode, body, nil
}

// AssumeRoleWithSession - Assume role
func AssumeRoleWithSession(sess *session.Session, roleArn string) *session.Session {
	sCreds := stscreds.NewCredentials(sess, roleArn)
	sConfig := aws.Config{Region: sess.Config.Region, Credentials: sCreds}
	sSess := session.New(&sConfig)
	return sSess
}

// handler - Lambda handler
func handler(ctx context.Context, s3Event events.S3Event) (string, error) {
	errCount := 0
	_, debug := os.LookupEnv("DEBUG")

	configs := map[string]*aws.Config{}
	if debug {
		configs["config"] = aws.NewConfig().WithLogLevel(
			aws.LogDebugWithRequestRetries |
				aws.LogDebugWithRequestErrors |
				aws.LogDebugWithHTTPBody,
		)
	} else {
		configs["config"] = aws.NewConfig()
	}

	sess, _ := session.NewSession(configs["config"])
	sessAssume := AssumeRoleWithSession(sess, os.Getenv("AWS_ASSUME_ROLE"))

	for _, records := range s3Event.Records {
		record := records.S3
		fmt.Printf("[INFO] EventSource = %s, EventTime = %s, Bucket = %s, Key = %s \n",
			records.EventSource, records.EventTime,
			record.Bucket.Name, record.Object.Key)

		fileName := fmt.Sprintf("/tmp/%s", filepath.Base(record.Object.Key))
		logFile, errOsCreate := os.Create(fileName)
		if errOsCreate != nil {
			fmt.Println("[ERROR] os.Create: ", errOsCreate)
			errCount++
			continue
		}
		fmt.Printf("[INFO] FileName = %s \n", fileName)

		defer logFile.Close()

		downloader := s3manager.NewDownloader(sessAssume)
		numBytes, errDownload := downloader.Download(logFile,
			&s3.GetObjectInput{
				Bucket: aws.String(record.Bucket.Name),
				Key:    aws.String(record.Object.Key),
			})
		if errDownload != nil {
			fmt.Println("[ERROR] Download: ", errDownload.Error())
			errCount++
			continue
		}
		fmt.Printf("[INFO] FileName = %s, Bucket = %s, Key = %s, Bytes = %d \n",
			fileName, record.Bucket.Name, record.Object.Key, numBytes)
		logs, errReadGzFile := ReadGzFile(fileName)
		if errReadGzFile != nil {
			fmt.Println("[ERROR] ReadGzFile: ", errReadGzFile)
			errCount++
			continue
		}

		scanner := bufio.NewScanner(strings.NewReader(string(logs)))
		for scanner.Scan() {
			re := regexp.MustCompile(regexpAlb)
			log := Groupmap(scanner.Text(), re)

			req := strings.Split(log["request"], " ")
			method := req[0]
			url := req[1]
			reURL, _ := regexp.Compile(regexpURL)
			submatches := reURL.FindStringSubmatch(url)
			if submatches == nil {
				if debug {
					fmt.Printf("[DEBUG] %s request uri is not url - %s \n", log["timestamp"], url)
				}
				continue
			}

			if method != "POST" {
				if debug {
					fmt.Printf("[DEBUG] Method is not POST - %s \n", method)
				}
				continue
			}

			fmt.Printf("[INFO] Timestamp = %s, Method = %s, Url = %s, Status code = %s, Company = %s \n",
				log["timestamp"], method, url, log["elb_status_code"], submatches[1])

			time, _ := time.Parse(time.RFC3339, log["timestamp"])
			status, body, errPostMetric := PostMetric(fmt.Sprintf("%s.%s", ddMetricName, submatches[1]), time.Unix(), submatches[2])
			if errPostMetric != nil {
				fmt.Println("[ERROR] PostMetric: ", errPostMetric)
				errCount++
			}
			if status != 202 {
				fmt.Println("[ERROR] PostMetric status code is not 202: ", status)
				errCount++
			}

			fmt.Printf("[INFO] Post datadog metric %s - %d %s \n", ddMetricName, status, body)
		}
	}

	if 0 < errCount {
		return "Failure", fmt.Errorf("[ERROR] Count: %d", errCount)
	}
	return "Success", nil
}

func main() {
	lambda.Start(handler)
}
