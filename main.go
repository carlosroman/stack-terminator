package main

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	cf "github.com/aws/aws-sdk-go/service/cloudformation"
	cfAPI "github.com/aws/aws-sdk-go/service/cloudformation/cloudformationiface"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"os"
)

func deleteDeleteMarkers(ctx context.Context, svc s3iface.S3API, bucket *string, objs []*s3.DeleteMarkerEntry) error {
	if len(objs) == 0 {
		return nil
	}

	dels := make([]*s3.ObjectIdentifier, len(objs))
	for i, obj := range objs {
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

func deleteObjectVersions(ctx context.Context, svc s3iface.S3API, bucket *string, objs []*s3.ObjectVersion) error {
	if len(objs) == 0 {
		return nil
	}

	dels := make([]*s3.ObjectIdentifier, len(objs))
	for i, obj := range objs {
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

func main() {
	log.SetFormatter(&log.TextFormatter{})
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)

	cli.NewApp()
	fmt.Printf("hello, world\n")
	sess := session.Must(session.NewSession())
	cfsvc := cf.New(sess)
	s3svc := s3.New(sess)
	ctx := context.Background()
	Termainte("bob", ctx, cfsvc, s3svc)
}

func deleteS3Content(ctx context.Context, r *cf.StackResource, s3svc s3iface.S3API) error {
	count := 0
	log.Info("===========================")
	log.Info(r)
	log.Info("===========================")

	objs, err := s3svc.ListObjectVersionsWithContext(ctx, &s3.ListObjectVersionsInput{
		Bucket:  r.PhysicalResourceId,
		MaxKeys: aws.Int64(5),
	})

	if err != nil {
		return err
	}

	if err := deleteObjectVersions(ctx, s3svc, r.PhysicalResourceId, objs.Versions); err != nil {
		return err
	}

	if err := deleteDeleteMarkers(ctx, s3svc, r.PhysicalResourceId, objs.DeleteMarkers); err != nil {
		return err
	}

	count = count + len(objs.Versions)
	vim := aws.StringValue(objs.NextVersionIdMarker)
	km := aws.StringValue(objs.NextKeyMarker)

	for {
		log.Info("vim:", vim, ",km:", km)
		if (len(objs.Versions) < 1) && (len(objs.DeleteMarkers) < 1) {
			return err
		}
		objs, err := s3svc.ListObjectVersionsWithContext(ctx, &s3.ListObjectVersionsInput{
			Bucket:          r.PhysicalResourceId,
			MaxKeys:         aws.Int64(5),
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

		if err := deleteObjectVersions(ctx, s3svc, r.PhysicalResourceId, objs.Versions); err != nil {
			return err
		}

		if err := deleteDeleteMarkers(ctx, s3svc, r.PhysicalResourceId, objs.DeleteMarkers); err != nil {
			return err
		}

		count = count + len(objs.Versions)

		if len(aws.StringValue(objs.NextKeyMarker)) < 1 {
			break
		}
		km = aws.StringValue(objs.NextKeyMarker)
		vim = aws.StringValue(objs.NextVersionIdMarker)
	}
	log.Info("===========================")
	log.Info(count)
	log.Info("===========================")
	return err
}

func Termainte(stackName string, ctx context.Context, cfsvc cfAPI.CloudFormationAPI, s3svc s3iface.S3API) error {

	res, err := cfsvc.DescribeStackResourcesWithContext(ctx, &cf.DescribeStackResourcesInput{
		StackName: aws.String("test-stack"),
	})

	if err != nil {
		return err
	}

	for _, r := range res.StackResources {

		switch rt := *r.ResourceType; rt {
		case "AWS::S3::Bucket":
			err = deleteS3Content(ctx, r, s3svc)
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

	log.Info(del.GoString())
	return err
}
