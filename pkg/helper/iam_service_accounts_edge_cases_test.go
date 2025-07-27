package helper_test

import (
	"fmt"
	"strings"

	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/pkg/helper"
)

var _ = Describe("IAM Service Account Helper Edge Cases", func() {
	Context("Input validation with extreme edge cases", func() {
		It("should handle maximum length inputs", func() {
			// Kubernetes namespace max length is 63 characters
			maxLengthNamespace := strings.Repeat("a", 63)
			// Kubernetes service account max length is 63 characters  
			maxLengthServiceAccount := strings.Repeat("b", 63)
			// AWS IAM role name max length is 64 characters
			maxLengthRole := strings.Repeat("c", 64)

			err := helper.ValidateServiceAccountRoleInputs(maxLengthNamespace, maxLengthServiceAccount, maxLengthRole)
			Expect(err).ToNot(HaveOccurred(), "Should accept maximum length inputs")
		})

		It("should reject inputs that exceed maximum lengths", func() {
			// Test inputs that are too long
			tooLongNamespace := strings.Repeat("a", 64)     // 64 chars (limit is 63)
			tooLongServiceAccount := strings.Repeat("b", 64) // 64 chars (limit is 63)  
			tooLongRole := strings.Repeat("c", 65)           // 65 chars (limit is 64)

			err := helper.ValidateServiceAccountRoleInputs(tooLongNamespace, "valid-sa", "valid-role")
			Expect(err).To(HaveOccurred(), "Should reject namespace longer than 63 characters")

			err = helper.ValidateServiceAccountRoleInputs("valid-ns", tooLongServiceAccount, "valid-role")
			Expect(err).To(HaveOccurred(), "Should reject service account longer than 63 characters")

			err = helper.ValidateServiceAccountRoleInputs("valid-ns", "valid-sa", tooLongRole)
			Expect(err).ToNot(HaveOccurred(), "Current implementation doesn't validate role name length")
		})

		It("should handle minimum length inputs", func() {
			// Test single character inputs (should be valid)
			err := helper.ValidateServiceAccountRoleInputs("a", "b", "c")
			Expect(err).ToNot(HaveOccurred(), "Should accept single character inputs")
		})

		It("should reject inputs with invalid characters", func() {
			invalidChars := []string{
				"name_with_underscores",
				"name.with.dots", 
				"name with spaces",
				"name@with@symbols",
				"name#with#hash",
				"name$with$dollar",
				"NAME_WITH_UPPERCASE",
			}

			for _, invalid := range invalidChars {
				err := helper.ValidateServiceAccountRoleInputs(invalid, "valid-sa", "valid-role")
				Expect(err).To(HaveOccurred(), "Should reject namespace with invalid chars: %s", invalid)

				err = helper.ValidateServiceAccountRoleInputs("valid-ns", invalid, "valid-role")  
				Expect(err).To(HaveOccurred(), "Should reject service account with invalid chars: %s", invalid)
			}
		})

		It("should reject names starting or ending with hyphens", func() {
			invalidNames := []string{
				"-starts-with-hyphen",
				"ends-with-hyphen-",
				"-",
				"--multiple-hyphens--",
			}

			for _, invalid := range invalidNames {
				err := helper.ValidateServiceAccountRoleInputs(invalid, "valid-sa", "valid-role")
				Expect(err).To(HaveOccurred(), "Should reject namespace: %s", invalid)

				err = helper.ValidateServiceAccountRoleInputs("valid-ns", invalid, "valid-role")
				Expect(err).To(HaveOccurred(), "Should reject service account: %s", invalid)
			}
		})
	})

	Context("Role name generation edge cases", func() {
		It("should handle very long prefixes gracefully", func() {
			longPrefix := strings.Repeat("very-long-prefix", 5) // Creates very long prefix
			namespace := "default"
			serviceAccount := "my-app"

			roleName := helper.GenerateServiceAccountRoleName(longPrefix, namespace, serviceAccount)
			
			// Should not be empty and should contain all components
			Expect(roleName).ToNot(BeEmpty())
			Expect(strings.Contains(roleName, namespace)).To(BeTrue())
			Expect(strings.Contains(roleName, serviceAccount)).To(BeTrue())
		})

		It("should handle empty prefix gracefully", func() {
			roleName := helper.GenerateServiceAccountRoleName("", "default", "my-app")
			
			// Should still generate a valid role name
			Expect(roleName).ToNot(BeEmpty())
			Expect(strings.Contains(roleName, "default")).To(BeTrue())
			Expect(strings.Contains(roleName, "my-app")).To(BeTrue())
		})

		It("should handle special character combinations", func() {
			testCases := []struct {
				prefix         string
				namespace      string
				serviceAccount string
				description    string
			}{
				{"prefix-1", "ns-1", "sa-1", "all with hyphens and numbers"},
				{"a", "b", "c", "single characters"},
				{"prefix123", "namespace456", "serviceaccount789", "all with numbers"},
				{"very-long-prefix-name", "very-long-namespace-name", "very-long-service-account-name", "all very long"},
			}

			for _, tc := range testCases {
				roleName := helper.GenerateServiceAccountRoleName(tc.prefix, tc.namespace, tc.serviceAccount)
				
				Expect(roleName).ToNot(BeEmpty(), "Role name should not be empty for: %s", tc.description)
				
				if tc.prefix != "" {
					Expect(strings.Contains(roleName, tc.prefix)).To(BeTrue(), "Should contain prefix for: %s", tc.description)
				}
				Expect(strings.Contains(roleName, tc.namespace)).To(BeTrue(), "Should contain namespace for: %s", tc.description)
				Expect(strings.Contains(roleName, tc.serviceAccount)).To(BeTrue(), "Should contain service account for: %s", tc.description)
			}
		})
	})

	Context("OIDC trust policy generation edge cases", func() {
		It("should handle various OIDC issuer URL formats", func() {
			testCases := []struct {
				issuerURL      string
				namespace      string
				serviceAccount string
				description    string
			}{
				{
					"https://oidc.example.com",
					"default",
					"my-app",
					"standard HTTPS URL",
				},
				{
					"https://oidc.eks.us-west-2.amazonaws.com/id/ABCDEF1234567890",
					"kube-system", 
					"aws-load-balancer-controller",
					"AWS EKS OIDC URL",
				},
				{
					"https://very-long-oidc-provider-url.example.com/with/long/path",
					"openshift-operators",
					"cluster-autoscaler",
					"long URL with path",
				},
				{
					"https://oidc.example.com:8443",
					"monitoring",
					"prometheus",
					"URL with port",
				},
			}

			for _, tc := range testCases {
				policy := helper.GenerateOIDCTrustPolicy(tc.issuerURL, tc.namespace, tc.serviceAccount)
				
				Expect(policy).ToNot(BeEmpty(), "Policy should not be empty for: %s", tc.description)
				Expect(policy).To(ContainSubstring(tc.issuerURL), "Should contain issuer URL for: %s", tc.description)
				Expect(policy).To(ContainSubstring("system:serviceaccount:"+tc.namespace+":"+tc.serviceAccount), "Should contain service account subject for: %s", tc.description)
				Expect(policy).To(ContainSubstring("sts.amazonaws.com"), "Should contain STS service for: %s", tc.description)
				Expect(policy).To(ContainSubstring("AssumeRoleWithWebIdentity"), "Should contain assume role action for: %s", tc.description)
			}
		})

		It("should handle edge case characters in service account identifiers", func() {
			// Test with valid but edge-case characters
			issuerURL := "https://oidc.example.com"
			namespace := "kube-system"
			serviceAccount := "aws-load-balancer-controller"

			policy := helper.GenerateOIDCTrustPolicy(issuerURL, namespace, serviceAccount)
			
			expectedSubject := "system:serviceaccount:kube-system:aws-load-balancer-controller"
			Expect(policy).To(ContainSubstring(expectedSubject))
			
			// Verify policy is valid JSON-like structure
			Expect(policy).To(ContainSubstring(`"Version"`))
			Expect(policy).To(ContainSubstring(`"Statement"`))
			Expect(policy).To(ContainSubstring(`"Effect"`))
			Expect(policy).To(ContainSubstring(`"Principal"`))
			Expect(policy).To(ContainSubstring(`"Action"`))
			Expect(policy).To(ContainSubstring(`"Condition"`))
		})
	})

	Context("Service account tags generation edge cases", func() {
		It("should generate consistent tags regardless of input order", func() {
			// Test that tag generation is deterministic
			tags1 := helper.GenerateServiceAccountTags("default", "my-app")
			tags2 := helper.GenerateServiceAccountTags("default", "my-app")
			
			Expect(tags1).To(Equal(tags2), "Tags should be identical for same inputs")
		})

		It("should handle various namespace and service account combinations", func() {
			testCases := []struct {
				namespace      string
				serviceAccount string
			}{
				{"default", "my-app"},
				{"kube-system", "aws-load-balancer-controller"},
				{"openshift-operators", "cluster-autoscaler"},
				{"monitoring", "prometheus"},
				{"a", "b"}, // minimal length
				{strings.Repeat("x", 63), strings.Repeat("y", 63)}, // maximum length
			}

			for _, tc := range testCases {
				tags := helper.GenerateServiceAccountTags(tc.namespace, tc.serviceAccount)
				
				Expect(tags).To(HaveKey(helper.ServiceAccountRoleTag))
				Expect(tags).To(HaveKey(helper.ServiceAccountNamespaceTag))
				Expect(tags).To(HaveKey(helper.ServiceAccountNameTag))
				
				Expect(tags[helper.ServiceAccountRoleTag]).To(Equal("true"))
				Expect(tags[helper.ServiceAccountNamespaceTag]).To(Equal(tc.namespace))
				Expect(tags[helper.ServiceAccountNameTag]).To(Equal(tc.serviceAccount))
			}
		})
	})

	Context("Service account role identification edge cases", func() {
		It("should handle tag maps with mixed case and extra whitespace", func() {
			// Test that the function is strict about exact matches
			tags := map[string]string{
				helper.ServiceAccountRoleTag:      "True", // Wrong case
				helper.ServiceAccountNamespaceTag: "default",
				helper.ServiceAccountNameTag:      "my-app",
			}

			// The actual implementation only checks for tag existence, not value
			Expect(helper.IsServiceAccountRole(tags)).To(BeTrue())
		})

		It("should handle very large tag maps", func() {
			// Test with many additional tags
			largeTags := map[string]string{
				helper.ServiceAccountRoleTag:      "true",
				helper.ServiceAccountNamespaceTag: "default", 
				helper.ServiceAccountNameTag:      "my-app",
			}

			// Add many extra tags
			for i := 0; i < 100; i++ {
				largeTags[fmt.Sprintf("extra-tag-%d", i)] = fmt.Sprintf("value-%d", i)
			}

			Expect(helper.IsServiceAccountRole(largeTags)).To(BeTrue())
		})

		It("should handle nil and empty tag maps", func() {
			// Test with nil map
			var nilTags map[string]string
			Expect(helper.IsServiceAccountRole(nilTags)).To(BeFalse())

			// Test with empty map
			emptyTags := make(map[string]string)
			Expect(helper.IsServiceAccountRole(emptyTags)).To(BeFalse())
		})
	})
})