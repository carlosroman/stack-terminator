package main_test

import (
	. "github.com/carlosroman/stack-terminator"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("StackTerminator", func() {
	Describe("Should terminate stack", func() {
		err := Termainte("bob")
		It("should call CloudFormation Delete", func() {
			// Some  sort of assert on AWS SDK
		})

		It("should not get an erro", func() {
			Expect(err).To(BeNil())
		})
	})
})
