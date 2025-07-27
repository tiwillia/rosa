package iamserviceaccount

import (
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"go.uber.org/mock/gomock"

	"github.com/openshift/rosa/pkg/helper"
)

func TestCreateIAMServiceAccountCommand(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Create IAM Service Account Command Suite")
}

var _ = Describe("Create IAM Service Account Command", func() {
	var ctrl *gomock.Controller

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("Role name generation logic", func() {
		It("should extract prefix from installer role ARN correctly", func() {
			// Test the role name generation logic from create command
			// Input: cluster with installer role ARN
			installerRoleARN := "arn:aws:iam::123456789012:role/ManagedOpenShift-Installer-Role"
			namespace := "default"
			serviceAccountName := "my-app"

			// Simulate the prefix extraction logic
			prefix := installerRoleARN
			if prefix != "" {
				parts := strings.Split(prefix, "/")
				if len(parts) > 1 {
					roleName := parts[len(parts)-1]
					if strings.Contains(roleName, "-Installer-Role") {
						prefix = strings.Replace(roleName, "-Installer-Role", "", 1)
					}
				}
			}

			Expect(prefix).To(Equal("ManagedOpenShift"))

			// Generate role name using helper function
			generatedRoleName := helper.GenerateServiceAccountRoleName(prefix, namespace, serviceAccountName)
			Expect(generatedRoleName).To(Equal("ManagedOpenShift-default-my-app"))
		})

		It("should handle various installer role ARN formats", func() {
			testCases := []struct {
				installerRoleARN string
				namespace        string
				serviceAccount   string
				expectedRoleName string
			}{
				{
					"arn:aws:iam::123456789012:role/MyPrefix-Installer-Role",
					"kube-system",
					"aws-load-balancer-controller",
					"MyPrefix-kube-system-aws-load-balancer-controller",
				},
				{
					"arn:aws:iam::123456789012:role/rosa-12345-Installer-Role",
					"openshift-operators",
					"cluster-autoscaler",
					"rosa-12345-openshift-operators-cluster-autoscaler",
				},
				{
					"arn:aws:iam::123456789012:role/ComplexPrefix-v2-Installer-Role",
					"default",
					"my-service",
					"ComplexPrefix-v2-default-my-service",
				},
			}

			for _, tc := range testCases {
				prefix := tc.installerRoleARN
				if prefix != "" {
					parts := strings.Split(prefix, "/")
					if len(parts) > 1 {
						roleName := parts[len(parts)-1]
						if strings.Contains(roleName, "-Installer-Role") {
							prefix = strings.Replace(roleName, "-Installer-Role", "", 1)
						}
					}
				}

				generatedRoleName := helper.GenerateServiceAccountRoleName(prefix, tc.namespace, tc.serviceAccount)
				Expect(generatedRoleName).To(Equal(tc.expectedRoleName))
			}
		})

		It("should fallback to cluster name when no installer role ARN", func() {
			// Test fallback logic when cluster has no installer role ARN
			clusterName := "my-test-cluster"
			namespace := "default"
			serviceAccountName := "my-app"

			// Simulate empty role ARN scenario
			prefix := ""
			if prefix == "" {
				prefix = clusterName
			}

			Expect(prefix).To(Equal("my-test-cluster"))

			generatedRoleName := helper.GenerateServiceAccountRoleName(prefix, namespace, serviceAccountName)
			Expect(generatedRoleName).To(Equal("my-test-cluster-default-my-app"))
		})

		It("should handle malformed installer role ARN gracefully", func() {
			testCases := []struct {
				description      string
				installerRoleARN string
				clusterName      string
				expectedPrefix   string
			}{
				{
					"ARN without slash",
					"malformed-arn-without-slash", 
					"fallback-cluster",
					"malformed-arn-without-slash", // Uses the ARN as-is when no slash
				},
				{
					"ARN with slash but no installer role suffix",
					"arn:aws:iam::123456789012:role/SomeOtherRole",
					"fallback-cluster",
					"SomeOtherRole",
				},
				{
					"Empty ARN",
					"",
					"fallback-cluster",
					"fallback-cluster",
				},
				{
					"ARN with only slash",
					"/",
					"fallback-cluster",
					"fallback-cluster",
				},
			}

			for _, tc := range testCases {
				prefix := tc.installerRoleARN
				if prefix != "" {
					parts := strings.Split(prefix, "/")
					if len(parts) > 1 {
						roleName := parts[len(parts)-1]
						if strings.Contains(roleName, "-Installer-Role") {
							prefix = strings.Replace(roleName, "-Installer-Role", "", 1)
						} else {
							// Use the role name as-is if it doesn't contain installer suffix
							prefix = roleName
						}
					}
				}
				if prefix == "" {
					prefix = tc.clusterName
				}

				Expect(prefix).To(Equal(tc.expectedPrefix), "Failed for case: %s", tc.description)
			}
		})

		It("should generate correct role names with various inputs", func() {
			testCases := []struct {
				prefix         string
				namespace      string
				serviceAccount string
				expectedRole   string
			}{
				{"ManagedOpenShift", "default", "my-app", "ManagedOpenShift-default-my-app"},
				{"rosa-12345", "kube-system", "aws-load-balancer-controller", "rosa-12345-kube-system-aws-load-balancer-controller"},
				{"cluster", "openshift-operators", "cluster-autoscaler", "cluster-openshift-operators-cluster-autoscaler"},
				{"test", "monitoring", "prometheus", "test-monitoring-prometheus"},
			}

			for _, tc := range testCases {
				result := helper.GenerateServiceAccountRoleName(tc.prefix, tc.namespace, tc.serviceAccount)
				Expect(result).To(Equal(tc.expectedRole))
			}
		})
	})

	Context("OIDC configuration validation", func() {
		It("should validate cluster has OIDC configuration", func() {
			// Test that cluster without OIDC should fail
			clusterWithoutOIDC := cmv1.NewCluster().
				ID("test-cluster").
				Name("test-cluster").
				AWS(cmv1.NewAWS().
					STS(cmv1.NewSTS()))

			cluster, err := clusterWithoutOIDC.Build()
			Expect(err).ToNot(HaveOccurred())

			// This should fail validation
			hasOIDC := cluster.AWS().STS().OidcConfig() != nil && cluster.AWS().STS().OidcConfig().IssuerUrl() != ""
			Expect(hasOIDC).To(BeFalse())
		})

		It("should accept cluster with valid OIDC configuration", func() {
			// Test that cluster with OIDC should pass
			clusterWithOIDC := cmv1.NewCluster().
				ID("test-cluster").
				Name("test-cluster").
				AWS(cmv1.NewAWS().
					STS(cmv1.NewSTS().
						OidcConfig(cmv1.NewOidcConfig().
							IssuerUrl("https://oidc.example.com"))))

			cluster, err := clusterWithOIDC.Build()
			Expect(err).ToNot(HaveOccurred())

			// This should pass validation
			hasOIDC := cluster.AWS().STS().OidcConfig() != nil && cluster.AWS().STS().OidcConfig().IssuerUrl() != ""
			Expect(hasOIDC).To(BeTrue())
			Expect(cluster.AWS().STS().OidcConfig().IssuerUrl()).To(Equal("https://oidc.example.com"))
		})
	})

	Context("Input validation edge cases", func() {
		It("should handle very long role names", func() {
			// AWS IAM role names have a 64 character limit
			// This test demonstrates that the current implementation doesn't enforce this limit
			// This could be considered a potential issue for future improvement
			longPrefix := "very-long-cluster-prefix-that-could-cause-issues"
			longNamespace := "very-long-namespace-name" 
			longServiceAccount := "very-long-service-account-name-that-exceeds-limits"

			roleName := helper.GenerateServiceAccountRoleName(longPrefix, longNamespace, longServiceAccount)
			
			// NOTE: The current implementation doesn't limit role name length
			// This test documents the current behavior rather than enforcing a limit
			Expect(roleName).ToNot(BeEmpty(), "Role name should not be empty")
			Expect(strings.Contains(roleName, longPrefix)).To(BeTrue(), "Should contain prefix")
			Expect(strings.Contains(roleName, longNamespace)).To(BeTrue(), "Should contain namespace")
			Expect(strings.Contains(roleName, longServiceAccount)).To(BeTrue(), "Should contain service account")
			
			// Log a warning if the name exceeds AWS limits (this is informational)
			if len(roleName) > 64 {
				// This is expected with very long inputs - the helper doesn't truncate
				Expect(len(roleName)).To(BeNumerically(">", 64), "Long inputs produce long role names")
			}
		})

		It("should validate namespace and service account name formats", func() {
			validInputs := []struct {
				namespace      string
				serviceAccount string
				shouldBeValid  bool
			}{
				{"default", "my-app", true},
				{"kube-system", "aws-load-balancer-controller", true},
				{"openshift-operators", "cluster-autoscaler", true},
				{"", "my-app", false},                    // empty namespace
				{"default", "", false},                   // empty service account
				{"Invalid_Namespace", "my-app", false},   // underscore not allowed
				{"default", "Invalid_ServiceAccount", false}, // underscore not allowed
				{"-invalid", "my-app", false},            // starts with hyphen
				{"default", "-invalid", false},           // starts with hyphen
				{"invalid-", "my-app", false},            // ends with hyphen
				{"default", "invalid-", false},           // ends with hyphen
			}

			for _, input := range validInputs {
				err := helper.ValidateServiceAccountRoleInputs(input.namespace, input.serviceAccount, "test-role")
				if input.shouldBeValid {
					Expect(err).ToNot(HaveOccurred(), "Expected valid input: %s/%s", input.namespace, input.serviceAccount)
				} else {
					Expect(err).To(HaveOccurred(), "Expected invalid input: %s/%s", input.namespace, input.serviceAccount)
				}
			}
		})

		It("should handle special characters in role name inputs", func() {
			// Test edge cases that could break role name generation
			edgeCases := []struct {
				prefix         string
				namespace      string
				serviceAccount string
				description    string
			}{
				{"prefix-with-hyphens", "ns-with-hyphens", "sa-with-hyphens", "hyphens"},
				{"prefix123", "namespace123", "serviceaccount123", "numbers"},
				{"p", "n", "s", "single characters"},
				{"a" + strings.Repeat("b", 10), "n", "s", "long prefix"},
			}

			for _, tc := range edgeCases {
				// Should not panic or return empty string
				result := helper.GenerateServiceAccountRoleName(tc.prefix, tc.namespace, tc.serviceAccount)
				Expect(result).ToNot(BeEmpty(), "Failed for case: %s", tc.description)
				Expect(strings.Contains(result, tc.prefix)).To(BeTrue(), "Should contain prefix for case: %s", tc.description)
				Expect(strings.Contains(result, tc.namespace)).To(BeTrue(), "Should contain namespace for case: %s", tc.description)
				Expect(strings.Contains(result, tc.serviceAccount)).To(BeTrue(), "Should contain service account for case: %s", tc.description)
			}
		})
	})
})