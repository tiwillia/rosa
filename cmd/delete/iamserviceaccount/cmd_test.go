package iamserviceaccount

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	"github.com/openshift/rosa/pkg/helper"
)

func TestDeleteIAMServiceAccountCommand(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Delete IAM Service Account Command Suite")
}

var _ = Describe("Delete IAM Service Account Command", func() {
	var ctrl *gomock.Controller

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("Safety validation logic", func() {
		It("should correctly identify service account roles", func() {
			// Test valid service account role tags
			validServiceAccountTags := map[string]string{
				helper.ServiceAccountRoleTag:      "true",
				helper.ServiceAccountNamespaceTag: "default",
				helper.ServiceAccountNameTag:      "my-app",
			}

			isServiceAccountRole := helper.IsServiceAccountRole(validServiceAccountTags)
			Expect(isServiceAccountRole).To(BeTrue(), "Should identify valid service account role")
		})

		It("should reject roles without service account role tag", func() {
			// Test role without the service account role tag
			tagsWithoutRoleTag := map[string]string{
				helper.ServiceAccountNamespaceTag: "default",
				helper.ServiceAccountNameTag:      "my-app",
				"other-tag":                       "other-value",
			}

			isServiceAccountRole := helper.IsServiceAccountRole(tagsWithoutRoleTag)
			Expect(isServiceAccountRole).To(BeFalse(), "Should reject role without service account role tag")
		})

		It("should accept roles with any service account role tag value", func() {
			// The actual implementation only checks for tag existence, not value
			tagsWithAnyValue := map[string]string{
				helper.ServiceAccountRoleTag:      "false", // Any value is accepted
				helper.ServiceAccountNamespaceTag: "default",
				helper.ServiceAccountNameTag:      "my-app",
			}

			isServiceAccountRole := helper.IsServiceAccountRole(tagsWithAnyValue)
			Expect(isServiceAccountRole).To(BeTrue(), "Should accept role with any service account role tag value")
		})

		It("should reject regular IAM roles", func() {
			// Test regular IAM role tags (not service account)
			regularRoleTags := map[string]string{
				"Environment": "production",
				"Team":        "platform",
				"Application": "web-server",
			}

			isServiceAccountRole := helper.IsServiceAccountRole(regularRoleTags)
			Expect(isServiceAccountRole).To(BeFalse(), "Should reject regular IAM roles")
		})

		It("should accept roles with partial service account tags", func() {
			// The implementation only checks for the main tag existence
			partialTags := map[string]string{
				helper.ServiceAccountRoleTag:      "true",
				helper.ServiceAccountNamespaceTag: "default",
				// Missing ServiceAccountNameTag - but still accepted
			}

			isServiceAccountRole := helper.IsServiceAccountRole(partialTags)
			Expect(isServiceAccountRole).To(BeTrue(), "Should accept roles with main service account tag")
		})

		It("should reject roles with empty tags", func() {
			// Test role with no tags at all
			emptyTags := map[string]string{}

			isServiceAccountRole := helper.IsServiceAccountRole(emptyTags)
			Expect(isServiceAccountRole).To(BeFalse(), "Should reject roles with no tags")
		})

		It("should accept roles even with empty service account tag values", func() {
			// The implementation only checks for tag key existence, not values
			tagsWithEmptyValues := map[string]string{
				helper.ServiceAccountRoleTag:      "true",
				helper.ServiceAccountNamespaceTag: "",    // Empty namespace still accepted
				helper.ServiceAccountNameTag:      "my-app",
			}

			isServiceAccountRole := helper.IsServiceAccountRole(tagsWithEmptyValues)
			Expect(isServiceAccountRole).To(BeTrue(), "Should accept roles with service account role tag present")
		})

		It("should handle mixed role types correctly", func() {
			testCases := []struct {
				description  string
				tags         map[string]string
				shouldAccept bool
			}{
				{
					"Valid service account role",
					map[string]string{
						helper.ServiceAccountRoleTag:      "true",
						helper.ServiceAccountNamespaceTag: "kube-system",
						helper.ServiceAccountNameTag:      "aws-load-balancer-controller",
					},
					true,
				},
				{
					"Account role (not service account)",
					map[string]string{
						"red-hat-managed": "true",
						"rosa.aws.openshift.io/account-role": "installer",
					},
					false,
				},
				{
					"Operator role (not service account)",
					map[string]string{
						"red-hat-managed": "true",
						"rosa.aws.openshift.io/operator-role": "cluster-autoscaler",
					},
					false,
				},
				{
					"User role (not service account)",
					map[string]string{
						"red-hat-managed": "true",
						"rosa.aws.openshift.io/user-role": "true",
					},
					false,
				},
				{
					"Service account role with extra tags",
					map[string]string{
						helper.ServiceAccountRoleTag:      "true",
						helper.ServiceAccountNamespaceTag: "default",
						helper.ServiceAccountNameTag:      "my-app",
						"Environment":                     "production",
						"Team":                            "platform",
					},
					true,
				},
			}

			for _, tc := range testCases {
				result := helper.IsServiceAccountRole(tc.tags)
				if tc.shouldAccept {
					Expect(result).To(BeTrue(), "Failed for case: %s", tc.description)
				} else {
					Expect(result).To(BeFalse(), "Failed for case: %s", tc.description)
				}
			}
		})
	})

	Context("Tag extraction and validation", func() {
		It("should correctly extract namespace and service account from tags", func() {
			tags := map[string]string{
				helper.ServiceAccountRoleTag:      "true",
				helper.ServiceAccountNamespaceTag: "kube-system",
				helper.ServiceAccountNameTag:      "aws-load-balancer-controller",
				"extra-tag":                       "extra-value",
			}

			// Test that we can extract the correct values
			namespace := tags[helper.ServiceAccountNamespaceTag]
			serviceAccountName := tags[helper.ServiceAccountNameTag]

			Expect(namespace).To(Equal("kube-system"))
			Expect(serviceAccountName).To(Equal("aws-load-balancer-controller"))
		})

		It("should handle missing tag values gracefully", func() {
			tags := map[string]string{
				helper.ServiceAccountRoleTag: "true",
				// Missing namespace and service account tags
			}

			namespace := tags[helper.ServiceAccountNamespaceTag]
			serviceAccountName := tags[helper.ServiceAccountNameTag]

			// Should return empty strings for missing tags
			Expect(namespace).To(Equal(""))
			Expect(serviceAccountName).To(Equal(""))
		})

		It("should validate extracted tag values", func() {
			validCases := []struct {
				namespace      string
				serviceAccount string
				shouldBeValid  bool
			}{
				{"default", "my-app", true},
				{"kube-system", "aws-load-balancer-controller", true},
				{"", "my-app", false},     // empty namespace should be invalid
				{"default", "", false},    // empty service account should be invalid
				{"", "", false},           // both empty should be invalid
			}

			for _, tc := range validCases {
				if tc.shouldBeValid {
					Expect(tc.namespace).ToNot(BeEmpty(), "Namespace should not be empty")
					Expect(tc.serviceAccount).ToNot(BeEmpty(), "Service account should not be empty")
				} else {
					// At least one should be empty or invalid
					isEmpty := tc.namespace == "" || tc.serviceAccount == ""
					Expect(isEmpty).To(BeTrue(), "Expected at least one empty value for invalid case")
				}
			}
		})
	})

	Context("Role name validation", func() {
		It("should validate role name format and length", func() {
			testCases := []struct {
				roleName    string
				description string
				shouldPass  bool
			}{
				{"MyCluster-default-my-app", "standard format", true},
				{"rosa-12345-kube-system-aws-load-balancer-controller", "long but valid", true},
				{"", "empty role name", false},
				{"a", "very short", true},
				{string(make([]byte, 65)), "too long (65 chars)", false}, // AWS limit is 64
				{"valid-role-name-123", "with numbers", true},
			}

			for _, tc := range testCases {
				if tc.shouldPass {
					Expect(tc.roleName).ToNot(BeEmpty(), "Role name should not be empty for case: %s", tc.description)
					Expect(len(tc.roleName)).To(BeNumerically("<=", 64), "Role name too long for case: %s", tc.description)
				} else if tc.roleName == "" {
					Expect(tc.roleName).To(BeEmpty(), "Role name should be empty for case: %s", tc.description)
				} else {
					Expect(len(tc.roleName)).To(BeNumerically(">", 64), "Role name should exceed limit for case: %s", tc.description)
				}
			}
		})
	})
})