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

		Context("when stack deletes succesfully", func() {
			mockCfSvc := &mockCloudFormationClient{ctx: ctx}
			mockS3Svc := &mockS3Client{ctx: ctx}

			mockCfSvc.givenDescribeStackResourcesWithContextReturns(nil)
			mockS3Svc.givenListObjectVersionsWithContextReturns(nil)

			mockCfSvc.givenDeleteStackWithContextReturns(nil)

			err := Termainte("bob", ctx, mockCfSvc, mockS3Svc)

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
				// Some  sort of assert on AWS SDK
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

func (m *mockS3Client) ListObjectVersionsWithContext(ctx aws.Context, input *s3.ListObjectVersionsInput, opts ...request.Option) (*s3.ListObjectVersionsOutput, error) {
	log.Info("ListObjectVersionsWithContext called with:", input, opts)
	args := m.Called(ctx, input, opts)
	out := args.Get(0).(*s3.ListObjectVersionsOutput)
	return out, args.Error(1)
}

func (m *mockS3Client) givenListObjectVersionsWithContextReturns(err error) {
	m.On(
		"ListObjectVersionsWithContext",
		m.ctx,
		mock.AnythingOfType("*s3.ListObjectVersionsInput"),
		[]request.Option(nil)).
		Return(&s3.ListObjectVersionsOutput{}, err)
}
