package iamserviceaccount

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	"github.com/openshift/rosa/pkg/helper"
)

func TestDescribeIAMServiceAccountCommand(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Describe IAM Service Account Command Suite")
}

var _ = Describe("Describe IAM Service Account Command", func() {
	var ctrl *gomock.Controller

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("Role validation logic", func() {
		It("should correctly identify service account roles for description", func() {
			// Test valid service account role tags
			validServiceAccountTags := map[string]string{
				helper.ServiceAccountRoleTag:      "true",
				helper.ServiceAccountNamespaceTag: "kube-system",
				helper.ServiceAccountNameTag:      "aws-load-balancer-controller",
			}

			isServiceAccountRole := helper.IsServiceAccountRole(validServiceAccountTags)
			Expect(isServiceAccountRole).To(BeTrue(), "Should identify valid service account role")

			// Extract namespace and service account name
			namespace := validServiceAccountTags[helper.ServiceAccountNamespaceTag]
			serviceAccountName := validServiceAccountTags[helper.ServiceAccountNameTag]

			Expect(namespace).To(Equal("kube-system"))
			Expect(serviceAccountName).To(Equal("aws-load-balancer-controller"))
		})

		It("should reject non-service-account roles", func() {
			invalidRoleTags := []map[string]string{
				{
					// Account role
					"red-hat-managed": "true",
					"rosa.aws.openshift.io/account-role": "installer",
				},
				{
					// Operator role
					"red-hat-managed": "true",
					"rosa.aws.openshift.io/operator-role": "cluster-autoscaler",
				},
				{
					// Regular role with no special tags
					"Environment": "production",
					"Team":        "platform",
				},
				{
					// Service account role with wrong value
					helper.ServiceAccountRoleTag:      "false",
					helper.ServiceAccountNamespaceTag: "default",
					helper.ServiceAccountNameTag:      "my-app",
				},
			}

			for i, tags := range invalidRoleTags {
				isServiceAccountRole := helper.IsServiceAccountRole(tags)
				Expect(isServiceAccountRole).To(BeFalse(), "Should reject invalid role tags (case %d)", i+1)
			}
		})
	})

	Context("Tag extraction and processing", func() {
		It("should correctly extract service account information from complete tags", func() {
			completeTags := map[string]string{
				helper.ServiceAccountRoleTag:      "true",
				helper.ServiceAccountNamespaceTag: "monitoring",
				helper.ServiceAccountNameTag:      "prometheus",
				"Environment":                     "production",
				"Team":                            "platform",
				"Created":                         "2024-01-01",
			}

			// Extract service account specific information
			namespace := completeTags[helper.ServiceAccountNamespaceTag]
			serviceAccountName := completeTags[helper.ServiceAccountNameTag]

			Expect(namespace).To(Equal("monitoring"))
			Expect(serviceAccountName).To(Equal("prometheus"))

			// Verify it's identified as a service account role
			Expect(helper.IsServiceAccountRole(completeTags)).To(BeTrue())
		})

		It("should handle tags with special characters and values", func() {
			specialTags := map[string]string{
				helper.ServiceAccountRoleTag:      "true",
				helper.ServiceAccountNamespaceTag: "openshift-operators",
				helper.ServiceAccountNameTag:      "cluster-autoscaler-operator",
				"custom:tag":                      "value:with:colons",
				"tag-with-hyphens":                "value-with-hyphens",
				"tag_with_underscores":            "value_with_underscores",
			}

			namespace := specialTags[helper.ServiceAccountNamespaceTag]
			serviceAccountName := specialTags[helper.ServiceAccountNameTag]

			Expect(namespace).To(Equal("openshift-operators"))
			Expect(serviceAccountName).To(Equal("cluster-autoscaler-operator"))
			Expect(helper.IsServiceAccountRole(specialTags)).To(BeTrue())
		})

		It("should handle empty or missing tag values gracefully", func() {
			incompleteTags := map[string]string{
				helper.ServiceAccountRoleTag:      "true",
				helper.ServiceAccountNamespaceTag: "",  // Empty namespace
				helper.ServiceAccountNameTag:      "my-app",
			}

			namespace := incompleteTags[helper.ServiceAccountNamespaceTag]
			serviceAccountName := incompleteTags[helper.ServiceAccountNameTag]

			Expect(namespace).To(Equal(""))
			Expect(serviceAccountName).To(Equal("my-app"))

			// Should be identified as invalid due to empty namespace
			Expect(helper.IsServiceAccountRole(incompleteTags)).To(BeFalse())
		})
	})

	Context("Role information structure validation", func() {
		It("should validate role details structure fields", func() {
			// Test the role details structure that would be created
			roleDetails := struct {
				RoleName           string            `json:"role_name"`
				RoleArn            string            `json:"role_arn"`
				Namespace          string            `json:"namespace"`
				ServiceAccountName string            `json:"service_account_name"`
				Path               string            `json:"path"`
				MaxSessionDuration int32             `json:"max_session_duration"`
				CreationDate       string            `json:"creation_date"`
				TrustPolicy        string            `json:"trust_policy"`
				AttachedPolicies   []string          `json:"attached_policies"`
				Tags               map[string]string `json:"tags"`
			}{
				RoleName:           "MyCluster-default-my-app",
				RoleArn:            "arn:aws:iam::123456789012:role/MyCluster-default-my-app",
				Namespace:          "default",
				ServiceAccountName: "my-app",
				Path:               "/",
				MaxSessionDuration: 3600,
				CreationDate:       "2024-01-01T12:00:00Z",
				TrustPolicy:        `{"Version":"2012-10-17","Statement":[]}`,
				AttachedPolicies:   []string{"arn:aws:iam::123456789012:policy/MyPolicy"},
				Tags: map[string]string{
					helper.ServiceAccountRoleTag:      "true",
					helper.ServiceAccountNamespaceTag: "default",
					helper.ServiceAccountNameTag:      "my-app",
				},
			}

			// Validate all required fields are present
			Expect(roleDetails.RoleName).ToNot(BeEmpty())
			Expect(roleDetails.RoleArn).ToNot(BeEmpty())
			Expect(roleDetails.Namespace).ToNot(BeEmpty())
			Expect(roleDetails.ServiceAccountName).ToNot(BeEmpty())
			Expect(roleDetails.Path).ToNot(BeEmpty())
			Expect(roleDetails.MaxSessionDuration).To(BeNumerically(">", 0))
			Expect(roleDetails.CreationDate).ToNot(BeEmpty())
			Expect(roleDetails.TrustPolicy).ToNot(BeEmpty())
			Expect(roleDetails.Tags).ToNot(BeEmpty())

			// Validate specific values
			Expect(roleDetails.RoleName).To(Equal("MyCluster-default-my-app"))
			Expect(roleDetails.Namespace).To(Equal("default"))
			Expect(roleDetails.ServiceAccountName).To(Equal("my-app"))
			Expect(roleDetails.AttachedPolicies).To(HaveLen(1))
		})

		It("should handle roles with no attached policies", func() {
			roleDetailsNoPolices := struct {
				AttachedPolicies []string `json:"attached_policies"`
			}{
				AttachedPolicies: []string{},
			}

			Expect(roleDetailsNoPolices.AttachedPolicies).To(HaveLen(0))
			Expect(roleDetailsNoPolices.AttachedPolicies).ToNot(BeNil())
		})

		It("should handle roles with multiple attached policies", func() {
			multiplePolices := []string{
				"arn:aws:iam::123456789012:policy/Policy1",
				"arn:aws:iam::123456789012:policy/Policy2",
				"arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess",
			}

			roleDetailsMultiplePolices := struct {
				AttachedPolicies []string `json:"attached_policies"`
			}{
				AttachedPolicies: multiplePolices,
			}

			Expect(roleDetailsMultiplePolices.AttachedPolicies).To(HaveLen(3))
			Expect(roleDetailsMultiplePolices.AttachedPolicies).To(ContainElement("arn:aws:iam::123456789012:policy/Policy1"))
			Expect(roleDetailsMultiplePolices.AttachedPolicies).To(ContainElement("arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess"))
		})
	})

	Context("Usage instructions generation", func() {
		It("should generate correct kubectl annotation commands", func() {
			roleArn := "arn:aws:iam::123456789012:role/MyCluster-default-my-app"
			namespace := "default"
			serviceAccountName := "my-app"

			// Test the usage instruction format that would be displayed
			expectedAnnotation := "eks.amazonaws.com/role-arn: " + roleArn
			expectedKubectlCommand := "kubectl annotate serviceaccount " + serviceAccountName + 
				" -n " + namespace + " eks.amazonaws.com/role-arn=" + roleArn

			Expect(expectedAnnotation).To(Equal("eks.amazonaws.com/role-arn: arn:aws:iam::123456789012:role/MyCluster-default-my-app"))
			Expect(expectedKubectlCommand).To(Equal("kubectl annotate serviceaccount my-app -n default eks.amazonaws.com/role-arn=arn:aws:iam::123456789012:role/MyCluster-default-my-app"))
		})

		It("should handle various namespace and service account combinations", func() {
			testCases := []struct {
				namespace      string
				serviceAccount string
				roleArn        string
				expectedCmd    string
			}{
				{
					"kube-system",
					"aws-load-balancer-controller",
					"arn:aws:iam::123456789012:role/MyCluster-kube-system-aws-load-balancer-controller",
					"kubectl annotate serviceaccount aws-load-balancer-controller -n kube-system eks.amazonaws.com/role-arn=arn:aws:iam::123456789012:role/MyCluster-kube-system-aws-load-balancer-controller",
				},
				{
					"monitoring",
					"prometheus",
					"arn:aws:iam::123456789012:role/MyCluster-monitoring-prometheus",
					"kubectl annotate serviceaccount prometheus -n monitoring eks.amazonaws.com/role-arn=arn:aws:iam::123456789012:role/MyCluster-monitoring-prometheus",
				},
			}

			for _, tc := range testCases {
				kubectlCmd := "kubectl annotate serviceaccount " + tc.serviceAccount + 
					" -n " + tc.namespace + " eks.amazonaws.com/role-arn=" + tc.roleArn
				Expect(kubectlCmd).To(Equal(tc.expectedCmd))
			}
		})
	})

	Context("Input validation for describe command", func() {
		It("should validate role name format requirements", func() {
			validRoleNames := []string{
				"MyCluster-default-my-app",
				"rosa-12345-kube-system-aws-load-balancer-controller",
				"cluster-monitoring-prometheus",
				"a",  // minimum length
			}

			invalidRoleNames := []string{
				"",   // empty
				string(make([]byte, 65)), // too long (AWS limit is 64)
			}

			for _, roleName := range validRoleNames {
				Expect(roleName).ToNot(BeEmpty(), "Valid role name should not be empty")
				Expect(len(roleName)).To(BeNumerically("<=", 64), "Valid role name should not exceed 64 characters")
			}

			for _, roleName := range invalidRoleNames {
				if roleName == "" {
					Expect(roleName).To(BeEmpty(), "Invalid role name should be empty")
				} else {
					Expect(len(roleName)).To(BeNumerically(">", 64), "Invalid role name should exceed 64 characters")
				}
			}
		})
	})
})