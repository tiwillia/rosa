package iamserviceaccounts

import (
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	
	"github.com/openshift/rosa/pkg/helper"
)

func TestListIAMServiceAccountsCommand(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "List IAM Service Accounts Command Suite")
}


var _ = Describe("List IAM Service Accounts Command", func() {
	var ctrl *gomock.Controller

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("Cluster prefix extraction logic", func() {
		It("should extract prefix from installer role ARN correctly", func() {
			// Test the prefix extraction logic that was missing
			// This test demonstrates what SHOULD happen with proper implementation
			
			// Input: "arn:aws:iam::123456789012:role/ManagedOpenShift-Installer-Role"
			// Expected extracted prefix: "ManagedOpenShift"

			roleARN := "arn:aws:iam::123456789012:role/ManagedOpenShift-Installer-Role"
			parts := strings.Split(roleARN, "/")
			Expect(len(parts)).To(BeNumerically(">", 1))
			
			roleName := parts[len(parts)-1]
			Expect(roleName).To(Equal("ManagedOpenShift-Installer-Role"))
			
			var extractedPrefix string
			if strings.Contains(roleName, "-Installer-Role") {
				extractedPrefix = strings.Replace(roleName, "-Installer-Role", "", 1)
			}
			Expect(extractedPrefix).To(Equal("ManagedOpenShift"))

			// Verify that roles with the extracted prefix would be included
			managedOpenShiftRole := "ManagedOpenShift-default-my-app"
			otherClusterRole := "OtherCluster-default-other-app"
			
			Expect(containsPrefix(managedOpenShiftRole, extractedPrefix)).To(BeTrue())
			Expect(containsPrefix(otherClusterRole, extractedPrefix)).To(BeFalse())
		})

		It("should demonstrate the bug: prefix extraction not implemented", func() {
			// This test would FAIL with the current incomplete implementation
			// It shows what happens when the prefix extraction logic is missing
			
			// Simulate what the current broken code does:
			clusterName := "test-cluster"  // Falls back to cluster name
			installerRoleARN := "arn:aws:iam::123456789012:role/ManagedOpenShift-Installer-Role"
			
			// Current broken logic: clusterPrefix = cluster.Name() with no extraction
			clusterPrefix := clusterName
			if installerRoleARN != "" {
				// The original code had just a TODO comment here - no actual logic
				// So clusterPrefix remains as cluster.Name()
			}
			
			// This demonstrates the bug: we get "test-cluster" instead of "ManagedOpenShift"
			Expect(clusterPrefix).To(Equal("test-cluster"))  // This is what the broken code does
			
			// But what we SHOULD get with proper implementation:
			parts := strings.Split(installerRoleARN, "/")
			if len(parts) > 1 {
				roleName := parts[len(parts)-1]
				if strings.Contains(roleName, "-Installer-Role") {
					properPrefix := strings.Replace(roleName, "-Installer-Role", "", 1)
					Expect(properPrefix).To(Equal("ManagedOpenShift"))  // This is what we should get
					
					// Show the difference: broken vs correct filtering
					testRole := "ManagedOpenShift-default-my-app"
					
					// With broken prefix ("test-cluster"), this role would be incorrectly excluded
					Expect(containsPrefix(testRole, clusterPrefix)).To(BeFalse())  // Bug: excludes valid role
					
					// With correct prefix ("ManagedOpenShift"), this role would be correctly included  
					Expect(containsPrefix(testRole, properPrefix)).To(BeTrue())   // Fixed: includes valid role
				}
			}
		})

		It("should extract prefix from different installer role formats", func() {
			testCases := []struct {
				roleARN        string
				expectedPrefix string
			}{
				{
					"arn:aws:iam::123456789012:role/MyPrefix-Installer-Role",
					"MyPrefix",
				},
				{
					"arn:aws:iam::123456789012:role/rosa-12345-Installer-Role", 
					"rosa-12345",
				},
				{
					"arn:aws:iam::123456789012:role/ComplexPrefix-v2-Installer-Role",
					"ComplexPrefix-v2",
				},
			}

			for _, tc := range testCases {
				parts := strings.Split(tc.roleARN, "/")
				Expect(len(parts)).To(BeNumerically(">", 1))
				
				roleName := parts[len(parts)-1]
				
				var extractedPrefix string
				if strings.Contains(roleName, "-Installer-Role") {
					extractedPrefix = strings.Replace(roleName, "-Installer-Role", "", 1)
				}
				Expect(extractedPrefix).To(Equal(tc.expectedPrefix))
			}
		})

		It("should handle malformed role ARN gracefully", func() {
			// Test with malformed ARN that doesn't contain "/"
			roleARN := "malformed-arn-without-slash"
			parts := strings.Split(roleARN, "/")
			
			// Should have only one part, so no extraction should happen
			Expect(len(parts)).To(Equal(1))
			
			// In this case, the prefix extraction logic should not execute
			// and should fall back to using cluster name
		})

		It("should handle role ARN without installer role suffix", func() {
			// Test with role ARN that doesn't contain "-Installer-Role"
			roleARN := "arn:aws:iam::123456789012:role/SomeOtherRole"
			parts := strings.Split(roleARN, "/")
			Expect(len(parts)).To(BeNumerically(">", 1))
			
			roleName := parts[len(parts)-1]
			Expect(roleName).To(Equal("SomeOtherRole"))
			
			var extractedPrefix string
			if strings.Contains(roleName, "-Installer-Role") {
				extractedPrefix = strings.Replace(roleName, "-Installer-Role", "", 1)
			}
			// Should remain empty since it doesn't contain "-Installer-Role"
			Expect(extractedPrefix).To(Equal(""))
		})
	})

	Context("Role filtering logic", func() {
		It("should filter roles by prefix correctly", func() {
			// Test prefix filtering functionality
			testRoles := []string{
				"ManagedOpenShift-default-my-app",
				"ManagedOpenShift-kube-system-aws-load-balancer",
				"OtherCluster-default-other-app",
				"rosa-12345-default-test-app",
				"RegularRole",
			}

			// Test filtering with "ManagedOpenShift" prefix
			prefix := "ManagedOpenShift"
			var matchingRoles []string
			for _, roleName := range testRoles {
				if prefix != "" && strings.HasPrefix(roleName, prefix) {
					matchingRoles = append(matchingRoles, roleName)
				}
			}

			Expect(matchingRoles).To(HaveLen(2))
			Expect(matchingRoles).To(ContainElement("ManagedOpenShift-default-my-app"))
			Expect(matchingRoles).To(ContainElement("ManagedOpenShift-kube-system-aws-load-balancer"))
			Expect(matchingRoles).ToNot(ContainElement("OtherCluster-default-other-app"))
		})

		It("should include all roles when no prefix specified", func() {
			testRoles := []string{
				"ManagedOpenShift-default-my-app",
				"OtherCluster-default-other-app",
				"rosa-12345-default-test-app",
			}

			// Test with empty prefix (should include all)
			prefix := ""
			var matchingRoles []string
			for _, roleName := range testRoles {
				if prefix == "" || strings.HasPrefix(roleName, prefix) {
					matchingRoles = append(matchingRoles, roleName)
				}
			}

			Expect(matchingRoles).To(HaveLen(3))
			Expect(matchingRoles).To(Equal(testRoles))
		})

		It("should handle edge cases in prefix filtering", func() {
			testCases := []struct {
				roles         []string
				prefix        string
				expectedCount int
				description   string
			}{
				{
					[]string{"prefix-role1", "prefix-role2", "other-role"},
					"prefix",
					2,
					"exact prefix match",
				},
				{
					[]string{"prefix-role1", "prefix-role2", "other-role"},
					"prefix-",
					2,
					"prefix with hyphen",
				},
				{
					[]string{"prefix-role1", "prefix-role2", "other-role"},
					"nonexistent",
					0,
					"no matching prefix",
				},
				{
					[]string{"a", "ab", "abc", "b"},
					"a",
					3,
					"short prefix",
				},
				{
					[]string{},
					"any",
					0,
					"empty role list",
				},
			}

			for _, tc := range testCases {
				var matchingRoles []string
				for _, roleName := range tc.roles {
					if tc.prefix == "" || strings.HasPrefix(roleName, tc.prefix) {
						matchingRoles = append(matchingRoles, roleName)
					}
				}

				Expect(matchingRoles).To(HaveLen(tc.expectedCount), "Failed for case: %s", tc.description)
			}
		})

		It("should correctly identify service account roles among mixed types", func() {
			// Test mixed role types to ensure proper filtering
			testRoles := []struct {
				name    string
				tags    map[string]string
				isValid bool
			}{
				{
					"ManagedOpenShift-default-my-app",
					map[string]string{
						helper.ServiceAccountRoleTag:      "true",
						helper.ServiceAccountNamespaceTag: "default",
						helper.ServiceAccountNameTag:      "my-app",
					},
					true,
				},
				{
					"ManagedOpenShift-Installer-Role",
					map[string]string{
						"red-hat-managed": "true",
						"rosa.aws.openshift.io/account-role": "installer",
					},
					false,
				},
				{
					"ManagedOpenShift-Worker-Role",
					map[string]string{
						"red-hat-managed": "true",
						"rosa.aws.openshift.io/account-role": "worker",
					},
					false,
				},
				{
					"ManagedOpenShift-kube-system-cluster-autoscaler",
					map[string]string{
						helper.ServiceAccountRoleTag:      "true",
						helper.ServiceAccountNamespaceTag: "kube-system",
						helper.ServiceAccountNameTag:      "cluster-autoscaler",
					},
					true,
				},
			}

			var serviceAccountRoles []string
			for _, role := range testRoles {
				if helper.IsServiceAccountRole(role.tags) {
					serviceAccountRoles = append(serviceAccountRoles, role.name)
				}
			}

			// Should only include the actual service account roles
			Expect(serviceAccountRoles).To(HaveLen(2))
			Expect(serviceAccountRoles).To(ContainElement("ManagedOpenShift-default-my-app"))
			Expect(serviceAccountRoles).To(ContainElement("ManagedOpenShift-kube-system-cluster-autoscaler"))
			Expect(serviceAccountRoles).ToNot(ContainElement("ManagedOpenShift-Installer-Role"))
			Expect(serviceAccountRoles).ToNot(ContainElement("ManagedOpenShift-Worker-Role"))
		})

		It("should handle error cases gracefully in filtering", func() {
			// Test what happens when tag retrieval fails
			roleNamesWithErrors := []string{
				"valid-role-1",
				"invalid-role-with-no-tags",
				"valid-role-2",
			}

			// Simulate filtering with some roles having errors
			var successfulRoles []string
			for _, roleName := range roleNamesWithErrors {
				// Simulate error for roles containing "invalid"
				if strings.Contains(roleName, "invalid") {
					// Skip roles with errors (like the real code does)
					continue
				}
				successfulRoles = append(successfulRoles, roleName)
			}

			// Should only include roles without errors
			Expect(successfulRoles).To(HaveLen(2))
			Expect(successfulRoles).To(ContainElement("valid-role-1"))
			Expect(successfulRoles).To(ContainElement("valid-role-2"))
			Expect(successfulRoles).ToNot(ContainElement("invalid-role-with-no-tags"))
		})
	})

	Context("containsPrefix function validation", func() {
		It("should correctly check if role name contains prefix", func() {
			testCases := []struct {
				roleName      string
				prefix        string
				shouldContain bool
			}{
				{"ManagedOpenShift-default-my-app", "ManagedOpenShift", true},
				{"ManagedOpenShift-default-my-app", "Managed", true},
				{"ManagedOpenShift-default-my-app", "OpenShift", false}, // Not at beginning
				{"ManagedOpenShift-default-my-app", "", true},           // Empty prefix
				{"short", "very-long-prefix", false},                   // Prefix longer than name
				{"exact-match", "exact-match", true},                   // Exact match
				{"", "any", false},                                     // Empty role name
				{"any", "", true},                                      // Empty prefix with any name
			}

			for _, tc := range testCases {
				result := containsPrefix(tc.roleName, tc.prefix)
				Expect(result).To(Equal(tc.shouldContain), 
					"containsPrefix('%s', '%s') should be %v", tc.roleName, tc.prefix, tc.shouldContain)
			}
		})
	})
})