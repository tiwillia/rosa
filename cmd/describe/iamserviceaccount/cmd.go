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

	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/arguments"
	"github.com/openshift/rosa/pkg/helper"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
)

var args struct {
	clusterKey string
	roleName   string
}

var Cmd = &cobra.Command{
	Use:   "iam-service-account",
	Short: "Describe IAM service account role",
	Long: "Show detailed information about an IAM role created for a Kubernetes service account, " +
		"including trust policy, attached policies, and service account details.",
	Example: `  # Describe IAM role by name
  rosa describe iam-service-account --role-name=MyCluster-default-my-app

  # Describe IAM role with cluster context
  rosa describe iam-service-account --cluster=my-cluster --role-name=MyCluster-default-my-app`,
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
		"Name or ID of the cluster (optional, used for validation).",
	)

	flags.StringVar(
		&args.roleName,
		"role-name",
		"",
		"Name of the IAM role to describe.",
	)

	output.AddFlag(Cmd)
	arguments.AddRegionFlag(flags)
	arguments.AddProfileFlag(flags)
	interactive.AddFlag(flags)
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

	// Validate role name
	if args.roleName == "" {
		r.Reporter.Errorf("Role name is required")
		os.Exit(1)
	}

	// Get role details using existing AWS client function
	role, err := r.AWSClient.GetRoleByName(args.roleName)
	if err != nil {
		r.Reporter.Errorf("Failed to get role '%s': %v", args.roleName, err)
		os.Exit(1)
	}

	// Verify this is a service account role by checking tags
	tags := make(map[string]string)
	for _, tag := range role.Tags {
		if tag.Key != nil && tag.Value != nil {
			tags[*tag.Key] = *tag.Value
		}
	}

	if !helper.IsServiceAccountRole(tags) {
		r.Reporter.Errorf("Role '%s' does not appear to be a service account role. "+
			"This command is for roles created with 'rosa create iam-service-account'.", args.roleName)
		os.Exit(1)
	}

	// Get attached policies using existing AWS client function
	attachedPolicies, err := r.AWSClient.ListAttachedRolePolicies(args.roleName)
	if err != nil {
		r.Reporter.Errorf("Failed to get attached policies for role '%s': %v", args.roleName, err)
		os.Exit(1)
	}

	// Extract service account information from tags
	namespace := tags[helper.ServiceAccountNamespaceTag]
	serviceAccountName := tags[helper.ServiceAccountNameTag]

	// Create detailed role information structure
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
		RoleName:           *role.RoleName,
		RoleArn:            *role.Arn,
		Namespace:          namespace,
		ServiceAccountName: serviceAccountName,
		Path:               *role.Path,
		MaxSessionDuration: *role.MaxSessionDuration,
		CreationDate:       role.CreateDate.Format("2006-01-02T15:04:05Z"),
		TrustPolicy:        *role.AssumeRolePolicyDocument,
		AttachedPolicies:   attachedPolicies,
		Tags:               tags,
	}

	if output.HasFlag() {
		err = output.Print(roleDetails)
		if err != nil {
			r.Reporter.Errorf("%s", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Print formatted output
	fmt.Printf("IAM Service Account Role Details:\n\n")
	fmt.Printf("Role Name:              %s\n", roleDetails.RoleName)
	fmt.Printf("Role ARN:               %s\n", roleDetails.RoleArn)
	fmt.Printf("Service Account:        %s/%s\n", roleDetails.Namespace, roleDetails.ServiceAccountName)
	fmt.Printf("Path:                   %s\n", roleDetails.Path)
	fmt.Printf("Max Session Duration:   %d seconds\n", roleDetails.MaxSessionDuration)
	fmt.Printf("Created:                %s\n", roleDetails.CreationDate)

	fmt.Printf("\nTrust Policy:\n")
	fmt.Printf("%s\n", roleDetails.TrustPolicy)

	fmt.Printf("\nAttached Policies:\n")
	if len(roleDetails.AttachedPolicies) == 0 {
		fmt.Printf("  No policies attached\n")
	} else {
		for _, policy := range roleDetails.AttachedPolicies {
			fmt.Printf("  - %s\n", policy)
		}
	}

	fmt.Printf("\nTags:\n")
	if len(roleDetails.Tags) == 0 {
		fmt.Printf("  No tags\n")
	} else {
		for key, value := range roleDetails.Tags {
			fmt.Printf("  %s: %s\n", key, value)
		}
	}

	fmt.Printf("\nUsage:\n")
	fmt.Printf("To use this role with a Kubernetes service account, annotate the service account with:\n")
	fmt.Printf("  eks.amazonaws.com/role-arn: %s\n", roleDetails.RoleArn)
	fmt.Printf("\nExample kubectl command:\n")
	fmt.Printf("  kubectl annotate serviceaccount %s -n %s eks.amazonaws.com/role-arn=%s\n",
		roleDetails.ServiceAccountName, roleDetails.Namespace, roleDetails.RoleArn)
}

func validateArgs() error {
	if args.roleName == "" && !interactive.Enabled() {
		return fmt.Errorf("role name is required")
	}
	return nil
}

func runInteractive(r *rosa.Runtime) error {
	var err error

	// Role name
	if args.roleName == "" {
		args.roleName, err = interactive.GetString(interactive.Input{
			Question: "IAM role name",
			Help:     "Name of the IAM role to describe.",
			Required: true,
		})
		if err != nil {
			return err
		}
	}

	// Cluster (optional for validation)
	if args.clusterKey == "" {
		args.clusterKey, err = interactive.GetString(interactive.Input{
			Question: "Cluster name or ID (optional)",
			Help:     "Name or ID of the cluster for validation (optional).",
			Required: false,
		})
		if err != nil {
			return err
		}
	}

	return nil
}
