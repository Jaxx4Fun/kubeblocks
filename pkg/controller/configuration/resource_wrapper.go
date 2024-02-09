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

package configuration

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	appsv1alpha1 "github.com/apecloud/kubeblocks/apis/apps/v1alpha1"
	cfgcore "github.com/apecloud/kubeblocks/pkg/configuration/core"
	"github.com/apecloud/kubeblocks/pkg/controllerutil"
)

type ResourceCtx struct {
	context.Context

	Err    error
	Client client.Client

	Namespace     string
	ClusterName   string
	ComponentName string
}

type ResourceFetcher[T any] struct {
	obj *T
	*ResourceCtx

	ClusterObj    *appsv1alpha1.Cluster
	ClusterDefObj *appsv1alpha1.ClusterDefinition
	ClusterVerObj *appsv1alpha1.ClusterVersion

	ConfigMapObj        *corev1.ConfigMap
	ConfigurationObj    *appsv1alpha1.Configuration
	ConfigConstraintObj *appsv1alpha1.ConfigConstraint

	ClusterComObj *appsv1alpha1.ClusterComponentSpec
}

func (r *ResourceFetcher[T]) Init(ctx *ResourceCtx, object *T) *T {
	r.obj = object
	r.ResourceCtx = ctx
	return r.obj
}

func (r *ResourceFetcher[T]) Wrap(fn func() error) (ret *T) {
	ret = r.obj
	if r.Err != nil {
		return
	}
	r.Err = fn()
	return
}

func (r *ResourceFetcher[T]) Cluster() *T {
	clusterKey := client.ObjectKey{
		Namespace: r.Namespace,
		Name:      r.ClusterName,
	}
	return r.Wrap(func() error {
		r.ClusterObj = &appsv1alpha1.Cluster{}
		return r.Client.Get(r.Context, clusterKey, r.ClusterObj)
	})
}

func (r *ResourceFetcher[T]) ClusterDef() *T {
	clusterDefKey := client.ObjectKey{
		Namespace: "",
		Name:      r.ClusterObj.Spec.ClusterDefRef,
	}
	return r.Wrap(func() error {
		r.ClusterDefObj = &appsv1alpha1.ClusterDefinition{}
		return r.Client.Get(r.Context, clusterDefKey, r.ClusterDefObj)
	})
}

func (r *ResourceFetcher[T]) ClusterVer() *T {
	clusterVerKey := client.ObjectKey{
		Namespace: "",
		Name:      r.ClusterObj.Spec.ClusterVersionRef,
	}
	return r.Wrap(func() error {
		if clusterVerKey.Name == "" {
			return nil
		}
		r.ClusterVerObj = &appsv1alpha1.ClusterVersion{}
		return r.Client.Get(r.Context, clusterVerKey, r.ClusterVerObj)
	})
}

func (r *ResourceFetcher[T]) ClusterComponent() *T {
	return r.Wrap(func() (err error) {
		r.ClusterComObj, err = controllerutil.GetOriginalOrGeneratedComponentSpecByName(r.Context, r.Client, r.ClusterObj, r.ComponentName)
		if err != nil {
			return err
		}
		return
	})
}

func (r *ResourceFetcher[T]) Configuration() *T {
	configKey := client.ObjectKey{
		Name:      cfgcore.GenerateComponentConfigurationName(r.ClusterName, r.ComponentName),
		Namespace: r.Namespace,
	}
	return r.Wrap(func() (err error) {
		configuration := appsv1alpha1.Configuration{}
		err = r.Client.Get(r.Context, configKey, &configuration)
		if err != nil {
			return client.IgnoreNotFound(err)
		}
		r.ConfigurationObj = &configuration
		return
	})
}

func (r *ResourceFetcher[T]) ConfigMap(configSpec string) *T {
	cmKey := client.ObjectKey{
		Name:      cfgcore.GetComponentCfgName(r.ClusterName, r.ComponentName, configSpec),
		Namespace: r.Namespace,
	}

	return r.Wrap(func() error {
		r.ConfigMapObj = &corev1.ConfigMap{}
		return r.Client.Get(r.Context, cmKey, r.ConfigMapObj)
	})
}

func (r *ResourceFetcher[T]) ConfigConstraints(ccName string) *T {
	return r.Wrap(func() error {
		if ccName != "" {
			r.ConfigConstraintObj = &appsv1alpha1.ConfigConstraint{}
			return r.Client.Get(r.Context, client.ObjectKey{Name: ccName}, r.ConfigConstraintObj)
		}
		return nil
	})
}

func (r *ResourceFetcher[T]) Complete() error {
	return r.Err
}

type Fetcher struct {
	ResourceFetcher[Fetcher]
}

func NewResourceFetcher(resourceCtx *ResourceCtx) *Fetcher {
	f := &Fetcher{}
	return f.Init(resourceCtx, f)
}