package exporters

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"github.com/alonana/httshark/core"
	"github.com/alonana/httshark/har"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/sirupsen/logrus"
	"strconv"
	"sync"
	"time"
)

type Reason int

const (
	Size Reason = iota
	Time
)

func (r Reason) String() string {
	return []string{"S", "T"}[r]
}

type S3Client struct {
	s3Service  *s3.S3
	dataHolder map[string][]har.Entry
	mutex      sync.Mutex
	timer      *time.Ticker
	Logger     *logrus.Logger
}

func (s *S3Client) init()  {
	s.s3Service = s3.New(session.Must(session.NewSession(&aws.Config{DisableSSL: aws.Bool(core.Config.AWSDisableSSL),
		Region: &core.Config.AWSRegion})))
	s.dataHolder = make(map[string][]har.Entry)
	s.timer = time.NewTicker(core.Config.S3ExporterPurgeInterval)
	for {
		select {
		case <-s.timer.C:
			err := s.doExportWrapper(Time)
			if err != nil {
				//TODO -report as severe error
			}
		}
	}
}

func (s *S3Client) Process(harData *har.Har) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	appId := harData.Log.Entries[0].GetAppId()
	// empty app ids in order to remove them from JSON
	for _, entry := range harData.Log.Entries {
		entry.Request.AppId.Empty()
		entry.Request.AppId = nil
	}

	currentEntries := s.dataHolder[appId]
	for _, entry := range harData.Log.Entries {
		currentEntries = append(currentEntries, entry)
	}
	s.dataHolder[appId] = currentEntries
	numOfEntries := s.getNumOfEntries()
	if numOfEntries > core.Config.S3ExporterMaxNumOfEntries {
		err := s.doExport(numOfEntries,Size)
		if err != nil {
			return fmt.Errorf("failed to export har data: %v", err)
		}
	}
	return nil
}

func getFileName(entriesCount int,reason Reason) string {
	gzipExt := ""
	if core.Config.S3ExporterShouldCompress {
		gzipExt = ".gzip"
	}
	return fmt.Sprintf("%s__%s__%s__%s__%s.har%s",core.Config.DCVAName,
		strconv.FormatInt(int64(core.Config.InstanceId),10),
		strconv.FormatInt(int64(entriesCount),10),
		reason.String(),
		strconv.FormatInt(time.Now().UnixNano(),10),
		gzipExt)
}

func compress(data []byte,fileName string) ([]byte,error) {
	var buffer bytes.Buffer
	gz,_ := gzip.NewWriterLevel(&buffer,gzip.BestSpeed)
	if _, err := gz.Write(data); err != nil {
		return nil,fmt.Errorf("Failed to gzip(write) HAR data before sending it to bucket %s, object %s, %s\n",
			core.Config.S3ExporterBucketName, fileName, err.Error())
	}
	if err := gz.Close(); err != nil {
		return nil,fmt.Errorf("Failed to gzip(close) HAR data before sending it to bucket %s, object %s, %s\n",
			core.Config.S3ExporterBucketName, fileName, err.Error())
	}
	data = buffer.Bytes()
	return data,nil

}

func (s *S3Client) pushToS3(data []byte,fileName string) error {
	_, err := s.s3Service.PutObject(&s3.PutObjectInput{
		Body: bytes.NewReader(data),
		Bucket: &core.Config.S3ExporterBucketName,
		Key:    &fileName,
	})
	data = nil
	if err != nil {
		return fmt.Errorf("Failed to upload HAR data to bucket %s, object %s, %s\n",
			core.Config.S3ExporterBucketName, fileName, err.Error())
	}
	return nil
}

func (s *S3Client) doExport(numOfEntries int,reason Reason) error {
	if numOfEntries > 0 {
		data, err := json.Marshal(s.dataHolder)
		if err != nil {
			return fmt.Errorf("marshal har failed: %v", err)
		}
		fileName := getFileName(numOfEntries,reason)
		if core.Config.S3ExporterShouldCompress {
			compressedData, err := compress(data, fileName)
			if err != nil {
				return fmt.Errorf("compress har failed: %v", err)
			}
			err = s.pushToS3(compressedData, fileName)
			if err != nil {
				return fmt.Errorf("push har to s3 failed: %v", err)
			}
		} else {
			err = s.pushToS3(data, fileName)
			if err != nil {
				return fmt.Errorf("push har to s3 failed: %v", err)
			}
		}
		s.dataHolder = make(map[string][]har.Entry)
		return nil
	} else {
		return nil
	}
}

func (s *S3Client) getNumOfEntries() int {
	numOfEntries := 0
	for _, entries := range s.dataHolder {
		numOfEntries += len(entries)
	}
	return numOfEntries
}

func (s *S3Client) doExportWrapper(reason Reason) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	err := s.doExport(s.getNumOfEntries(),reason)
	return err
}

