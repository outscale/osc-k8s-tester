package awsapi

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"log"

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
	//.WithEndpoint("https://osu.eu-west-2.outscale.com")

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
	log.Println("CI828: New:awsCredsPath", awsCredsPath)

	if cfg.DebugAPICalls {
		lvl := aws.LogDebug |
			aws.LogDebugWithEventStreamBody |
			aws.LogDebugWithHTTPBody |
			aws.LogDebugWithRequestRetries |
			aws.LogDebugWithRequestErrors
		awsConfig.LogLevel = &lvl
	}

	var stsSession *session.Session


	log.Println("CI828: New:awsConfig", awsConfig)
	stsSession, err = session.NewSession(&awsConfig)
	log.Println("CI828: New:session.NewSession", stsSession, err)
	if err != nil {
		return nil, nil, "", err
	}
	log.Println("CI828: New:session.NewSession end ")
	stsSvc := sts.New(stsSession)
	//stsSvc := sts.New(stsSession, aws.NewConfig().WithRegion("eu-west-2").WithEndpoint("https://osu.eu-west-2.outscale.com"))

	log.Println("CI828: New:sts.New  %s", stsSvc)

	//stsOutput, err = stsSvc.GetCallerIdentity(&sts.GetCallerIdentityInput{})
	//log.Println("CI828: New:stsSvc.GetCallerIdentity end ", stsOutput, " ////// ", err)
	//if err != nil {
	//	return nil, nil, "", err
	//}
	stsOutput = new(sts.GetCallerIdentityOutput)
	stsOutput.Account = new(string)
	stsOutput.UserId = new(string)
	stsOutput.Arn = new(string)
	*stsOutput.Account = "awsCloud"
	*stsOutput.UserId = "6YU3EGNHVODO5A9IQPBD9BVLEG5BOE7"
	*stsOutput.Arn = "arn:aws:iam::334617742942:user/awsCloud"
	cfg.Logger.Info(
		"creating AWS session",
		zap.String("account-id", *stsOutput.Account),
		zap.String("user-id", *stsOutput.UserId),
		zap.String("arn", *stsOutput.Arn),
	)
	log.Println("CI828: stsOutput  ", stsOutput)

	resolver := endpoints.DefaultResolver()
	//log.Println("CI828: New:endpoints.DefaultResolver ", resolver)
	log.Println("CI828: New:endpoints.ResolverURL ", cfg.ResolverURL)
	log.Println("CI828: New:endpoints.SigningName ", cfg.SigningName)

	if cfg.ResolverURL != "" && cfg.SigningName == "" {
		return nil, nil, "", fmt.Errorf("got empty signing name for resolver %q", cfg.ResolverURL)
	}
	// support test endpoint (e.g. https://api.beta.us-west-2.wesley.amazonaws.com)
		resolver = endpoints.ResolverFunc(func(service, region string, optFns ...func(*endpoints.Options)) (endpoints.ResolvedEndpoint, error) {
                        if service == endpoints.S3ServiceID {
                                 return endpoints.ResolvedEndpoint{
					 URL:           "https://osu.eu-west-2.outscale.com",
                                       SigningRegion: "eu-west-2",
                                }, nil
                        }
                        if service == endpoints.Ec2ServiceID {
                                 return endpoints.ResolvedEndpoint{
					 URL:           "https://fcu.eu-west-2.outscale.com",
                                       SigningRegion: "eu-west-2",
                                }, nil
                        }
                        if service == endpoints.IamServiceID {
                                 return endpoints.ResolvedEndpoint{
					 URL:           "https://eim.eu-west-2.outscale.com",
                                       SigningRegion: "eu-west-2",
                                 }, nil
                         }

			return endpoints.DefaultResolver().EndpointFor(service, region, optFns...)
		})

	awsConfig.EndpointResolver = resolver
	log.Println("CI828: New:awsConfig ", awsConfig)
	ss, err = session.NewSession(&awsConfig)
	log.Println("CI828: New:session.NewSession.ss ", ss)

	log.Println("CI828:1  ss, stsOutput, awsCredsPath, err ",  ss, stsOutput, awsCredsPath, err)
	if err != nil {
		return nil, nil, "", err
	}
	log.Println("CI828:  ss, stsOutput, awsCredsPath, err ",  ss, stsOutput, awsCredsPath, err)
	return ss, stsOutput, awsCredsPath, err
}
