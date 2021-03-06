package main

import (
	"context"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	cf "github.com/aws/aws-sdk-go/service/cloudformation"
	cfAPI "github.com/aws/aws-sdk-go/service/cloudformation/cloudformationiface"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"os"
	"time"
)

const LOCAL_BUILD_VERSION = "snapshot"

var version = LOCAL_BUILD_VERSION

func main() {
	log.SetFormatter(&log.TextFormatter{})
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)

	app := cli.NewApp()
	app.Name = "Stack Terminator"
	app.Version = version
	app.Authors = []cli.Author{
		{
			Name:  "Carlos Roman",
			Email: "carlosr@cliche-corp.co.uk",
		},
	}

	var region string
	var timeout time.Duration
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "aws-region",
			Usage:       "the ID of AWS Region stack in, e.g. us-east-1",
			EnvVar:      "AWS_REGION",
			Value:       "us-east-1",
			Destination: &region,
		},
		cli.DurationFlag{
			Name:        "timeout",
			Usage:       "the time out duration for AWS calls, e.g. 3000ms",
			Destination: &timeout,
		},
	}

	app.Commands = []cli.Command{
		{
			Name:    "delete",
			Aliases: []string{"d"},
			Usage:   "Delete a CloudFormation stack",
			Action: func(c *cli.Context) error {
				sess := session.Must(session.NewSession(&aws.Config{
					Region: aws.String(region),
				}))
				cfsvc := cf.New(sess)
				s3svc := s3.New(sess)
				ctx := context.Background()
				var cancelFn func()
				if timeout > 0 {
					ctx, cancelFn = context.WithTimeout(ctx, timeout)
					defer cancelFn()
				}
				return Terminate(ctx, c.Args().First(), cfsvc, s3svc, 5)
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

type item struct {
	Key       *string
	VersionId *string
}

func deleteItems(ctx context.Context, svc s3iface.S3API, bucket *string, objs *[]item) error {

	if len(*objs) == 0 {
		return nil
	}

	dels := make([]*s3.ObjectIdentifier, len(*objs))
	for i, obj := range *objs {
		//obj := o.(item)
		log.Info("obj:", aws.StringValue(obj.Key), ",", aws.StringValue(obj.VersionId))
		key := *obj.Key
		versionId := *obj.VersionId
		dels[i] = &s3.ObjectIdentifier{
			Key:       aws.String(key),
			VersionId: aws.String(versionId),
		}
	}

	res, err := svc.DeleteObjectsWithContext(ctx, &s3.DeleteObjectsInput{
		Bucket: bucket,
		Delete: &s3.Delete{
			Objects: dels,
		},
	})
	log.Info(res)
	return err
}

//
//func convertDeleteMarkers(objs []*s3.DeleteMarkerEntry) *[]interface{} {
//	is := make([]interface{}, len(objs))
//	for i, o := range objs {
//		is[i] = o
//	}
//	return &is
//}
//
//func convertObjectVersion(objs []*s3.ObjectVersion) *[]interface{} {
//	is := make([]interface{}, len(objs))
//	for i, o := range objs {
//		is[i] = o
//	}
//	return &is
//}

func convertDeleteMarkers(objs []*s3.DeleteMarkerEntry) []item {
	is := make([]item, len(objs))
	for i, o := range objs {
		is[i] = item{o.Key, o.VersionId}
	}
	return is
}

func convertObjectVersion(objs []*s3.ObjectVersion) []item {
	is := make([]item, len(objs))
	for i, o := range objs {
		is[i] = item{o.Key, o.VersionId}
	}
	return is
}

func deleteS3Content(ctx context.Context, r *cf.StackResource, s3svc s3iface.S3API, maxKeys int64) error {
	log.Debug(r)

	objs, err := s3svc.ListObjectVersionsWithContext(ctx, &s3.ListObjectVersionsInput{
		Bucket:  r.PhysicalResourceId,
		MaxKeys: aws.Int64(maxKeys),
	})

	if err != nil {
		return err
	}

	ds := append(convertObjectVersion(objs.Versions), convertDeleteMarkers(objs.DeleteMarkers)...)
	if err := deleteItems(ctx, s3svc, r.PhysicalResourceId, &ds); err != nil {
		return err
	}

	vim := aws.StringValue(objs.NextVersionIdMarker)
	km := aws.StringValue(objs.NextKeyMarker)

	for {
		log.Info("vim:", vim, ",km:", km)
		if (len(objs.Versions) < 1) && (len(objs.DeleteMarkers) < 1) {
			return err
		}
		objs, err := s3svc.ListObjectVersionsWithContext(ctx, &s3.ListObjectVersionsInput{
			Bucket:          r.PhysicalResourceId,
			MaxKeys:         aws.Int64(maxKeys),
			KeyMarker:       aws.String(km),
			VersionIdMarker: aws.String(vim),
		})
		log.Info(objs)
		if err != nil {
			return err
		}

		if (len(objs.Versions) < 1) && (len(objs.DeleteMarkers) < 1) {
			return err
		}

		ds := append(convertObjectVersion(objs.Versions), convertDeleteMarkers(objs.DeleteMarkers)...)
		if err := deleteItems(ctx, s3svc, r.PhysicalResourceId, &ds); err != nil {
			return err
		}

		if len(aws.StringValue(objs.NextKeyMarker)) < 1 {
			break
		}
		km = aws.StringValue(objs.NextKeyMarker)
		vim = aws.StringValue(objs.NextVersionIdMarker)
	}
	return err
}

func Terminate(ctx context.Context, stackName string, cfsvc cfAPI.CloudFormationAPI, s3svc s3iface.S3API, maxKeys int64) error {
	log.Info("About to try and delete stack '", stackName, "'")

	res, err := cfsvc.DescribeStackResourcesWithContext(ctx, &cf.DescribeStackResourcesInput{
		StackName: aws.String(stackName),
	})

	if err != nil {
		return err
	}

	for _, r := range res.StackResources {

		switch rt := *r.ResourceType; rt {
		case "AWS::S3::Bucket":
			err = deleteS3Content(ctx, r, s3svc, maxKeys)
		default:
			log.Info("Ignoring resource : ", rt)
		}

		if err != nil {
			return err
		}
	}

	del, err := cfsvc.DeleteStackWithContext(ctx, &cf.DeleteStackInput{
		StackName: aws.String(stackName),
	})

	if err != nil {
		return err
	}

	log.Info(del)
	return err
}
