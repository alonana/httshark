package exporters

import (
	"encoding/json"
	"fmt"
	"github.com/alonana/httshark/core"
	"github.com/alonana/httshark/har"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"strconv"
	"strings"
	"time"
)



type S3Client struct {
	s3Service *s3.S3
}

func (s *S3Client) init()  {
	s.s3Service = s3.New(session.Must(session.NewSession(&aws.Config{DisableSSL: aws.Bool(true),
		Region: &core.Config.AWSRegion})))
}

func (s *S3Client) Process(harData *har.Har) error {
	data, err := json.Marshal(harData)
	if err != nil {
		return fmt.Errorf("marshal har failed: %v", err)
	}

	key := fmt.Sprintf("%s.har", strconv.FormatInt(time.Now().UnixNano(),10))
	// TODO gzip the data
	_, err = s.s3Service.PutObject(&s3.PutObjectInput{
		Body:   strings.NewReader(string(data)),
		Bucket: &core.Config.S3ExporterBucketName,
		Key:    &key,
	})
	data = nil
	if err != nil {
		return fmt.Errorf("Failed to upload HAR data to bucket %s, object %s, %s\n", core.Config.S3ExporterBucketName, key, err.Error())
	}
	return nil
}
