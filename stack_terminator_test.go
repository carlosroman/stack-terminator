package main_test

import (
	"context"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	cf "github.com/aws/aws-sdk-go/service/cloudformation"
	cfAPI "github.com/aws/aws-sdk-go/service/cloudformation/cloudformationiface"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	. "github.com/carlosroman/stack-terminator"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/mock"
	"os"
)

var _ = Describe("StackTerminator", func() {
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})
	log.SetOutput(os.Stdout)
	log.SetLevel(log.DebugLevel)
	Describe("Should terminate stack", func() {

		ctx := context.Background()
		maxKeys := int64(5)

		Context("when stack deletes succesfully", func() {
			mockCfSvc := &mockCloudFormationClient{ctx: ctx}
			mockS3Svc := &mockS3Client{ctx: ctx}

			mockCfSvc.givenDescribeStackResourcesWithContextReturns(nil)
			mockS3Svc.givenListObjectVersionsWithContextReturns([]item{
				{VersionId: "Vid", Key: "Vk"},
			}, []item{
				{VersionId: "Did", Key: "Dk"},
			},
				nil)
			mockS3Svc.givenDeleteObjectsWithContext(nil)

			mockCfSvc.givenDeleteStackWithContextReturns(nil)

			err := Terminate("bob", ctx, mockCfSvc, mockS3Svc, maxKeys)

			It("should clear the correct S3 bucket", func() {
				Expect(mockS3Svc.AssertCalled(
					GinkgoT(),
					"ListObjectVersionsWithContext",
					ctx,
					&s3.ListObjectVersionsInput{
						Bucket:  aws.String("bob-s3-bucket"),
						MaxKeys: aws.Int64(maxKeys),
					},
					[]request.Option(nil),
				)).To(BeTrue(), "Expect ListObjectVersionsWithContext to be called correctly")

				od := make([]*s3.ObjectIdentifier, 2)
				od[0] = &s3.ObjectIdentifier{
					VersionId: aws.String("Vid"),
					Key:       aws.String("Vk"),
				}
				od[1] = &s3.ObjectIdentifier{
					VersionId: aws.String("Did"),
					Key:       aws.String("Dk"),
				}
				Expect(mockS3Svc.AssertCalled(
					GinkgoT(),
					"DeleteObjectsWithContext",
					ctx,
					&s3.DeleteObjectsInput{
						Bucket: aws.String("bob-s3-bucket"),
						Delete: &s3.Delete{
							Objects: od,
						},
					},
					[]request.Option(nil),
				)).To(BeTrue(), "Expect DeleteObjectsWithContext to be called correctly")
			})

			It("should call CloudFormation Delete", func() {
				Expect(
					mockCfSvc.AssertCalled(
						GinkgoT(),
						"DeleteStackWithContext",
						ctx,
						&cf.DeleteStackInput{
							StackName: aws.String("bob"),
						},
						[]request.Option(nil),
					)).To(BeTrue(), "Expect DeleteStackWithContext to be called correctly")
			})

			It("should not get an erro", func() {
				Expect(err).To(BeNil())
			})
		})
	})
})

type mockCloudFormationClient struct {
	cfAPI.CloudFormationAPI
	mock.Mock
	ctx context.Context
}

func (m *mockCloudFormationClient) DeleteStackWithContext(ctx aws.Context, input *cf.DeleteStackInput, opts ...request.Option) (*cf.DeleteStackOutput, error) {
	log.Info("DeleteStackWithContext called with:", input, opts)
	args := m.Called(ctx, input, opts)
	out := args.Get(0).(*cf.DeleteStackOutput)
	return out, args.Error(1)
}

func (m *mockCloudFormationClient) givenDeleteStackWithContextReturns(err error) {
	m.On(
		"DeleteStackWithContext",
		m.ctx,
		mock.AnythingOfType("*cloudformation.DeleteStackInput"),
		[]request.Option(nil)).
		Return(&cf.DeleteStackOutput{}, err)
}

func (m *mockCloudFormationClient) DescribeStackResourcesWithContext(ctx aws.Context, input *cf.DescribeStackResourcesInput, opts ...request.Option) (*cf.DescribeStackResourcesOutput, error) {
	log.Info("DescribeStackResourcesWithContext called with:", input, opts)
	args := m.Called(ctx, input, opts)
	out := args.Get(0).(*cf.DescribeStackResourcesOutput)
	return out, args.Error(1)
}

func (m *mockCloudFormationClient) givenDescribeStackResourcesWithContextReturns(err error) {
	m.On(
		"DescribeStackResourcesWithContext",
		m.ctx,
		mock.AnythingOfType("*cloudformation.DescribeStackResourcesInput"),
		[]request.Option(nil)).
		Return(&cf.DescribeStackResourcesOutput{
			StackResources: []*cf.StackResource{
				{
					ResourceType:       aws.String("AWS::S3::Bucket"),
					PhysicalResourceId: aws.String("bob-s3-bucket"),
				},
			},
		}, err)
}

type mockS3Client struct {
	s3iface.S3API
	mock.Mock
	ctx context.Context
}

func (m *mockS3Client) DeleteObjectsWithContext(ctx aws.Context, input *s3.DeleteObjectsInput, opts ...request.Option) (*s3.DeleteObjectsOutput, error) {
	log.Info("DeleteObjectsWithContext called with:", input, opts)
	args := m.Called(ctx, input, opts)
	out := args.Get(0).(*s3.DeleteObjectsOutput)
	return out, args.Error(1)
}

func (m *mockS3Client) givenDeleteObjectsWithContext(err error) {
	m.On(
		"DeleteObjectsWithContext",
		m.ctx,
		mock.AnythingOfType("*s3.DeleteObjectsInput"),
		[]request.Option(nil)).
		Return(&s3.DeleteObjectsOutput{}, err)
}

func (m *mockS3Client) ListObjectVersionsWithContext(ctx aws.Context, input *s3.ListObjectVersionsInput, opts ...request.Option) (*s3.ListObjectVersionsOutput, error) {
	log.Info("ListObjectVersionsWithContext called with:", input, opts)
	args := m.Called(ctx, input, opts)
	out := args.Get(0).(*s3.ListObjectVersionsOutput)
	return out, args.Error(1)
}

func (m *mockS3Client) givenListObjectVersionsWithContextReturns(vs []item, ds []item, err error) {

	v := make([]*s3.ObjectVersion, len(vs))
	for i, o := range vs {
		v[i] = &s3.ObjectVersion{
			VersionId: aws.String(o.VersionId),
			Key:       aws.String(o.Key),
		}
	}
	d := make([]*s3.DeleteMarkerEntry, len(ds))
	for i, o := range ds {
		d[i] = &s3.DeleteMarkerEntry{
			VersionId: aws.String(o.VersionId),
			Key:       aws.String(o.Key),
		}
	}
	output := &s3.ListObjectVersionsOutput{
		Versions:      v,
		DeleteMarkers: d,
	}
	m.On(
		"ListObjectVersionsWithContext",
		m.ctx,
		mock.AnythingOfType("*s3.ListObjectVersionsInput"),
		[]request.Option(nil)).
		Return(output, err)
}

type item struct {
	Key       string
	VersionId string
}
