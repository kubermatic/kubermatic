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
	"errors"
	"fmt"
	"math/rand"
	"testing"

	"github.com/go-test/deep"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestBuildTaskGraph(t *testing.T) {
	tests := []struct {
		name         string
		graphBuilder *TasksGraphBuilder
		expLength    int
		expError     bool
	}{
		{
			name: "Linear sequence",
			graphBuilder: NewTasksGraphBuilder().
				AddTask("task1", nopTaskFn).
				AddTask("task2", nopTaskFn, "task1").
				AddTask("task3", nopTaskFn, "task2"),
			expLength: 3,
		},
		{
			name: "Multiple dependencies",
			graphBuilder: NewTasksGraphBuilder().
				AddTask("task4", nopTaskFn, "task1", "task2", "task3").
				AddTask("task1", nopTaskFn).
				AddTask("task2", nopTaskFn).
				AddTask("task3", nopTaskFn),
			expLength: 4,
		},
		{
			name: "Twisted",
			graphBuilder: NewTasksGraphBuilder().
				AddTask("task1", nopTaskFn).
				AddTask("task2", nopTaskFn, "task1", "task3").
				AddTask("task3", nopTaskFn, "task5").
				AddTask("task4", nopTaskFn, "task2").
				AddTask("task5", nopTaskFn, "task1"),
			expLength: 5,
		},
		{
			name: "Cycle",
			graphBuilder: NewTasksGraphBuilder().
				AddTask("task1", nopTaskFn, "task3").
				AddTask("task2", nopTaskFn, "task1").
				AddTask("task3", nopTaskFn, "task2"),
			expError: true,
		},
		{
			name: "Self cycle",
			graphBuilder: NewTasksGraphBuilder().
				AddTask("task1", nopTaskFn, "task1"),
			expError: true,
		},
		{
			name:         "Random DAG sparse",
			graphBuilder: generateRandomDAG(1000, 0.1),
			expError:     false,
			expLength:    1000,
		},
		{
			name:         "Random DAG tree",
			graphBuilder: generateRandomDAG(1000, 0.9),
			expError:     false,
			expLength:    1000,
		},
		{
			name:         "Random DAG tree complete",
			graphBuilder: generateRandomDAG(1000, 1.0),
			expError:     false,
			expLength:    1000,
		},
		{
			name:         "Random DAG tree no edges",
			graphBuilder: generateRandomDAG(1000, 0.0),
			expError:     false,
			expLength:    1000,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			g, err := test.graphBuilder.Build()
			t.Logf("Graph: %v", g)
			if (err != nil) != test.expError {
				t.Errorf("error expected=%t, but got: %v", test.expError, err)
			}
			// if an error was expected the other tests are not relevant
			if test.expError {
				return
			}
			if e, a := test.expLength, len(g); e != a {
				t.Errorf("expected %d nodes, but got: %d", e, a)
			}
			visited := map[*taskNode]struct{}{}
			for _, n := range g {
				if _, ok := visited[n]; ok {
					t.Errorf("nodes should not be duplicated: %v", n)
				}
				for _, d := range n.dependencies {
					if _, ok := visited[d]; !ok {
						t.Errorf("task node %s should be placed after their dependencies: %s", n, d)
					}
				}
				visited[n] = struct{}{}
			}
		})
	}
}

func TestRunTaskGraph(t *testing.T) {
	tests := []struct {
		name      string
		graph     TasksGraph
		expStatus RunStatus
	}{
		{
			name: "Base case",
			graph: buildTaskGraphForTest(t, NewTasksGraphBuilder().
				AddTask("task1", nopTaskFn).
				AddTask("task2", nopTaskFn, "task1").
				AddTask("task3", nopTaskFn, "task2")),
			expStatus: RunStatus{
				FailedTasks:  map[TaskID]error{},
				SkippedTasks: map[TaskID]struct{}{},
				SuccessTasks: map[TaskID]struct{}{
					"task1": {},
					"task2": {},
					"task3": {},
				},
			},
		},
		{
			name: "Skipped tasks",
			graph: buildTaskGraphForTest(t, NewTasksGraphBuilder().
				AddTask("task1", nopTaskFn).
				AddTask("task2", nopTaskFn, "task1").
				AddTask("task3", nopTaskFn, "task2").
				AddTask("task4", failedTaskFn(errors.New("task 4 failed")), "task3").
				AddTask("task5", nopTaskFn, "task4").
				AddTask("task6", nopTaskFn, "task5")),
			expStatus: RunStatus{
				FailedTasks: map[TaskID]error{
					"task4": errors.New("task 4 failed"),
				},
				SkippedTasks: map[TaskID]struct{}{
					"task5": {},
					"task6": {},
				},
				SuccessTasks: map[TaskID]struct{}{
					"task1": {},
					"task2": {},
					"task3": {},
				},
			},
		},
		{
			name: "No dependencies",
			graph: buildTaskGraphForTest(t, NewTasksGraphBuilder().
				AddTask("task1", nopTaskFn).
				AddTask("task2", nopTaskFn).
				AddTask("task3", nopTaskFn).
				AddTask("task4", failedTaskFn(errors.New("task 4 failed"))).
				AddTask("task5", nopTaskFn).
				AddTask("task6", failedTaskFn(errors.New("task 6 failed")))),
			expStatus: RunStatus{
				FailedTasks: map[TaskID]error{
					"task4": errors.New("task 4 failed"),
					"task6": errors.New("task 6 failed"),
				},
				SkippedTasks: map[TaskID]struct{}{},
				SuccessTasks: map[TaskID]struct{}{
					"task1": {},
					"task2": {},
					"task3": {},
					"task5": {},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rh := recordingHandler{}
			res := RunTasks(context.TODO(), fake.NewFakeClient(), test.graph, &rh)
			if diff := deep.Equal(res, test.expStatus); diff != nil {
				t.Errorf("unexpected run status: %v", diff)
			}
			var expRunOrder []TaskID
			// as tasks are run sequentially in the same gorouting so far we
			// can expect that the completion order and the start order match.
			// This login has to be changed if we parallelize tasks in future.
			for i := range test.graph {
				expRunOrder = append(expRunOrder, test.graph[i].id)
			}
			if diff := deep.Equal(rh.runOrder, expRunOrder); diff != nil {
				t.Errorf("unexpected completion order: %v", diff)
			}
		})
	}
}

type recordingHandler struct {
	runOrder []TaskID
}

func (h *recordingHandler) OnStarted(id TaskID) {
	//NOP
}
func (h *recordingHandler) OnSuccess(id TaskID) {
	h.runOrder = append(h.runOrder, id)
}
func (h *recordingHandler) OnSkipped(id TaskID) {
	h.runOrder = append(h.runOrder, id)
}
func (h *recordingHandler) OnError(id TaskID, _ error) {
	h.runOrder = append(h.runOrder, id)
}

func buildTaskGraphForTest(t *testing.T, b *TasksGraphBuilder) TasksGraph {
	tg, err := b.Build()
	if err != nil {
		t.Fatalf("error occurred while building the TasksGraph: %v", err)
		return TasksGraph{}
	}
	return tg
}

type dummyTask struct {
	called bool
	err    error
}

func (n *dummyTask) run(ctx context.Context, cli ctrlruntimeclient.Client) error {
	n.called = true
	return n.err
}

func failedTaskFn(err error) TaskFn {
	return (&dummyTask{err: err}).run
}

func nopTaskFn(ctx context.Context, cli ctrlruntimeclient.Client) error {
	return nil
}

func generateRandomDAG(numNodes int, edgeThreashold float64) *TasksGraphBuilder {
	var t = NewTasksGraphBuilder()
	var r = rand.New(rand.NewSource(123123))
	for i := 0; i < numNodes; i++ {
		var deps []TaskID
		for j := 0; j < i; j++ {
			if r.Float64() < edgeThreashold {
				deps = append(deps, TaskID(fmt.Sprintf("task-%d", j)))
			}
		}
		t.AddTask(TaskID(fmt.Sprintf("task-%d", i)), nopTaskFn, deps...)
	}
	return t
}
