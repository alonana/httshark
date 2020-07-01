package log

import (
	"fmt"
	"github.com/alonana/httshark/core"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/sirupsen/logrus"
	"os"
	"runtime"
	"strings"
)

var Logger *logrus.Logger

func NewLogger() *logrus.Logger {

	var level logrus.Level
	//  TODO: read from config
	level, err := logrus.ParseLevel("info")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	formatter := &logrus.TextFormatter{
		TimestampFormat:        "02-01-2006 15:04:05",
		FullTimestamp:          true,
		DisableColors: true,
		DisableLevelTruncation: true, // log level field configuration
		CallerPrettyfier: func(f *runtime.Frame) (string, string) {
			return "", fmt.Sprintf("%s:%d", formatFilePath(f.File), f.Line)
		},
	}

	logger := &logrus.Logger{
		Out:   os.Stdout,
		Level: level,
		Hooks: make(logrus.LevelHooks),
		Formatter: formatter,
	}
	logger.SetReportCaller(true)
	fileName := fmt.Sprintf("/var/log/httshark_%v.log",core.Config.InstanceId)


	rotateFileHook, err := NewRotateFileHook(RotateFileConfig{Filename: fileName,
		MaxSize: core.Config.RotateFileMaxSize, MaxBackups: core.Config.RotateFileMaxBackups,
		MaxAge: core.Config.RotateFileMaxAge,Level: logrus.DebugLevel,Formatter: formatter})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	logger.AddHook(rotateFileHook)
	if core.Config.UseCloudWatchLoggerHook {
		session := session.Must(session.NewSession(&aws.Config{DisableSSL: aws.Bool(core.Config.AWSDisableSSL),
			Region: &core.Config.AWSRegion,CredentialsChainVerboseErrors: aws.Bool(true)}))
		streamName := fmt.Sprintf("%v_%v",core.Config.DCVAName,os.Getpid())
		cloudWatchHook, err := NewBatchingHook("jordan_river_log_group", streamName, session, 15)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		logger.AddHook(cloudWatchHook)
	}


	Logger = logger
	return Logger
}

func formatFilePath(path string) string {
	arr := strings.Split(path, "/")
	return arr[len(arr)-1]
}
