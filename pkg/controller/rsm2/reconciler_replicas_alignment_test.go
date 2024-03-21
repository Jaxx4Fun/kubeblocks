/*
Copyright (C) 2022-2024 ApeCloud Co., Ltd

This file is part of KubeBlocks project

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <http://www.gnu.org/licenses/>.
*/

package rsm2

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	workloads "github.com/apecloud/kubeblocks/apis/workloads/v1alpha1"
	"github.com/apecloud/kubeblocks/pkg/controller/builder"
	"github.com/apecloud/kubeblocks/pkg/controller/kubebuilderx"
	"github.com/apecloud/kubeblocks/pkg/controller/model"
)

var _ = Describe("replicas alignment reconciler test", func() {
	BeforeEach(func() {
		rsm = builder.NewReplicatedStateMachineBuilder(namespace, name).
			SetService(&corev1.Service{}).
			SetReplicas(3).
			SetTemplate(template).
			SetVolumeClaimTemplates(volumeClaimTemplates...).
			SetRoles(roles).
			GetObject()
	})

	Context("PreCondition & Reconcile", func() {
		It("should work well", func() {
			By("PreCondition")
			rsm.Generation = 1
			tree := kubebuilderx.NewObjectTree()
			tree.SetRoot(rsm)
			reconciler = NewReplicasAlignmentReconciler()
			Expect(reconciler.PreCondition(tree)).Should(Equal(kubebuilderx.ResultSatisfied))

			By("prepare current tree")
			// desired: hello, foo-0, foo-1, bar-0, bar-1, bar-2, bar-3
			// current: foo-0, bar-1
			replicas := int32(7)
			rsm.Spec.Replicas = &replicas
			nameHello := "hello"
			instanceHello := workloads.InstanceTemplate{
				Name: &nameHello,
			}
			rsm.Spec.Instances = append(rsm.Spec.Instances, instanceHello)
			generateNameFoo := "foo"
			replicasFoo := int32(2)
			instanceFoo := workloads.InstanceTemplate{
				GenerateName: &generateNameFoo,
				Replicas:     &replicasFoo,
			}
			rsm.Spec.Instances = append(rsm.Spec.Instances, instanceFoo)
			podFoo0 := builder.NewPodBuilder(namespace, "foo-0").GetObject()
			podBar1 := builder.NewPodBuilder(namespace, "bar-1").GetObject()
			Expect(tree.Add(podFoo0, podBar1)).Should(Succeed())

			By("update revisions")
			revisionUpdateReconciler := NewRevisionUpdateReconciler()
			_, err := revisionUpdateReconciler.Reconcile(tree)
			Expect(err).Should(BeNil())

			By("do reconcile with OrderedReady(Serial) policy")
			orderedReadyTree, err := tree.DeepCopy()
			Expect(err).Should(BeNil())
			newTree, err := reconciler.Reconcile(orderedReadyTree)
			Expect(err).Should(BeNil())
			// desired: bar-0, bar-1, foo-0
			pods := newTree.List(&corev1.Pod{})
			Expect(pods).Should(HaveLen(3))
			currentPodSnapshot := make(model.ObjectSnapshot)
			for _, object := range pods {
				name, err := model.GetGVKName(object)
				Expect(err).Should(BeNil())
				currentPodSnapshot[*name] = object
			}
			podBar0 := builder.NewPodBuilder(namespace, "bar-0").GetObject()
			for _, object := range []client.Object{podFoo0, podBar0, podBar1} {
				name, err := model.GetGVKName(object)
				Expect(err).Should(BeNil())
				_, ok := currentPodSnapshot[*name]
				Expect(ok).Should(BeTrue())
			}

			By("do reconcile with Parallel policy")
			parallelTree, err := tree.DeepCopy()
			Expect(err).Should(BeNil())
			parallelRsm, ok := parallelTree.GetRoot().(*workloads.ReplicatedStateMachine)
			Expect(ok).Should(BeTrue())
			parallelRsm.Spec.PodManagementPolicy = appsv1.ParallelPodManagement
			newTree, err = reconciler.Reconcile(parallelTree)
			Expect(err).Should(BeNil())
			// desired: hello, foo-0, foo-1, bar-0, bar-1, bar-2, bar-3
			pods = newTree.List(&corev1.Pod{})
			Expect(pods).Should(HaveLen(7))
			currentPodSnapshot = make(model.ObjectSnapshot)
			for _, object := range pods {
				name, err := model.GetGVKName(object)
				Expect(err).Should(BeNil())
				currentPodSnapshot[*name] = object
			}

			podHello := builder.NewPodBuilder(namespace, "hello").GetObject()
			podFoo1 := builder.NewPodBuilder(namespace, "foo-1").GetObject()
			podBar2 := builder.NewPodBuilder(namespace, "bar-2").GetObject()
			podBar3 := builder.NewPodBuilder(namespace, "bar-3").GetObject()
			for _, object := range []client.Object{podHello, podFoo0, podFoo1, podBar0, podBar1, podBar2, podBar3} {
				name, err := model.GetGVKName(object)
				Expect(err).Should(BeNil())
				_, ok := currentPodSnapshot[*name]
				Expect(ok).Should(BeTrue())
			}
		})
	})
})
