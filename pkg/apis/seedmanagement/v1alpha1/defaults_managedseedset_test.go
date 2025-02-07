// Copyright 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package v1alpha1_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/utils/pointer"

	. "github.com/gardener/gardener/pkg/apis/seedmanagement/v1alpha1"
)

var _ = Describe("Defaults", func() {
	var obj *ManagedSeedSet

	BeforeEach(func() {
		obj = &ManagedSeedSet{}
	})

	Describe("ManagedSeedSet defaulting", func() {
		It("should default replicas to 1 and revisionHistoryLimit to 10", func() {
			SetObjectDefaults_ManagedSeedSet(obj)

			Expect(obj.Spec.Replicas).To(Equal(pointer.Int32(1)))
			Expect(obj.Spec.UpdateStrategy).NotTo(BeNil())
			Expect(obj.Spec.RevisionHistoryLimit).To(Equal(pointer.Int32(10)))
		})

		It("should not overwrite the already set values for ManagedSeedSet spec", func() {
			obj.Spec = ManagedSeedSetSpec{
				Replicas:             pointer.Int32(5),
				RevisionHistoryLimit: pointer.Int32(15),
			}
			SetObjectDefaults_ManagedSeedSet(obj)

			Expect(obj.Spec.Replicas).To(Equal(pointer.Int32(5)))
			Expect(obj.Spec.UpdateStrategy).NotTo(BeNil())
			Expect(obj.Spec.RevisionHistoryLimit).To(Equal(pointer.Int32(15)))
		})
	})

	Describe("UpdateStrategy defaulting", func() {
		It("should default type to RollingUpdate", func() {
			obj.Spec.UpdateStrategy = &UpdateStrategy{}
			SetObjectDefaults_ManagedSeedSet(obj)

			Expect(obj.Spec.UpdateStrategy).To(Equal(&UpdateStrategy{
				Type: updateStrategyTypePtr(RollingUpdateStrategyType),
			}))
		})

		It("should not overwrite already set values for UpdateStrategy", func() {
			obj.Spec.UpdateStrategy = &UpdateStrategy{
				Type: updateStrategyTypePtr(UpdateStrategyType("foo")),
			}
			SetObjectDefaults_ManagedSeedSet(obj)

			Expect(obj.Spec.UpdateStrategy).To(Equal(&UpdateStrategy{
				Type: updateStrategyTypePtr(UpdateStrategyType("foo")),
			}))
		})
	})

	Describe("RollingUpdateStrategy defaulting", func() {
		It("should default partition to 0", func() {
			obj.Spec.UpdateStrategy = &UpdateStrategy{
				RollingUpdate: &RollingUpdateStrategy{},
			}
			SetObjectDefaults_ManagedSeedSet(obj)

			Expect(obj.Spec.UpdateStrategy.RollingUpdate).To(Equal(&RollingUpdateStrategy{
				Partition: pointer.Int32(0),
			}))
		})

		It("should not overwrote the already set values for RollingUpdateStrategy", func() {
			obj.Spec.UpdateStrategy = &UpdateStrategy{
				RollingUpdate: &RollingUpdateStrategy{
					Partition: pointer.Int32(1),
				},
			}
			SetObjectDefaults_ManagedSeedSet(obj)

			Expect(obj.Spec.UpdateStrategy.RollingUpdate).To(Equal(&RollingUpdateStrategy{
				Partition: pointer.Int32(1),
			}))
		})
	})
})

func updateStrategyTypePtr(v UpdateStrategyType) *UpdateStrategyType {
	return &v
}
