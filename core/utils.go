package core
import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"time"
)

const PacketDrop = "Packets received/dropped on interface"
const PacketDropFileName = "/var/log/drooped_packets.txt"
const NAMESPACE = "httshark_stats"


type AWSCloudWatchClient struct {
	watchService *cloudwatch.CloudWatch
}

var  CloudWatchClient  = AWSCloudWatchClient{}

func (c *AWSCloudWatchClient)PutMetric(metricName string, unitName string, metricValue float64, namespace string) (err error)  {
	if c.watchService == nil {
		c.watchService = cloudwatch.New(session.Must(session.NewSession(&aws.Config{DisableSSL: aws.Bool(Config.AWSDisableSSL),
			Region: &Config.AWSRegion})))
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
						Value: aws.String(Config.DCVAName),
					},
				},
			},
		},
		Namespace: aws.String(namespace),
	}
	_, err = c.watchService.PutMetricData(params)
	return  err
}
