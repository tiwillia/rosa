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

package iamserviceaccounts

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/briandowns/spinner"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/arguments"
	"github.com/openshift/rosa/pkg/helper"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
)

var args struct {
	clusterKey string
	prefix     string
}

var Cmd = &cobra.Command{
	Use:     "iam-service-accounts",
	Aliases: []string{"iam-service-account", "iamserviceaccounts", "iamserviceaccount"},
	Short:   "List IAM service account roles",
	Long:    "List IAM service account roles created for Kubernetes service accounts.",
	Example: `  # List all service account roles
  rosa list iam-service-accounts

  # List service account roles with a specific prefix
  rosa list iam-service-accounts --prefix=MyCluster

  # List service account roles for a specific cluster
  rosa list iam-service-accounts --cluster=my-cluster`,
	Args: cobra.NoArgs,
	Run:  run,
}

func init() {
	flags := Cmd.Flags()
	flags.SortFlags = false

	flags.StringVarP(
		&args.clusterKey,
		"cluster",
		"c",
		"",
		"Name or ID of the cluster to filter service account roles for.",
	)

	flags.StringVar(
		&args.prefix,
		"prefix",
		"",
		"Prefix to filter role names by.",
	)

	output.AddFlag(Cmd)
	arguments.AddRegionFlag(flags)
	arguments.AddProfileFlag(flags)
	interactive.AddModeFlag(Cmd)
}

func run(cmd *cobra.Command, argv []string) {
	r := rosa.NewRuntime().WithAWS().WithOCM()
	defer r.Cleanup()

	var spin *spinner.Spinner
	if r.Reporter.IsTerminal() {
		spin = spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	}
	if spin != nil {
		r.Reporter.Infof("Fetching IAM service account roles")
		spin.Start()
	}

	// Get all IAM roles using existing AWS client function
	allRoles, err := r.AWSClient.ListRoles()
	if spin != nil {
		spin.Stop()
	}

	if err != nil {
		r.Reporter.Errorf("Failed to get IAM roles: %v", err)
		os.Exit(1)
	}

	// Filter to only service account roles
	serviceAccountRoles, err := filterServiceAccountRoles(r, allRoles, args.prefix)
	if err != nil {
		r.Reporter.Errorf("Failed to filter service account roles: %v", err)
		os.Exit(1)
	}

	// Additional filtering by cluster if specified
	if args.clusterKey != "" {
		cluster, err := r.OCMClient.GetCluster(args.clusterKey, r.Creator)
		if err != nil {
			r.Reporter.Errorf("Failed to get cluster '%s': %v", args.clusterKey, err)
			os.Exit(1)
		}

		clusterPrefix := cluster.Name()
		if cluster.AWS().STS().RoleARN() != "" {
			// Extract prefix from existing role ARN for more accurate filtering
			// This matches the logic used in the create command
			roleARN := cluster.AWS().STS().RoleARN()
			parts := strings.Split(roleARN, "/")
			if len(parts) > 1 {
				roleName := parts[len(parts)-1]
				if strings.Contains(roleName, "-Installer-Role") {
					clusterPrefix = strings.Replace(roleName, "-Installer-Role", "", 1)
				}
			}
		}

		var filteredRoles []struct {
			RoleName           string
			RoleArn            string
			Namespace          string
			ServiceAccountName string
			CreationDate       time.Time
		}

		for _, role := range serviceAccountRoles {
			// Get role tags to extract namespace and service account info
			tags, err := getRoleTags(r, *role.RoleName)
			if err != nil {
				continue
			}

			// Check if role belongs to this cluster (basic prefix matching)
			if clusterPrefix != "" && !containsPrefix(*role.RoleName, clusterPrefix) {
				continue
			}

			namespace := tags[helper.ServiceAccountNamespaceTag]
			serviceAccountName := tags[helper.ServiceAccountNameTag]

			filteredRoles = append(filteredRoles, struct {
				RoleName           string
				RoleArn            string
				Namespace          string
				ServiceAccountName string
				CreationDate       time.Time
			}{
				RoleName:           *role.RoleName,
				RoleArn:            *role.Arn,
				Namespace:          namespace,
				ServiceAccountName: serviceAccountName,
				CreationDate:       *role.CreateDate,
			})
		}

		if output.HasFlag() {
			err = output.Print(filteredRoles)
			if err != nil {
				r.Reporter.Errorf("%s", err)
				os.Exit(1)
			}
			os.Exit(0)
		}

		if len(filteredRoles) == 0 {
			r.Reporter.Infof("No IAM service account roles found for cluster '%s'", cluster.Name())
			os.Exit(0)
		}

		// Create tabulated output
		writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintf(writer, "ROLE NAME\tNAMESPACE\tSERVICE ACCOUNT\tROLE ARN\tCREATED\n")
		for _, role := range filteredRoles {
			fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\n",
				role.RoleName,
				role.Namespace,
				role.ServiceAccountName,
				role.RoleArn,
				role.CreationDate.Format(time.RFC3339))
		}
		writer.Flush()
		return
	}

	// Convert to output format for general listing
	var roleList []struct {
		RoleName           string
		RoleArn            string
		Namespace          string
		ServiceAccountName string
		CreationDate       time.Time
	}

	for _, role := range serviceAccountRoles {
		// Get role tags to extract namespace and service account info
		tags, err := getRoleTags(r, *role.RoleName)
		if err != nil {
			// Skip roles we can't get tags for
			continue
		}

		namespace := tags[helper.ServiceAccountNamespaceTag]
		serviceAccountName := tags[helper.ServiceAccountNameTag]

		roleList = append(roleList, struct {
			RoleName           string
			RoleArn            string
			Namespace          string
			ServiceAccountName string
			CreationDate       time.Time
		}{
			RoleName:           *role.RoleName,
			RoleArn:            *role.Arn,
			Namespace:          namespace,
			ServiceAccountName: serviceAccountName,
			CreationDate:       *role.CreateDate,
		})
	}

	if output.HasFlag() {
		err = output.Print(roleList)
		if err != nil {
			r.Reporter.Errorf("%s", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	if len(roleList) == 0 {
		r.Reporter.Infof("No IAM service account roles found")
		os.Exit(0)
	}

	// Create tabulated output
	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(writer, "ROLE NAME\tNAMESPACE\tSERVICE ACCOUNT\tROLE ARN\tCREATED\n")
	for _, role := range roleList {
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\n",
			role.RoleName,
			role.Namespace,
			role.ServiceAccountName,
			role.RoleArn,
			role.CreationDate.Format(time.RFC3339))
	}
	writer.Flush()
}

// getRoleTags gets tags for a role (helper function)
func getRoleTags(r *rosa.Runtime, roleName string) (map[string]string, error) {
	role, err := r.AWSClient.GetRoleByName(roleName)
	if err != nil {
		return nil, err
	}

	tags := make(map[string]string)
	for _, tag := range role.Tags {
		if tag.Key != nil && tag.Value != nil {
			tags[*tag.Key] = *tag.Value
		}
	}

	return tags, nil
}

// filterServiceAccountRoles filters roles by service account tags
func filterServiceAccountRoles(r *rosa.Runtime, roles []iamtypes.Role, prefix string) ([]iamtypes.Role, error) {
	var serviceAccountRoles []iamtypes.Role

	for _, role := range roles {
		roleName := *role.RoleName

		// Skip roles that don't match prefix if specified
		if prefix != "" && !strings.HasPrefix(roleName, prefix) {
			continue
		}

		// Get role tags to check if it's a service account role
		tags, err := getRoleTags(r, roleName)
		if err != nil {
			// Skip roles we can't get tags for
			continue
		}

		// Check if this is a service account role
		if helper.IsServiceAccountRole(tags) {
			serviceAccountRoles = append(serviceAccountRoles, role)
		}
	}

	return serviceAccountRoles, nil
}

// containsPrefix checks if a role name contains the given prefix
func containsPrefix(roleName, prefix string) bool {
	if prefix == "" {
		return true
	}
	// Simple prefix check - could be made more sophisticated
	return len(roleName) >= len(prefix) && roleName[:len(prefix)] == prefix
}
