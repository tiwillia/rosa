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
	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/rosa"
)

var args struct {
	clusterKey string
	roleName   string
}

var Cmd = &cobra.Command{
	Use:   "iam-service-account",
	Short: "Delete IAM role for Kubernetes service account",
	Long: "Deletes an IAM role that was created for a Kubernetes service account. " +
		"This will also detach any attached policies before deleting the role.",
	Example: `  # Delete IAM role by name
  rosa delete iam-service-account --role-name=MyCluster-default-my-app

  # Delete IAM role with cluster context
  rosa delete iam-service-account --cluster=my-cluster --role-name=MyCluster-default-my-app`,
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
		"Name of the IAM role to delete.",
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

	// Validate role name
	if args.roleName == "" {
		r.Reporter.Errorf("Role name is required")
		os.Exit(1)
	}

	// Check if role exists and verify it's a service account role
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
			"Only roles created with 'rosa create iam-service-account' should be deleted with this command.", args.roleName)
		os.Exit(1)
	}

	// Get namespace and service account name from tags for display
	namespace := tags[helper.ServiceAccountNamespaceTag]
	serviceAccountName := tags[helper.ServiceAccountNameTag]

	r.Reporter.Infof("Role '%s' is configured for service account '%s/%s'", args.roleName, namespace, serviceAccountName)

	// Confirm deletion
	if !confirm.Prompt(true, "Are you sure you want to delete the IAM role '%s'?", args.roleName) {
		os.Exit(0)
	}

	r.Reporter.Infof("Deleting IAM role '%s'", args.roleName)

	// Delete role using existing AWS client function (handles policy detachment automatically)
	err = r.AWSClient.DeleteRole(args.roleName)
	if err != nil {
		r.Reporter.Errorf("Failed to delete IAM role '%s': %v", args.roleName, err)
		os.Exit(1)
	}

	r.Reporter.Infof("Successfully deleted IAM role '%s'", args.roleName)
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
			Help:     "Name of the IAM role to delete.",
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
