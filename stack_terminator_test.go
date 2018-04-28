package main_test

import (
	"context"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	cf "github.com/aws/aws-sdk-go/service/cloudformation"
	cfAPI "github.com/aws/aws-sdk-go/service/cloudformation/cloudformationiface"
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
			mockCfSvc := &mockCloudFormationClient{}

			mockCfSvc.
				On(
					"DeleteStackWithContext",
					ctx,
					mock.AnythingOfType("*cloudformation.DeleteStackInput"),
					[]request.Option(nil)).
				Return(&cf.DeleteStackOutput{}, nil)
			err := Termainte("bob", ctx, mockCfSvc)

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
}

func (m *mockCloudFormationClient) DeleteStackWithContext(ctx aws.Context, input *cf.DeleteStackInput, opts ...request.Option) (*cf.DeleteStackOutput, error) {
	log.Info("DeleteStackWithContext called with:", input, opts)
	args := m.Called(ctx, input, opts)
	dso := args.Get(0).(*cf.DeleteStackOutput)
	return dso, args.Error(1)
}
