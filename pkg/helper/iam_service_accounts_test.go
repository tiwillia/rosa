package helper_test

import (
	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/gomega"
	
	"github.com/openshift/rosa/pkg/helper"
)

var _ = Describe("IAM Service Account Helper Functions", func() {
	Context("GenerateOIDCTrustPolicy", func() {
		It("should generate valid OIDC trust policy", func() {
			issuerURL := "https://oidc.example.com"
			namespace := "default"
			serviceAccountName := "my-app"

			policy := helper.GenerateOIDCTrustPolicy(issuerURL, namespace, serviceAccountName)

			Expect(policy).To(ContainSubstring(issuerURL))
			Expect(policy).To(ContainSubstring("system:serviceaccount:default:my-app"))
			Expect(policy).To(ContainSubstring("sts.amazonaws.com"))
			Expect(policy).To(ContainSubstring("AssumeRoleWithWebIdentity"))
		})
	})

	Context("GenerateServiceAccountTags", func() {
		It("should generate correct tags", func() {
			namespace := "kube-system"
			serviceAccountName := "aws-load-balancer-controller"

			tags := helper.GenerateServiceAccountTags(namespace, serviceAccountName)

			Expect(tags).To(HaveKeyWithValue(helper.ServiceAccountRoleTag, "true"))
			Expect(tags).To(HaveKeyWithValue(helper.ServiceAccountNamespaceTag, namespace))
			Expect(tags).To(HaveKeyWithValue(helper.ServiceAccountNameTag, serviceAccountName))
		})
	})

	Context("IsServiceAccountRole", func() {
		It("should return true for service account role tags", func() {
			tags := map[string]string{
				helper.ServiceAccountRoleTag:      "true",
				helper.ServiceAccountNamespaceTag: "default",
				helper.ServiceAccountNameTag:      "my-app",
			}

			result := helper.IsServiceAccountRole(tags)
			Expect(result).To(BeTrue())
		})

		It("should return false for missing service account role tag", func() {
			tags := map[string]string{
				"other-tag": "value",
			}

			result := helper.IsServiceAccountRole(tags)
			Expect(result).To(BeFalse())
		})
	})

	Context("GenerateServiceAccountRoleName", func() {
		It("should generate role name with prefix", func() {
			prefix := "MyCluster"
			namespace := "default"
			serviceAccountName := "my-app"

			roleName := helper.GenerateServiceAccountRoleName(prefix, namespace, serviceAccountName)

			Expect(roleName).To(Equal("MyCluster-default-my-app"))
		})

		It("should generate role name without prefix", func() {
			prefix := ""
			namespace := "kube-system"
			serviceAccountName := "aws-load-balancer-controller"

			roleName := helper.GenerateServiceAccountRoleName(prefix, namespace, serviceAccountName)

			Expect(roleName).To(Equal("kube-system-aws-load-balancer-controller"))
		})
	})

	Context("ValidateServiceAccountRoleInputs", func() {
		It("should validate correct inputs", func() {
			err := helper.ValidateServiceAccountRoleInputs("default", "my-app", "MyCluster-default-my-app")
			Expect(err).ToNot(HaveOccurred())
		})

		It("should reject empty namespace", func() {
			err := helper.ValidateServiceAccountRoleInputs("", "my-app", "role-name")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("namespace cannot be empty"))
		})

		It("should reject empty service account name", func() {
			err := helper.ValidateServiceAccountRoleInputs("default", "", "role-name")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("service account name cannot be empty"))
		})

		It("should reject empty role name", func() {
			err := helper.ValidateServiceAccountRoleInputs("default", "my-app", "")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("role name cannot be empty"))
		})

		It("should reject invalid namespace format", func() {
			err := helper.ValidateServiceAccountRoleInputs("Invalid_Namespace", "my-app", "role-name")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not a valid Kubernetes namespace name"))
		})

		It("should reject invalid service account name format", func() {
			err := helper.ValidateServiceAccountRoleInputs("default", "Invalid_ServiceAccount", "role-name")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not a valid Kubernetes service account name"))
		})
	})

})
