package main

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	cf "github.com/aws/aws-sdk-go/service/cloudformation"
	cfAPI "github.com/aws/aws-sdk-go/service/cloudformation/cloudformationiface"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"os"
)

func main() {
	log.SetFormatter(&log.TextFormatter{})
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)

	cli.NewApp()
	fmt.Printf("hello, world\n")
	sess := session.Must(session.NewSession())
	cfsvc := cf.New(sess)
	ctx := context.Background()
	Termainte("bob", ctx, cfsvc)
}

func Termainte(stackName string, ctx context.Context, cfsvc cfAPI.CloudFormationAPI) error {

	res, err := cfsvc.DeleteStackWithContext(ctx, &cf.DeleteStackInput{
		StackName: aws.String(stackName),
	})

	if err != nil {
		return err
	}

	log.Info(res.GoString())
	return err
}
