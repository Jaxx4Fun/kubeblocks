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

package kubebuilderx

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	workloads "github.com/apecloud/kubeblocks/apis/workloads/v1alpha1"
	"github.com/apecloud/kubeblocks/pkg/controller/graph"
	"github.com/apecloud/kubeblocks/pkg/controller/model"
	rsm1 "github.com/apecloud/kubeblocks/pkg/controller/rsm"
)

type transformContext struct {
	ctx      context.Context
	cli      client.Reader
	recorder record.EventRecorder
	logger   logr.Logger
}

type PlanBuilder struct {
	transCtx     *transformContext
	cli          client.Client
	transformers graph.TransformerChain
}

type Plan struct {
	dag      *graph.DAG
	walkFunc graph.WalkFunc
	transCtx *transformContext
}

var _ graph.TransformContext = &transformContext{}
var _ graph.PlanBuilder = &PlanBuilder{}
var _ graph.Plan = &Plan{}

func init() {
	model.AddScheme(workloads.AddToScheme)
}

func (t *transformContext) GetContext() context.Context {
	return t.ctx
}

func (t *transformContext) GetClient() client.Reader {
	return t.cli
}

func (t *transformContext) GetRecorder() record.EventRecorder {
	return t.recorder
}

func (t *transformContext) GetLogger() logr.Logger {
	return t.logger
}

// PlanBuilder implementation

func (b *PlanBuilder) Init() error {
	return nil
}

func (b *PlanBuilder) AddTransformer(transformer ...graph.Transformer) graph.PlanBuilder {
	b.transformers = append(b.transformers, transformer...)
	return b
}

func (b *PlanBuilder) AddParallelTransformer(transformer ...graph.Transformer) graph.PlanBuilder {
	b.transformers = append(b.transformers, &model.ParallelTransformer{Transformers: transformer})
	return b
}

func (b *PlanBuilder) Build() (graph.Plan, error) {
	var err error
	// new a DAG and apply chain on it, after that we should get the final Plan
	dag := graph.NewDAG()
	err = b.transformers.ApplyTo(b.transCtx, dag)
	// log for debug
	b.transCtx.logger.Info(fmt.Sprintf("DAG: %s", dag))

	// we got the execution plan
	plan := &Plan{
		dag:      dag,
		walkFunc: b.rsmWalkFunc,
		transCtx: b.transCtx,
	}
	return plan, err
}

// Plan implementation

func (p *Plan) Execute() error {
	return p.dag.WalkReverseTopoOrder(p.walkFunc, nil)
}

// Do the real works

func (b *PlanBuilder) rsmWalkFunc(v graph.Vertex) error {
	vertex, ok := v.(*model.ObjectVertex)
	if !ok {
		return fmt.Errorf("wrong vertex type %v", v)
	}
	if vertex.Action == nil {
		return errors.New("vertex action can't be nil")
	}
	ctx := b.transCtx.ctx
	switch *vertex.Action {
	case model.CREATE:
		return b.createObject(ctx, vertex)
	case model.UPDATE:
		return b.updateObject(ctx, vertex)
	case model.PATCH:
		return b.patchObject(ctx, vertex)
	case model.DELETE:
		return b.deleteObject(ctx, vertex)
	case model.STATUS:
		return b.statusObject(ctx, vertex)
	}
	return nil
}

func (b *PlanBuilder) createObject(ctx context.Context, vertex *model.ObjectVertex) error {
	err := b.cli.Create(ctx, vertex.Obj, rsm1.ClientOption(vertex))
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

func (b *PlanBuilder) updateObject(ctx context.Context, vertex *model.ObjectVertex) error {
	err := b.cli.Update(ctx, vertex.Obj, rsm1.ClientOption(vertex))
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	return nil
}

func (b *PlanBuilder) patchObject(ctx context.Context, vertex *model.ObjectVertex) error {
	patch := client.MergeFrom(vertex.OriObj)
	err := b.cli.Patch(ctx, vertex.Obj, patch, rsm1.ClientOption(vertex))
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	return nil
}

func (b *PlanBuilder) deleteObject(ctx context.Context, vertex *model.ObjectVertex) error {
	finalizer := rsm1.GetFinalizer(vertex.Obj)
	if controllerutil.RemoveFinalizer(vertex.Obj, finalizer) {
		err := b.cli.Update(ctx, vertex.Obj, rsm1.ClientOption(vertex))
		if err != nil && !apierrors.IsNotFound(err) {
			b.transCtx.logger.Error(err, fmt.Sprintf("delete %T error: %s", vertex.Obj, vertex.Obj.GetName()))
			return err
		}
	}
	if !model.IsObjectDeleting(vertex.Obj) {
		err := b.cli.Delete(ctx, vertex.Obj, rsm1.ClientOption(vertex))
		if err != nil && !apierrors.IsNotFound(err) {
			return err
		}
	}
	return nil
}

func (b *PlanBuilder) statusObject(ctx context.Context, vertex *model.ObjectVertex) error {
	if err := b.cli.Status().Update(ctx, vertex.Obj, rsm1.ClientOption(vertex)); err != nil {
		return err
	}
	return nil
}

// NewPlanBuilder returns a PlanBuilder
func NewPlanBuilder(ctx context.Context, cli client.Client, recorder record.EventRecorder, logger logr.Logger) graph.PlanBuilder {
	return &PlanBuilder{
		transCtx: &transformContext{
			ctx:      ctx,
			cli:      model.NewGraphClient(cli),
			recorder: recorder,
			logger:   logger,
		},
		cli: cli,
	}
}
