package awsapi

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/aws/aws-k8s-tester/pkg/fileutil"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
	"go.uber.org/zap"
	"k8s.io/client-go/util/homedir"
)

// Config defines a top-level AWS API configuration to create a session.
type Config struct {
	// Logger is the log object.
	Logger *zap.Logger

	// DebugAPICalls is true to log all AWS API call debugging messages.
	DebugAPICalls bool

	// Region is a separate AWS geographic area for EKS service.
	// Each AWS Region has multiple, isolated locations known as Availability Zones.
	Region string

	// ResolverURL is a custom resolver URL.
	ResolverURL string
	// SigningName is the API signing name.
	SigningName string
}

func OscEndpoint(region string, service string) (string) {
    return "https://" + service + "." + region + ".outscale.com"
}


// New creates a new AWS session.
// Specify a custom endpoint for tests.
func New(cfg *Config) (ss *session.Session, stsOutput *sts.GetCallerIdentityOutput, awsCredsPath string, err error) {
	if cfg == nil {
		return nil, nil, "", errors.New("got empty config")
	}
	if cfg.Logger == nil {
		return nil, nil, "", fmt.Errorf("missing logger")
	}
	if cfg.Region == "" {
		return nil, nil, "", fmt.Errorf("missing region")
	}

	awsConfig := aws.Config{
		Region:                        aws.String(cfg.Region),
		CredentialsChainVerboseErrors: aws.Bool(true),
		Logger:                        toLogger(cfg.Logger),
	}
	awsConfig.WithRegion("eu-west-2")

	// Credential is the path to the shared credentials file.
	//
	// If empty will look for "AWS_SHARED_CREDENTIALS_FILE" env variable. If the
	// env value is empty will default to current user's home directory.
	// Linux/OSX: "$HOME/.aws/credentials"
	// Windows:   "%USERPROFILE%\.aws\credentials"
	//
	// See https://godoc.org/github.com/aws/aws-sdk-go/aws/credentials#SharedCredentialsProvider.
	// See https://godoc.org/github.com/aws/aws-sdk-go/aws/session#hdr-Environment_Variables.

	awsCredsPath = filepath.Join(homedir.HomeDir(), ".aws", "credentials")
	if os.Getenv("AWS_SHARED_CREDENTIALS_FILE") != "" {
		awsCredsPath = os.Getenv("AWS_SHARED_CREDENTIALS_FILE")
	}
	if fileutil.Exist(awsCredsPath) {
		cfg.Logger.Info("creating session from AWS cred file", zap.String("path", awsCredsPath))
		// TODO: support temporary credentials with refresh mechanism
	} else {
		cfg.Logger.Info("cannot find AWS cred file", zap.String("path", awsCredsPath))
		if os.Getenv("AWS_ACCESS_KEY_ID") == "" ||
			os.Getenv("AWS_SECRET_ACCESS_KEY") == "" {
			return nil, nil, "", errors.New("cannot find AWS credentials")
		}
		cfg.Logger.Info("creating session from env vars")
	}

	if cfg.DebugAPICalls {
		lvl := aws.LogDebug |
			aws.LogDebugWithEventStreamBody |
			aws.LogDebugWithHTTPBody |
			aws.LogDebugWithRequestRetries |
			aws.LogDebugWithRequestErrors
		awsConfig.LogLevel = &lvl
	}


    if os.Getenv("OSC_ACCOUNT_IAM") == "" ||
       os.Getenv("OSC_USER_ID") == ""  ||
       os.Getenv("OSC_ARN")  == "" {
	        return nil, nil, "", errors.New("cannot find OSC IAM credentials")
    }

	stsOutput = new(sts.GetCallerIdentityOutput)
	stsOutput.Account = new(string)
	stsOutput.UserId = new(string)
	stsOutput.Arn = new(string)

	*stsOutput.Account = os.Getenv("OSC_ACCOUNT_IAM")
	*stsOutput.UserId = os.Getenv("OSC_USER_ID")
	*stsOutput.Arn = os.Getenv("OSC_ARN")
	cfg.Logger.Info(
		"creating AWS session",
		zap.String("account-id", *stsOutput.Account),
		zap.String("user-id", *stsOutput.UserId),
		zap.String("arn", *stsOutput.Arn),
	)
	awsConfig.EndpointResolver = endpoints.ResolverFunc(
        func(service, region string, optFns ...func(*endpoints.Options))(endpoints.ResolvedEndpoint, error) {
            supported_service := map[string]string  {
                endpoints.Ec2ServiceID:                    "fcu",
                endpoints.ElasticloadbalancingServiceID:   "lbu",
                endpoints.IamServiceID:                    "eim",
                endpoints.DirectconnectServiceID:          "directlink",
            }
            var osc_service string
            var ok bool
            if osc_service, ok =  supported_service[service]; ok {
                return endpoints.ResolvedEndpoint{
                        URL:           OscEndpoint(region, osc_service),
                        SigningRegion: region,
                        SigningName:   service,
                }, nil
            } else {
                return endpoints.DefaultResolver().EndpointFor(service, region, optFns...)
            }
    })
	ss, err = session.NewSession(&awsConfig)
	if err != nil {
		return nil, nil, "", err
	}
	return ss, stsOutput, awsCredsPath, err
}
