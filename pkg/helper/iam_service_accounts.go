package helper

import (
	"fmt"
)

const (
	ServiceAccountRoleTag      = "rosa.openshift.io/service-account-role"
	ServiceAccountNamespaceTag = "rosa.openshift.io/service-account-namespace"
	ServiceAccountNameTag      = "rosa.openshift.io/service-account-name"
	OIDCTrustPolicyTemplate    = `{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Principal": {
                "Federated": "%s"
            },
            "Action": "sts:AssumeRoleWithWebIdentity",
            "Condition": {
                "StringEquals": {
                    "%s:sub": "system:serviceaccount:%s:%s",
                    "%s:aud": "sts.amazonaws.com"
                }
            }
        }
    ]
}`
)

// GenerateOIDCTrustPolicy creates OIDC trust policy for service account
func GenerateOIDCTrustPolicy(issuerURL, namespace, serviceAccountName string) string {
	return fmt.Sprintf(OIDCTrustPolicyTemplate,
		issuerURL, issuerURL, namespace, serviceAccountName, issuerURL)
}

// GenerateServiceAccountTags creates standard tags for service account roles
func GenerateServiceAccountTags(namespace, serviceAccountName string) map[string]string {
	return map[string]string{
		ServiceAccountRoleTag:      "true",
		ServiceAccountNamespaceTag: namespace,
		ServiceAccountNameTag:      serviceAccountName,
	}
}

// IsServiceAccountRole checks if role has service account tags
func IsServiceAccountRole(tags map[string]string) bool {
	_, exists := tags[ServiceAccountRoleTag]
	return exists
}

// GenerateServiceAccountRoleName creates a standardized role name
func GenerateServiceAccountRoleName(prefix, namespace, serviceAccountName string) string {
	if prefix == "" {
		return fmt.Sprintf("%s-%s", namespace, serviceAccountName)
	}
	return fmt.Sprintf("%s-%s-%s", prefix, namespace, serviceAccountName)
}

// ValidateServiceAccountRoleInputs validates user inputs
func ValidateServiceAccountRoleInputs(namespace, serviceAccountName, roleName string) error {
	if namespace == "" {
		return fmt.Errorf("namespace cannot be empty")
	}
	if serviceAccountName == "" {
		return fmt.Errorf("service account name cannot be empty")
	}
	if roleName == "" {
		return fmt.Errorf("role name cannot be empty")
	}

	// Validate namespace format (Kubernetes DNS-1123 label)
	if !isValidKubernetesName(namespace) {
		return fmt.Errorf("namespace '%s' is not a valid Kubernetes namespace name", namespace)
	}

	// Validate service account name format
	if !isValidKubernetesName(serviceAccountName) {
		return fmt.Errorf("service account name '%s' is not a valid Kubernetes service account name", serviceAccountName)
	}

	return nil
}


// isValidKubernetesName validates if a string is a valid Kubernetes DNS-1123 label
func isValidKubernetesName(name string) bool {
	if len(name) == 0 || len(name) > 63 {
		return false
	}

	// Must start and end with alphanumeric character
	if !isAlphaNumeric(name[0]) || !isAlphaNumeric(name[len(name)-1]) {
		return false
	}

	// Can contain lowercase letters, numbers, and hyphens
	for _, char := range name {
		if !isAlphaNumeric(byte(char)) && char != '-' {
			return false
		}
		// No uppercase letters allowed
		if char >= 'A' && char <= 'Z' {
			return false
		}
	}

	return true
}

// isAlphaNumeric checks if a byte is alphanumeric
func isAlphaNumeric(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9')
}
