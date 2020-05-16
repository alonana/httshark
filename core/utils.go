package core
import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"log"
	"time"
	"os"
)


type AWSCloudWatchClient struct {
	watchService *cloudwatch.CloudWatch
}

var  CloudWatchClient  = AWSCloudWatchClient{}

func (c AWSCloudWatchClient)PutMetric(metricName string, unitName string, metricValue float64, namespace string) {
	if c.watchService == nil {
		c.watchService = cloudwatch.New(session.Must(session.NewSession(&aws.Config{DisableSSL: aws.Bool(true),
			Region: &Config.AWSRegion})))
	}
	dcvaName,ok := os.LookupEnv("DCVA_NAME")
	if !ok {
		dcvaName = "UnknownDCVA"
	}
	params := &cloudwatch.PutMetricDataInput{
		MetricData: []*cloudwatch.MetricDatum{
			&cloudwatch.MetricDatum{
				MetricName: aws.String(metricName),
				Timestamp:  aws.Time(time.Now()),
				Unit:       aws.String(unitName),
				Value:      aws.Float64(metricValue),
				Dimensions: []*cloudwatch.Dimension{
					&cloudwatch.Dimension{
						Name:  aws.String(metricName),
						Value: aws.String(dcvaName),
					},
				},
			},
		},
		Namespace: aws.String(namespace),
	}

	_, err := c.watchService.PutMetricData(params)
	if err != nil {
		log.Printf("Failure to put cloudwatch metric: %s", err)
	}
}

