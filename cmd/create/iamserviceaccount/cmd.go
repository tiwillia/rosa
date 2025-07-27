/*
Copyright (c) 2024 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package iamserviceaccount

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/arguments"
	"github.com/openshift/rosa/pkg/helper"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/rosa"
)

var args struct {
	clusterKey          string
	namespace           string
	serviceAccountName  string
	roleName            string
	policyArns          []string
	path                string
	permissionsBoundary string
	managedPolicies     bool
}

var Cmd = &cobra.Command{
	Use:   "iam-service-account",
	Short: "Create IAM role for Kubernetes service account",
	Long: "Creates an IAM role with OIDC trust policy that allows a Kubernetes service account " +
		"to assume the role using AWS STS AssumeRoleWithWebIdentity.",
	Example: `  # Create IAM role for a service account
  rosa create iam-service-account --cluster=my-cluster --namespace=default --service-account=my-app

  # Create IAM role with custom role name and policies
  rosa create iam-service-account --cluster=my-cluster --namespace=kube-system --service-account=aws-load-balancer-controller --role-name=MyCluster-AWSLoadBalancerController --policy-arns=arn:aws:iam::123456789012:policy/AWSLoadBalancerControllerIAMPolicy`,
	Args: cobra.NoArgs,
	Run:  run,
}

func init() {
	flags := Cmd.Flags()

	flags.StringVarP(
		&args.clusterKey,
		"cluster",
		"c",
		"",
		"Name or ID of the cluster to create the service account role for.",
	)

	flags.StringVar(
		&args.namespace,
		"namespace",
		"",
		"Kubernetes namespace where the service account is located.",
	)

	flags.StringVar(
		&args.serviceAccountName,
		"service-account",
		"",
		"Name of the Kubernetes service account.",
	)

	flags.StringVar(
		&args.roleName,
		"role-name",
		"",
		"Name of the IAM role to create. If not specified, a name will be generated.",
	)

	flags.StringSliceVar(
		&args.policyArns,
		"policy-arns",
		nil,
		"ARNs of IAM policies to attach to the role (comma-separated).",
	)

	flags.StringVar(
		&args.path,
		"path",
		"/",
		"Path for the IAM role.",
	)

	flags.StringVar(
		&args.permissionsBoundary,
		"permissions-boundary",
		"",
		"ARN of the permissions boundary policy to apply to the role.",
	)

	flags.BoolVar(
		&args.managedPolicies,
		"managed-policies",
		false,
		"Use AWS managed policies instead of inline policies.",
	)

	arguments.AddRegionFlag(flags)
	arguments.AddProfileFlag(flags)
	interactive.AddFlag(flags)
	confirm.AddFlag(flags)
	interactive.AddModeFlag(Cmd)
}

func run(cmd *cobra.Command, argv []string) {
	r := rosa.NewRuntime().WithAWS().WithOCM()
	defer r.Cleanup()

	// Validate required arguments
	err := validateArgs()
	if err != nil {
		r.Reporter.Errorf("%s", err)
		os.Exit(1)
	}

	// Interactive mode
	if interactive.Enabled() {
		err = runInteractive(r)
		if err != nil {
			r.Reporter.Errorf("%s", err)
			os.Exit(1)
		}
	}

	// Validate inputs
	err = helper.ValidateServiceAccountRoleInputs(args.namespace, args.serviceAccountName, args.roleName)
	if err != nil {
		r.Reporter.Errorf("%s", err)
		os.Exit(1)
	}

	// Get cluster
	cluster, err := r.OCMClient.GetCluster(args.clusterKey, r.Creator)
	if err != nil {
		r.Reporter.Errorf("Failed to get cluster '%s': %v", args.clusterKey, err)
		os.Exit(1)
	}

	if cluster.AWS().STS().OidcConfig() == nil || cluster.AWS().STS().OidcConfig().IssuerUrl() == "" {
		r.Reporter.Errorf("Cluster '%s' does not have OIDC configuration. OIDC is required for service account roles.", cluster.Name())
		os.Exit(1)
	}

	// Generate role name if not provided
	if args.roleName == "" {
		prefix := cluster.AWS().STS().RoleARN()
		if prefix != "" {
			// Extract prefix from existing role ARN
			parts := strings.Split(prefix, "/")
			if len(parts) > 1 {
				roleName := parts[len(parts)-1]
				if strings.Contains(roleName, "-Installer-Role") {
					prefix = strings.Replace(roleName, "-Installer-Role", "", 1)
				}
			}
		}
		if prefix == "" {
			prefix = cluster.Name()
		}
		args.roleName = helper.GenerateServiceAccountRoleName(prefix, args.namespace, args.serviceAccountName)
	}

	// Check if role already exists
	existingRole, err := r.AWSClient.GetRoleByName(args.roleName)
	if err == nil {
		r.Reporter.Warnf("Role '%s' already exists with ARN: %s", args.roleName, *existingRole.Arn)
		if !confirm.Confirm("continue") {
			os.Exit(0)
		}
	}

	// Generate OIDC trust policy
	issuerURL := cluster.AWS().STS().OidcConfig().IssuerUrl()
	trustPolicy := helper.GenerateOIDCTrustPolicy(issuerURL, args.namespace, args.serviceAccountName)

	// Generate tags
	tags := helper.GenerateServiceAccountTags(args.namespace, args.serviceAccountName)

	r.Reporter.Infof("Creating IAM role '%s' for service account '%s/%s'", args.roleName, args.namespace, args.serviceAccountName)

	// Create role using existing AWS client function
	roleARN, err := r.AWSClient.EnsureRole(r.Reporter, args.roleName, trustPolicy, args.permissionsBoundary,
		"", tags, args.path, args.managedPolicies)
	if err != nil {
		r.Reporter.Errorf("Failed to create IAM role: %v", err)
		os.Exit(1)
	}

	// Attach policies if specified
	if len(args.policyArns) > 0 {
		r.Reporter.Infof("Attaching policies to role '%s'", args.roleName)
		for _, policyArn := range args.policyArns {
			err = r.AWSClient.AttachRolePolicy(r.Reporter, args.roleName, strings.TrimSpace(policyArn))
			if err != nil {
				r.Reporter.Errorf("Failed to attach policy '%s' to role '%s': %v", policyArn, args.roleName, err)
				os.Exit(1)
			}
		}
	}

	r.Reporter.Infof("Successfully created IAM role '%s' with ARN: %s", args.roleName, roleARN)
	r.Reporter.Infof("To use this role with a Kubernetes service account, annotate the service account with:")
	r.Reporter.Infof("  eks.amazonaws.com/role-arn: %s", roleARN)
}

func validateArgs() error {
	if args.clusterKey == "" {
		return fmt.Errorf("cluster name or ID is required")
	}
	if args.namespace == "" {
		return fmt.Errorf("namespace is required")
	}
	if args.serviceAccountName == "" {
		return fmt.Errorf("service account name is required")
	}
	return nil
}

func runInteractive(r *rosa.Runtime) error {
	var err error

	// Cluster
	if args.clusterKey == "" {
		args.clusterKey, err = interactive.GetString(interactive.Input{
			Question: "Cluster name or ID",
			Help:     "Name or ID of the cluster to create the service account role for.",
			Required: true,
		})
		if err != nil {
			return err
		}
	}

	// Namespace
	if args.namespace == "" {
		args.namespace, err = interactive.GetString(interactive.Input{
			Question: "Kubernetes namespace",
			Help:     "Kubernetes namespace where the service account is located.",
			Default:  "default",
			Required: true,
		})
		if err != nil {
			return err
		}
	}

	// Service account name
	if args.serviceAccountName == "" {
		args.serviceAccountName, err = interactive.GetString(interactive.Input{
			Question: "Service account name",
			Help:     "Name of the Kubernetes service account.",
			Required: true,
		})
		if err != nil {
			return err
		}
	}

	// Role name (optional)
	if args.roleName == "" {
		args.roleName, err = interactive.GetString(interactive.Input{
			Question: "IAM role name (optional)",
			Help:     "Name of the IAM role to create. If not specified, a name will be generated.",
			Required: false,
		})
		if err != nil {
			return err
		}
	}

	// Policy ARNs (optional)
	if len(args.policyArns) == 0 {
		policyArnsStr, err := interactive.GetString(interactive.Input{
			Question: "Policy ARNs (optional)",
			Help:     "Comma-separated list of IAM policy ARNs to attach to the role.",
			Required: false,
		})
		if err != nil {
			return err
		}
		if policyArnsStr != "" {
			args.policyArns = strings.Split(policyArnsStr, ",")
		}
	}

	// Permissions boundary (optional)
	if args.permissionsBoundary == "" {
		args.permissionsBoundary, err = interactive.GetString(interactive.Input{
			Question: "Permissions boundary ARN (optional)",
			Help:     "ARN of the permissions boundary policy to apply to the role.",
			Required: false,
		})
		if err != nil {
			return err
		}
	}

	return nil
}
