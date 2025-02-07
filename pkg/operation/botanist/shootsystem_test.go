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

package botanist_test

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"

	kubernetesfake "github.com/gardener/gardener/pkg/client/kubernetes/fake"
	mockshootsystem "github.com/gardener/gardener/pkg/component/shootsystem/mock"
	"github.com/gardener/gardener/pkg/operation"
	. "github.com/gardener/gardener/pkg/operation/botanist"
	shootpkg "github.com/gardener/gardener/pkg/operation/shoot"
)

var _ = Describe("ShootSystem", func() {
	var (
		ctrl     *gomock.Controller
		botanist *Botanist
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		botanist = &Botanist{Operation: &operation.Operation{}}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Describe("#DeployShootSystem", func() {
		var (
			shootSystem *mockshootsystem.MockInterface

			ctx     = context.TODO()
			fakeErr = fmt.Errorf("fake err")

			apiResourceList = []*metav1.APIResourceList{
				{
					GroupVersion: "foo/v1",
					APIResources: []metav1.APIResource{
						{Name: "bar", Verbs: metav1.Verbs{"create", "delete"}},
						{Name: "baz", Verbs: metav1.Verbs{"get", "list", "watch"}},
					},
				},
				{
					GroupVersion: "bar/v1beta1",
					APIResources: []metav1.APIResource{
						{Name: "foo", Verbs: metav1.Verbs{"get", "list", "watch"}},
						{Name: "baz", Verbs: metav1.Verbs{"get", "list", "watch"}},
					},
				},
			}
		)

		BeforeEach(func() {
			shootSystem = mockshootsystem.NewMockInterface(ctrl)

			fakeKubernetes := fake.NewSimpleClientset()
			fakeKubernetes.Fake = testing.Fake{Resources: apiResourceList}
			botanist.ShootClientSet = kubernetesfake.NewClientSetBuilder().WithKubernetes(fakeKubernetes).Build()

			botanist.Shoot = &shootpkg.Shoot{
				Components: &shootpkg.Components{
					SystemComponents: &shootpkg.SystemComponents{
						Resources: shootSystem,
					},
				},
			}

			shootSystem.EXPECT().SetAPIResourceList(apiResourceList)
		})

		It("should discover the API and deploy", func() {
			shootSystem.EXPECT().Deploy(ctx)
			Expect(botanist.DeployShootSystem(ctx)).To(Succeed())
		})

		It("should fail when the deploy function fails", func() {
			shootSystem.EXPECT().Deploy(ctx).Return(fakeErr)
			Expect(botanist.DeployShootSystem(ctx)).To(Equal(fakeErr))
		})
	})
})
