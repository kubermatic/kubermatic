/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

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

package reconciling

import (
	"context"
	"fmt"
	"strings"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// TaskID represents the task identifier.
type TaskID string

// TaskIDSet is a set of TaskIDs
type TaskIDSet map[TaskID]struct{}

// Has returns true if and only if item is contained in the set.
func (s TaskIDSet) Has(item TaskID) bool {
	_, contained := s[item]
	return contained
}

// TaskFn represents a reconciliation task.
type TaskFn func(context.Context, ctrlruntimeclient.Client) error

// taskNode represents a node of the taskGraph.
type taskNode struct {
	id   TaskID
	task TaskFn
	// Only populated after graph is built.
	dependencies []*taskNode
}

func (tn *taskNode) String() string {
	var deps []string
	for _, d := range tn.dependencies {
		deps = append(deps, string(d.id))
	}
	return fmt.Sprintf("ID: %s out edges: %v", tn.id, deps)
}

// TasksGraph is a DAG where each node represent a task. Task nodes are sorted
// with topological ordering.
//
// This is useful to represent state-machines. e.g. migration tasks, updates.
type TasksGraph []*taskNode

// TasksGraphBuilder allows to create a TaskGraph.
// It is not thread-safe.
type TasksGraphBuilder struct {
	// tasksMap maps a task id with the corresponding taskNode.
	tasksMap map[TaskID]*taskNode
	// dependenciesMap keeps track of dependencies to be able to build the graph.
	// TODO(irozzo) Forbid forward references would make this unnecessary.
	dependenciesMap map[TaskID][]TaskID

	topologicalOrder []*taskNode
	mark             map[*taskNode]struct{}
	tempMark         map[*taskNode]struct{}
}

// TaskEventHandler can handle notifications for events that occur to a task.
type TaskEventHandler interface {
	// OnStarted is called when the task starts running.
	OnStarted(TaskID)
	// OnCompleted is called when the task completes.
	OnSuccess(TaskID)
	// OnSkipped is called when the task is skipped.
	OnSkipped(TaskID)
	// OnError is called when the task terminates with an error.
	OnError(TaskID, error)
}

// NewTasksGraphBuilder creates a new TaskRunner.
func NewTasksGraphBuilder() *TasksGraphBuilder {
	return &TasksGraphBuilder{
		tasksMap:        map[TaskID]*taskNode{},
		dependenciesMap: map[TaskID][]TaskID{},
	}
}

// AddTask adds a TaskFn to the TasksGraphBuilder with the given ID and
// optional dependencies.
func (tr *TasksGraphBuilder) AddTask(id TaskID, task TaskFn, dependencies ...TaskID) *TasksGraphBuilder {
	n := &taskNode{id: id, task: task}
	// TODO(irozzo): We should maybe check if node with such ID is already present?
	tr.tasksMap[id] = n
	tr.dependenciesMap[n.id] = dependencies
	return tr
}

// Build creates a nes TaskGraph based on the TaskFn added so far.
// It returns a non-nil error if some invariants are violated.
// * All dependencies shoud be present.
// * No cycles between tasks.
func (tr *TasksGraphBuilder) Build() (TasksGraph, error) {
	tr.mark = make(map[*taskNode]struct{})
	tr.tempMark = make(map[*taskNode]struct{})
	tr.topologicalOrder = make([]*taskNode, 0, len(tr.tasksMap))
	for _, n := range tr.tasksMap {
		if _, ok := tr.mark[n]; !ok {
			if err := tr.visitRec(n); err != nil {
				return nil, err
			}
		}
	}
	return TasksGraph(tr.topologicalOrder), nil
}

type cycleFoundError struct {
	cycle []*taskNode
}

func (e *cycleFoundError) addNode(n *taskNode) {
	e.cycle = append(e.cycle, n)
}

func (e *cycleFoundError) Error() string {
	var b strings.Builder
	b.WriteString("cycle was found in task graph: ")
	for i := len(e.cycle) - 1; i >= 0; i-- {
		b.WriteString(string(e.cycle[i].id))
		if i > 0 {
			b.WriteString(" -> ")
		}
	}
	return b.String()
}

func (tr *TasksGraphBuilder) visitRec(n *taskNode) error {
	if _, ok := tr.mark[n]; ok {
		return nil
	}
	if _, ok := tr.tempMark[n]; ok {
		return &cycleFoundError{[]*taskNode{n}}
	}

	// temporary mark the node.
	tr.tempMark[n] = struct{}{}

	n.dependencies = nil
	for _, c := range tr.dependenciesMap[n.id] {
		// re-initialize in case it is run multiple times.
		if t, ok := tr.tasksMap[c]; ok {
			n.dependencies = append(n.dependencies, t)
			if err := tr.visitRec(t); err != nil {
				// Construct the cycle path
				if cerr, ok := err.(*cycleFoundError); ok {
					cerr.addNode(n)
				}
				return err
			}
		} else {
			return fmt.Errorf("depenency of node %s was not found: %s", n.id, c)
		}
	}

	delete(tr.tempMark, n)
	tr.mark[n] = struct{}{}
	tr.topologicalOrder = append(tr.topologicalOrder, n)
	return nil
}

// RunTasks executes the provided task graph.
func RunTasks(ctx context.Context, client ctrlruntimeclient.Client, tasks TasksGraph, handlers ...TaskEventHandler) RunStatus {
	runner := tasksRunner{ctx: ctx, client: client, eventHandlers: handlers}
	return runner.run(tasks)
}

// RunStatus contains the results of the TaskGraph execution.
type RunStatus struct {
	SuccessTasks TaskIDSet
	SkippedTasks TaskIDSet
	FailedTasks  map[TaskID]error
}

// Error returns the error returned by the task execution or nil if it did not
// return an error.
func (s RunStatus) Error(id TaskID) error {
	return s.FailedTasks[id]
}

// Failed returns the IDs of the tasks that failed during the execution.
func (s RunStatus) Failed() []TaskID {
	ids := make([]TaskID, len(s.FailedTasks), 0)
	for id := range s.FailedTasks {
		ids = append(ids, id)
	}
	return ids
}

// tasksRunner runs the tasks of the given TaskGraph.
type tasksRunner struct {
	RunStatus
	ctx           context.Context
	client        ctrlruntimeclient.Client
	eventHandlers []TaskEventHandler
}

func (r *tasksRunner) run(g TasksGraph) RunStatus {
	r.SuccessTasks = TaskIDSet{}
	r.SkippedTasks = TaskIDSet{}
	r.FailedTasks = map[TaskID]error{}

OUTER:
	for _, t := range g {
		for _, d := range t.dependencies {
			// Skip the task if any of the dependencies was not run or skipped.
			if r.Error(d.id) != nil || r.SkippedTasks.Has(d.id) {
				for _, h := range r.eventHandlers {
					h.OnSkipped(t.id)
				}
				r.SkippedTasks[t.id] = struct{}{}
				continue OUTER
			}
			if _, ok := r.SuccessTasks[d.id]; !ok {
				// TODO(irozzo): We assume that task nodes in TaskGraph are sorted
				// by topological ordering. Verify before running.
				panic(fmt.Sprintf("task %s cannot be run as it depends on task %s that did not complete", t.id, d.id))
			}
		}
		for _, h := range r.eventHandlers {
			h.OnStarted(t.id)
		}
		if err := t.task(r.ctx, r.client); err != nil {
			for _, h := range r.eventHandlers {
				h.OnError(t.id, err)
			}
			r.FailedTasks[t.id] = err
		} else {
			for _, h := range r.eventHandlers {
				h.OnSuccess(t.id)
			}
			r.SuccessTasks[t.id] = struct{}{}
		}
	}

	return r.RunStatus
}
