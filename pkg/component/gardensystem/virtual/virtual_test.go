// Copyright 2023 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package virtual_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	resourcesv1alpha1 "github.com/gardener/gardener/pkg/apis/resources/v1alpha1"
	"github.com/gardener/gardener/pkg/component"
	. "github.com/gardener/gardener/pkg/component/gardensystem/virtual"
	componenttest "github.com/gardener/gardener/pkg/component/test"
	operatorclient "github.com/gardener/gardener/pkg/operator/client"
	"github.com/gardener/gardener/pkg/resourcemanager/controller/garbagecollector/references"
	"github.com/gardener/gardener/pkg/utils/retry"
	retryfake "github.com/gardener/gardener/pkg/utils/retry/fake"
	"github.com/gardener/gardener/pkg/utils/test"
	. "github.com/gardener/gardener/pkg/utils/test/matchers"
)

var _ = Describe("Virtual", func() {
	var (
		ctx = context.TODO()

		managedResourceName = "garden-system-virtual"
		namespace           = "some-namespace"

		c         client.Client
		component component.DeployWaiter
		values    Values

		managedResource       *resourcesv1alpha1.ManagedResource
		managedResourceSecret *corev1.Secret

		namespaceGarden                                   *corev1.Namespace
		clusterRoleSeedBootstrapper                       *rbacv1.ClusterRole
		clusterRoleBindingSeedBootstrapper                *rbacv1.ClusterRoleBinding
		clusterRoleSeeds                                  *rbacv1.ClusterRole
		clusterRoleBindingSeeds                           *rbacv1.ClusterRoleBinding
		clusterRoleGardenerAdmin                          *rbacv1.ClusterRole
		clusterRoleBindingGardenerAdmin                   *rbacv1.ClusterRoleBinding
		clusterRoleGardenerAdminAggregated                *rbacv1.ClusterRole
		clusterRoleGardenerViewer                         *rbacv1.ClusterRole
		clusterRoleGardenerViewerAggregated               *rbacv1.ClusterRole
		clusterRoleReadGlobalResources                    *rbacv1.ClusterRole
		clusterRoleBindingReadGlobalResources             *rbacv1.ClusterRoleBinding
		clusterRoleUserAuth                               *rbacv1.ClusterRole
		clusterRoleBindingUserAuth                        *rbacv1.ClusterRoleBinding
		clusterRoleProjectCreation                        *rbacv1.ClusterRole
		clusterRoleProjectMember                          *rbacv1.ClusterRole
		clusterRoleProjectMemberAggregated                *rbacv1.ClusterRole
		clusterRoleProjectServiceAccountManager           *rbacv1.ClusterRole
		clusterRoleProjectServiceAccountManagerAggregated *rbacv1.ClusterRole
		clusterRoleProjectViewer                          *rbacv1.ClusterRole
		clusterRoleProjectViewerAggregated                *rbacv1.ClusterRole
		roleReadClusterIdentityConfigMap                  *rbacv1.Role
		roleBindingReadClusterIdentityConfigMap           *rbacv1.RoleBinding
	)

	BeforeEach(func() {
		c = fakeclient.NewClientBuilder().WithScheme(operatorclient.RuntimeScheme).Build()
		values = Values{}
		component = New(c, namespace, values)

		managedResource = &resourcesv1alpha1.ManagedResource{
			ObjectMeta: metav1.ObjectMeta{
				Name:      managedResourceName,
				Namespace: namespace,
			},
		}
		managedResourceSecret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "managedresource-" + managedResource.Name,
				Namespace: namespace,
			},
		}

		namespaceGarden = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "garden",
				Labels:      map[string]string{"app": "gardener"},
				Annotations: map[string]string{"resources.gardener.cloud/keep-object": "true"},
			},
		}
		clusterRoleSeedBootstrapper = &rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{
				Name: "gardener.cloud:system:seed-bootstrapper",
			},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{"certificates.k8s.io"},
					Resources: []string{"certificatesigningrequests"},
					Verbs:     []string{"create", "get"},
				},
				{
					APIGroups: []string{"certificates.k8s.io"},
					Resources: []string{"certificatesigningrequests/seedclient"},
					Verbs:     []string{"create"},
				},
			},
		}
		clusterRoleBindingSeedBootstrapper = &rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: "gardener.cloud:system:seed-bootstrapper",
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     "gardener.cloud:system:seed-bootstrapper",
			},
			Subjects: []rbacv1.Subject{{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "Group",
				Name:     "system:bootstrappers",
			}},
		}
		clusterRoleSeeds = &rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{
				Name: "gardener.cloud:system:seeds",
			},
			Rules: []rbacv1.PolicyRule{{
				APIGroups: []string{"*"},
				Resources: []string{"*"},
				Verbs:     []string{"*"},
			}},
		}
		clusterRoleBindingSeeds = &rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: "gardener.cloud:system:seeds",
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     "gardener.cloud:system:seeds",
			},
			Subjects: []rbacv1.Subject{{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "Group",
				Name:     "gardener.cloud:system:seeds",
			}},
		}
		clusterRoleGardenerAdmin = &rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "gardener.cloud:admin",
				Labels: map[string]string{"gardener.cloud/role": "admin"},
			},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{
						"core.gardener.cloud",
						"seedmanagement.gardener.cloud",
						"dashboard.gardener.cloud",
						"settings.gardener.cloud",
						"operations.gardener.cloud",
					},
					Resources: []string{"*"},
					Verbs:     []string{"*"},
				},
				{
					APIGroups: []string{""},
					Resources: []string{"events", "namespaces", "resourcequotas"},
					Verbs:     []string{"*"},
				},
				{
					APIGroups: []string{"events.k8s.io"},
					Resources: []string{"events"},
					Verbs:     []string{"*"},
				},
				{
					APIGroups: []string{"rbac.authorization.k8s.io"},
					Resources: []string{"clusterroles", "clusterrolebindings", "roles", "rolebindings"},
					Verbs:     []string{"*"},
				},
				{
					APIGroups: []string{"admissionregistration.k8s.io"},
					Resources: []string{"mutatingwebhookconfigurations", "validatingwebhookconfigurations"},
					Verbs:     []string{"*"},
				},
				{
					APIGroups: []string{"apiregistration.k8s.io"},
					Resources: []string{"apiservices"},
					Verbs:     []string{"*"},
				},
				{
					APIGroups: []string{"apiextensions.k8s.io"},
					Resources: []string{"customresourcedefinitions"},
					Verbs:     []string{"*"},
				},
				{
					APIGroups: []string{"coordination.k8s.io"},
					Resources: []string{"leases"},
					Verbs:     []string{"*"},
				},
				{
					APIGroups: []string{"certificates.k8s.io"},
					Resources: []string{"certificatesigningrequests"},
					Verbs:     []string{"*"},
				},
			},
		}
		clusterRoleBindingGardenerAdmin = &rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: "gardener.cloud:admin",
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     "gardener.cloud:admin",
			},
			Subjects: []rbacv1.Subject{{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "User",
				Name:     "system:kube-aggregator",
			}},
		}
		clusterRoleGardenerAdminAggregated = &rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{
				Name: "gardener.cloud:system:administrators",
			},
			AggregationRule: &rbacv1.AggregationRule{
				ClusterRoleSelectors: []metav1.LabelSelector{
					{MatchLabels: map[string]string{"gardener.cloud/role": "admin"}},
					{MatchLabels: map[string]string{"gardener.cloud/role": "project-member"}},
					{MatchLabels: map[string]string{"gardener.cloud/role": "project-serviceaccountmanager"}},
				},
			},
		}
		clusterRoleGardenerViewer = &rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "gardener.cloud:viewer",
				Labels: map[string]string{"gardener.cloud/role": "viewer"},
			},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{"core.gardener.cloud"},
					Resources: []string{
						"backupbuckets",
						"backupentries",
						"cloudprofiles",
						"controllerinstallations",
						"quotas",
						"projects",
						"seeds",
						"shoots",
					},
					Verbs: []string{"get", "list", "watch"},
				},
				{
					APIGroups: []string{
						"seedmanagement.gardener.cloud",
						"dashboard.gardener.cloud",
						"settings.gardener.cloud",
						"operations.gardener.cloud",
					},
					Resources: []string{"*"},
					Verbs:     []string{"get", "list", "watch"},
				},
				{
					APIGroups: []string{""},
					Resources: []string{"events", "namespaces", "resourcequotas"},
					Verbs:     []string{"get", "list", "watch"},
				},
				{
					APIGroups: []string{"events.k8s.io"},
					Resources: []string{"events"},
					Verbs:     []string{"get", "list", "watch"},
				},
				{
					APIGroups: []string{"rbac.authorization.k8s.io"},
					Resources: []string{"clusterroles", "clusterrolebindings", "roles", "rolebindings"},
					Verbs:     []string{"get", "list", "watch"},
				},
				{
					APIGroups: []string{"admissionregistration.k8s.io"},
					Resources: []string{"mutatingwebhookconfigurations", "validatingwebhookconfigurations"},
					Verbs:     []string{"get", "list", "watch"},
				},
				{
					APIGroups: []string{"apiregistration.k8s.io"},
					Resources: []string{"apiservices"},
					Verbs:     []string{"get", "list", "watch"},
				},
				{
					APIGroups: []string{"apiextensions.k8s.io"},
					Resources: []string{"customresourcedefinitions"},
					Verbs:     []string{"get", "list", "watch"},
				},
				{
					APIGroups: []string{"coordination.k8s.io"},
					Resources: []string{"leases"},
					Verbs:     []string{"get", "list", "watch"},
				},
			},
		}
		clusterRoleGardenerViewerAggregated = &rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{
				Name: "gardener.cloud:system:viewers",
			},
			AggregationRule: &rbacv1.AggregationRule{
				ClusterRoleSelectors: []metav1.LabelSelector{
					{MatchLabels: map[string]string{"gardener.cloud/role": "viewer"}},
					{MatchLabels: map[string]string{"gardener.cloud/role": "project-viewer"}},
				},
			},
		}
		clusterRoleReadGlobalResources = &rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{
				Name: "gardener.cloud:system:read-global-resources",
			},
			Rules: []rbacv1.PolicyRule{{
				APIGroups: []string{"core.gardener.cloud"},
				Resources: []string{
					"cloudprofiles",
					"exposureclasses",
					"seeds",
				},
				Verbs: []string{"get", "list", "watch"},
			}},
		}
		clusterRoleBindingReadGlobalResources = &rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: "gardener.cloud:system:read-global-resources",
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     "gardener.cloud:system:read-global-resources",
			},
			Subjects: []rbacv1.Subject{{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "Group",
				Name:     "system:authenticated",
			}},
		}
		clusterRoleUserAuth = &rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{
				Name: "gardener.cloud:system:user-auth",
			},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{"authentication.k8s.io"},
					Resources: []string{"tokenreviews"},
					Verbs:     []string{"create"},
				},
				{
					APIGroups: []string{"authorization.k8s.io"},
					Resources: []string{"selfsubjectaccessreviews"},
					Verbs:     []string{"create"},
				},
			},
		}
		clusterRoleBindingUserAuth = &rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: "gardener.cloud:system:user-auth",
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     "gardener.cloud:system:user-auth",
			},
			Subjects: []rbacv1.Subject{{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "Group",
				Name:     "system:authenticated",
			}},
		}
		clusterRoleProjectCreation = &rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{
				Name: "gardener.cloud:system:project-creation",
			},
			Rules: []rbacv1.PolicyRule{{
				APIGroups: []string{"core.gardener.cloud"},
				Resources: []string{"projects"},
				Verbs:     []string{"create"},
			}},
		}
		clusterRoleProjectMember = &rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "gardener.cloud:system:project-member-aggregation",
				Labels: map[string]string{"rbac.gardener.cloud/aggregate-to-project-member": "true"},
			},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{
						"secrets",
						"configmaps",
					},
					Verbs: []string{"create", "delete", "deletecollection", "get", "list", "watch", "patch", "update"},
				},
				{
					APIGroups: []string{""},
					Resources: []string{
						"events",
						"resourcequotas",
						"serviceaccounts",
					},
					Verbs: []string{"get", "list", "watch"},
				},
				{
					APIGroups: []string{"events.k8s.io"},
					Resources: []string{"events"},
					Verbs:     []string{"get", "list", "watch"},
				},
				{
					APIGroups: []string{"core.gardener.cloud"},
					Resources: []string{
						"shoots",
						"secretbindings",
						"quotas",
					},
					Verbs: []string{"create", "delete", "deletecollection", "get", "list", "watch", "patch", "update"},
				},
				{
					APIGroups: []string{"settings.gardener.cloud"},
					Resources: []string{"openidconnectpresets"},
					Verbs:     []string{"create", "delete", "deletecollection", "get", "list", "watch", "patch", "update"},
				},
				{
					APIGroups: []string{"operations.gardener.cloud"},
					Resources: []string{"bastions"},
					Verbs:     []string{"create", "delete", "deletecollection", "get", "list", "watch", "patch", "update"},
				},
				{
					APIGroups: []string{"rbac.authorization.k8s.io"},
					Resources: []string{
						"roles",
						"rolebindings",
					},
					Verbs: []string{"create", "delete", "deletecollection", "get", "list", "watch", "patch", "update"},
				},
				{
					APIGroups: []string{"core.gardener.cloud"},
					Resources: []string{
						"shoots/adminkubeconfig",
						"shoots/viewerkubeconfig",
					},
					Verbs: []string{"create"},
				},
			},
		}
		clusterRoleProjectMemberAggregated = &rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "gardener.cloud:system:project-member",
				Labels: map[string]string{"gardener.cloud/role": "project-member"},
			},
			AggregationRule: &rbacv1.AggregationRule{
				ClusterRoleSelectors: []metav1.LabelSelector{
					{MatchLabels: map[string]string{"rbac.gardener.cloud/aggregate-to-project-member": "true"}},
				},
			},
		}
		clusterRoleProjectServiceAccountManager = &rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "gardener.cloud:system:project-serviceaccountmanager-aggregation",
				Labels: map[string]string{"rbac.gardener.cloud/aggregate-to-project-serviceaccountmanager": "true"},
			},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{"serviceaccounts"},
					Verbs:     []string{"create", "delete", "deletecollection", "get", "list", "watch", "patch", "update"},
				},
				{
					APIGroups: []string{""},
					Resources: []string{"serviceaccounts/token"},
					Verbs:     []string{"create"},
				},
			},
		}
		clusterRoleProjectServiceAccountManagerAggregated = &rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "gardener.cloud:system:project-serviceaccountmanager",
				Labels: map[string]string{"gardener.cloud/role": "project-serviceaccountmanager"},
			},
			AggregationRule: &rbacv1.AggregationRule{
				ClusterRoleSelectors: []metav1.LabelSelector{
					{MatchLabels: map[string]string{"rbac.gardener.cloud/aggregate-to-project-serviceaccountmanager": "true"}},
				},
			},
		}
		clusterRoleProjectViewer = &rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "gardener.cloud:system:project-viewer-aggregation",
				Labels: map[string]string{"rbac.gardener.cloud/aggregate-to-project-viewer": "true"},
			},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{
						"events",
						"configmaps",
						"resourcequotas",
						"serviceaccounts",
					},
					Verbs: []string{"get", "list", "watch"},
				},
				{
					APIGroups: []string{"events.k8s.io"},
					Resources: []string{"events"},
					Verbs:     []string{"get", "list", "watch"},
				},
				{
					APIGroups: []string{"core.gardener.cloud"},
					Resources: []string{
						"shoots",
						"secretbindings",
						"quotas",
					},
					Verbs: []string{"get", "list", "watch"},
				},
				{
					APIGroups: []string{"settings.gardener.cloud"},
					Resources: []string{"openidconnectpresets"},
					Verbs:     []string{"get", "list", "watch"},
				},
				{
					APIGroups: []string{"operations.gardener.cloud"},
					Resources: []string{"bastions"},
					Verbs:     []string{"get", "list", "watch"},
				},
				{
					APIGroups: []string{"rbac.authorization.k8s.io"},
					Resources: []string{
						"roles",
						"rolebindings",
					},
					Verbs: []string{"get", "list", "watch"},
				},
				{
					APIGroups: []string{"core.gardener.cloud"},
					Resources: []string{"shoots/viewerkubeconfig"},
					Verbs:     []string{"create"},
				},
			},
		}
		clusterRoleProjectViewerAggregated = &rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "gardener.cloud:system:project-viewer",
				Labels: map[string]string{"gardener.cloud/role": "project-viewer"},
			},
			AggregationRule: &rbacv1.AggregationRule{
				ClusterRoleSelectors: []metav1.LabelSelector{
					{MatchLabels: map[string]string{"rbac.gardener.cloud/aggregate-to-project-viewer": "true"}},
				},
			},
		}
		roleReadClusterIdentityConfigMap = &rbacv1.Role{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "gardener.cloud:system:read-cluster-identity-configmap",
				Namespace: "kube-system",
			},
			Rules: []rbacv1.PolicyRule{{
				APIGroups:     []string{""},
				Resources:     []string{"configmaps"},
				ResourceNames: []string{"cluster-identity"},
				Verbs:         []string{"get", "list", "watch"},
			}},
		}
		roleBindingReadClusterIdentityConfigMap = &rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "gardener.cloud:system:read-cluster-identity-configmap",
				Namespace: "kube-system",
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "Role",
				Name:     "gardener.cloud:system:read-cluster-identity-configmap",
			},
			Subjects: []rbacv1.Subject{{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "Group",
				Name:     "system:authenticated",
			}},
		}
	})

	Describe("#Deploy", func() {
		JustBeforeEach(func() {
			Expect(c.Get(ctx, client.ObjectKeyFromObject(managedResource), managedResource)).To(BeNotFoundError())

			Expect(component.Deploy(ctx)).To(Succeed())

			Expect(c.Get(ctx, client.ObjectKeyFromObject(managedResource), managedResource)).To(Succeed())
			expectedMr := &resourcesv1alpha1.ManagedResource{
				TypeMeta: metav1.TypeMeta{
					APIVersion: resourcesv1alpha1.SchemeGroupVersion.String(),
					Kind:       "ManagedResource",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:            managedResource.Name,
					Namespace:       managedResource.Namespace,
					ResourceVersion: "1",
					Labels:          map[string]string{"origin": "gardener"},
				},
				Spec: resourcesv1alpha1.ManagedResourceSpec{
					InjectLabels: map[string]string{"shoot.gardener.cloud/no-cleanup": "true"},
					SecretRefs:   []corev1.LocalObjectReference{{Name: managedResource.Spec.SecretRefs[0].Name}},
					KeepObjects:  pointer.Bool(false),
				},
			}
			utilruntime.Must(references.InjectAnnotations(expectedMr))
			Expect(managedResource).To(DeepEqual(expectedMr))

			managedResourceSecret.Name = managedResource.Spec.SecretRefs[0].Name
			Expect(c.Get(ctx, client.ObjectKeyFromObject(managedResourceSecret), managedResourceSecret)).To(Succeed())
			Expect(managedResourceSecret.Type).To(Equal(corev1.SecretTypeOpaque))
			Expect(managedResourceSecret.Immutable).To(Equal(pointer.Bool(true)))
			Expect(managedResourceSecret.Labels["resources.gardener.cloud/garbage-collectable-reference"]).To(Equal("true"))
		})

		It("should successfully deploy the resources when seed authorizer is disabled", func() {
			Expect(managedResourceSecret.Data).To(HaveLen(23))
			Expect(string(managedResourceSecret.Data["namespace____garden.yaml"])).To(Equal(componenttest.Serialize(namespaceGarden)))
			Expect(string(managedResourceSecret.Data["clusterrole____gardener.cloud_system_seed-bootstrapper.yaml"])).To(Equal(componenttest.Serialize(clusterRoleSeedBootstrapper)))
			Expect(string(managedResourceSecret.Data["clusterrolebinding____gardener.cloud_system_seed-bootstrapper.yaml"])).To(Equal(componenttest.Serialize(clusterRoleBindingSeedBootstrapper)))
			Expect(string(managedResourceSecret.Data["clusterrole____gardener.cloud_system_seeds.yaml"])).To(Equal(componenttest.Serialize(clusterRoleSeeds)))
			Expect(string(managedResourceSecret.Data["clusterrolebinding____gardener.cloud_system_seeds.yaml"])).To(Equal(componenttest.Serialize(clusterRoleBindingSeeds)))
			Expect(string(managedResourceSecret.Data["clusterrole____gardener.cloud_admin.yaml"])).To(Equal(componenttest.Serialize(clusterRoleGardenerAdmin)))
			Expect(string(managedResourceSecret.Data["clusterrolebinding____gardener.cloud_admin.yaml"])).To(Equal(componenttest.Serialize(clusterRoleBindingGardenerAdmin)))
			Expect(string(managedResourceSecret.Data["clusterrole____gardener.cloud_system_administrators.yaml"])).To(Equal(componenttest.Serialize(clusterRoleGardenerAdminAggregated)))
			Expect(string(managedResourceSecret.Data["clusterrole____gardener.cloud_viewer.yaml"])).To(Equal(componenttest.Serialize(clusterRoleGardenerViewer)))
			Expect(string(managedResourceSecret.Data["clusterrole____gardener.cloud_system_viewers.yaml"])).To(Equal(componenttest.Serialize(clusterRoleGardenerViewerAggregated)))
			Expect(string(managedResourceSecret.Data["clusterrole____gardener.cloud_system_read-global-resources.yaml"])).To(Equal(componenttest.Serialize(clusterRoleReadGlobalResources)))
			Expect(string(managedResourceSecret.Data["clusterrolebinding____gardener.cloud_system_read-global-resources.yaml"])).To(Equal(componenttest.Serialize(clusterRoleBindingReadGlobalResources)))
			Expect(string(managedResourceSecret.Data["clusterrole____gardener.cloud_system_user-auth.yaml"])).To(Equal(componenttest.Serialize(clusterRoleUserAuth)))
			Expect(string(managedResourceSecret.Data["clusterrolebinding____gardener.cloud_system_user-auth.yaml"])).To(Equal(componenttest.Serialize(clusterRoleBindingUserAuth)))
			Expect(string(managedResourceSecret.Data["clusterrole____gardener.cloud_system_project-creation.yaml"])).To(Equal(componenttest.Serialize(clusterRoleProjectCreation)))
			Expect(string(managedResourceSecret.Data["clusterrole____gardener.cloud_system_project-member.yaml"])).To(Equal(componenttest.Serialize(clusterRoleProjectMemberAggregated)))
			Expect(string(managedResourceSecret.Data["clusterrole____gardener.cloud_system_project-member-aggregation.yaml"])).To(Equal(componenttest.Serialize(clusterRoleProjectMember)))
			Expect(string(managedResourceSecret.Data["clusterrole____gardener.cloud_system_project-serviceaccountmanager.yaml"])).To(Equal(componenttest.Serialize(clusterRoleProjectServiceAccountManagerAggregated)))
			Expect(string(managedResourceSecret.Data["clusterrole____gardener.cloud_system_project-serviceaccountmanager-aggregation.yaml"])).To(Equal(componenttest.Serialize(clusterRoleProjectServiceAccountManager)))
			Expect(string(managedResourceSecret.Data["clusterrole____gardener.cloud_system_project-viewer.yaml"])).To(Equal(componenttest.Serialize(clusterRoleProjectViewerAggregated)))
			Expect(string(managedResourceSecret.Data["clusterrole____gardener.cloud_system_project-viewer-aggregation.yaml"])).To(Equal(componenttest.Serialize(clusterRoleProjectViewer)))
			Expect(string(managedResourceSecret.Data["role__kube-system__gardener.cloud_system_read-cluster-identity-configmap.yaml"])).To(Equal(componenttest.Serialize(roleReadClusterIdentityConfigMap)))
			Expect(string(managedResourceSecret.Data["rolebinding__kube-system__gardener.cloud_system_read-cluster-identity-configmap.yaml"])).To(Equal(componenttest.Serialize(roleBindingReadClusterIdentityConfigMap)))
		})

		Context("when seed authorizer is enabled", func() {
			BeforeEach(func() {
				values.SeedAuthorizerEnabled = true
				component = New(c, namespace, values)
			})

			It("should successfully deploy the resources when seed authorizer is enabled", func() {
				Expect(managedResourceSecret.Data).NotTo(HaveKey("clusterrole____gardener.cloud_system_seeds.yaml"))
				Expect(managedResourceSecret.Data).NotTo(HaveKey("clusterrolebinding____gardener.cloud_system_seeds.yaml"))
			})
		})
	})

	Describe("#Destroy", func() {
		It("should successfully destroy all resources", func() {
			Expect(c.Create(ctx, managedResource)).To(Succeed())
			Expect(c.Create(ctx, managedResourceSecret)).To(Succeed())

			Expect(c.Get(ctx, client.ObjectKeyFromObject(managedResource), managedResource)).To(Succeed())
			Expect(c.Get(ctx, client.ObjectKeyFromObject(managedResourceSecret), managedResourceSecret)).To(Succeed())

			Expect(component.Destroy(ctx)).To(Succeed())

			Expect(c.Get(ctx, client.ObjectKeyFromObject(managedResource), managedResource)).To(BeNotFoundError())
			Expect(c.Get(ctx, client.ObjectKeyFromObject(managedResourceSecret), managedResourceSecret)).To(BeNotFoundError())
		})
	})

	Context("waiting functions", func() {
		var fakeOps *retryfake.Ops

		BeforeEach(func() {
			fakeOps = &retryfake.Ops{MaxAttempts: 1}
			DeferCleanup(test.WithVars(
				&retry.Until, fakeOps.Until,
				&retry.UntilTimeout, fakeOps.UntilTimeout,
			))
		})

		Describe("#Wait", func() {
			It("should fail because reading the ManagedResource fails", func() {
				Expect(component.Wait(ctx)).To(MatchError(ContainSubstring("not found")))
			})

			It("should fail because the ManagedResource doesn't become healthy", func() {
				fakeOps.MaxAttempts = 2

				Expect(c.Create(ctx, &resourcesv1alpha1.ManagedResource{
					ObjectMeta: metav1.ObjectMeta{
						Name:       managedResourceName,
						Namespace:  namespace,
						Generation: 1,
					},
					Status: resourcesv1alpha1.ManagedResourceStatus{
						ObservedGeneration: 1,
						Conditions: []gardencorev1beta1.Condition{
							{
								Type:   resourcesv1alpha1.ResourcesApplied,
								Status: gardencorev1beta1.ConditionFalse,
							},
							{
								Type:   resourcesv1alpha1.ResourcesHealthy,
								Status: gardencorev1beta1.ConditionFalse,
							},
						},
					},
				})).To(Succeed())

				Expect(component.Wait(ctx)).To(MatchError(ContainSubstring("is not healthy")))
			})

			It("should successfully wait for the managed resource to become healthy", func() {
				fakeOps.MaxAttempts = 2

				Expect(c.Create(ctx, &resourcesv1alpha1.ManagedResource{
					ObjectMeta: metav1.ObjectMeta{
						Name:       managedResourceName,
						Namespace:  namespace,
						Generation: 1,
					},
					Status: resourcesv1alpha1.ManagedResourceStatus{
						ObservedGeneration: 1,
						Conditions: []gardencorev1beta1.Condition{
							{
								Type:   resourcesv1alpha1.ResourcesApplied,
								Status: gardencorev1beta1.ConditionTrue,
							},
							{
								Type:   resourcesv1alpha1.ResourcesHealthy,
								Status: gardencorev1beta1.ConditionTrue,
							},
						},
					},
				})).To(Succeed())

				Expect(component.Wait(ctx)).To(Succeed())
			})
		})

		Describe("#WaitCleanup", func() {
			It("should fail when the wait for the managed resource deletion times out", func() {
				fakeOps.MaxAttempts = 2

				Expect(c.Create(ctx, managedResource)).To(Succeed())

				Expect(component.WaitCleanup(ctx)).To(MatchError(ContainSubstring("still exists")))
			})

			It("should not return an error when it's already removed", func() {
				Expect(component.WaitCleanup(ctx)).To(Succeed())
			})
		})
	})
})
