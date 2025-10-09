/*
Copyright 2025 The Dapr Authors
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

package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/dapr/durabletask-go/workflow"
	"github.com/dapr/go-sdk/client"
	"github.com/dapr/kit/signals"
)

func main() {
	ctx := signals.Context()
	register(ctx)

	log.Println("Workflow worker started and ready to accept workflow requests")

	<-ctx.Done()
}

func register(ctx context.Context) {
	r := workflow.NewRegistry()

	workflows := []workflow.Workflow{
		WNoOp,
		WTimer,
		WActivity1,
		SimpleWorkflow,
		EventWorkflow,
		LongWorkflow,
		ChildWorkflow,
		ParentWorkflow,
		NestedParentWorkflow,
		RecursiveChildWorkflow,
		FanOutWorkflow,
		DataWorkflow,
	}
	activities := []workflow.Activity{
		ANoOP,
		SimpleActivity,
		LongRunningActivity,
		DataProcessingActivity,
	}

	for _, w := range workflows {
		if err := r.AddWorkflow(w); err != nil {
			log.Fatalf("error adding workflow %T: %v", w, err)
		}
	}

	for _, a := range activities {
		if err := r.AddActivity(a); err != nil {
			log.Fatalf("error adding activity %T: %v", a, err)
		}
	}

	wf, err := client.NewWorkflowClient()
	if err != nil {
		log.Fatal(err)
	}

	if err = wf.StartWorker(ctx, r); err != nil {
		log.Fatal(err)
	}
}

func WNoOp(ctx *workflow.WorkflowContext) (any, error) {
	return nil, nil
}

func WTimer(ctx *workflow.WorkflowContext) (any, error) {
	return nil, ctx.CreateTimer(time.Hour * 10).Await(nil)
}

func WActivity1(ctx *workflow.WorkflowContext) (any, error) {
	return nil, ctx.CallActivity(ANoOP).Await(nil)
}

func ANoOP(ctx workflow.ActivityContext) (any, error) {
	return nil, nil
}

func SimpleWorkflow(ctx *workflow.WorkflowContext) (any, error) {
	var input any
	ctx.GetInput(&input)

	var result string
	err := ctx.CallActivity(SimpleActivity, workflow.WithActivityInput(input)).Await(&result)
	if err != nil {
		return nil, fmt.Errorf("activity failed: %w", err)
	}

	ctx.CreateTimer(time.Second * 2).Await(nil)

	return map[string]interface{}{
		"status": "completed",
		"result": result,
	}, nil
}

func LongWorkflow(ctx *workflow.WorkflowContext) (any, error) {
	stages := []string{"initialization", "processing", "validation", "finalization"}
	results := make([]string, 0, len(stages))

	for _, stage := range stages {
		var stageResult string
		err := ctx.CallActivity(LongRunningActivity, workflow.WithActivityInput(stage)).Await(&stageResult)
		if err != nil {
			return nil, fmt.Errorf("stage %s failed: %w", stage, err)
		}
		results = append(results, stageResult)

		ctx.CreateTimer(time.Second * 1).Await(nil)
	}

	return map[string]interface{}{
		"status":  "completed",
		"stages":  stages,
		"results": results,
	}, nil
}

func EventWorkflow(ctx *workflow.WorkflowContext) (any, error) {
	return nil, ctx.WaitForExternalEvent("test-event", time.Hour).Await(nil)
}

func DataWorkflow(ctx *workflow.WorkflowContext) (any, error) {
	var input struct {
		Name  string                 `json:"name"`
		Value int                    `json:"value"`
		Data  map[string]interface{} `json:"data"`
	}

	if err := ctx.GetInput(&input); err != nil {
		return nil, fmt.Errorf("failed to get input: %w", err)
	}

	var processedData any
	err := ctx.CallActivity(DataProcessingActivity, workflow.WithActivityInput(input)).Await(&processedData)
	if err != nil {
		return nil, fmt.Errorf("data processing failed: %w", err)
	}

	output := map[string]interface{}{
		"originalName":  input.Name,
		"processedName": fmt.Sprintf("processed_%s", input.Name),
		"originalValue": input.Value,
		"doubledValue":  input.Value * 2,
		"processedData": processedData,
	}

	return output, nil
}

func SimpleActivity(ctx workflow.ActivityContext) (any, error) {
	var input map[string]interface{}
	if err := ctx.GetInput(&input); err != nil {
		input = make(map[string]interface{})
	}

	time.Sleep(time.Millisecond * 500)

	return fmt.Sprintf("Processed simple activity with input: %v", input), nil
}

func LongRunningActivity(ctx workflow.ActivityContext) (any, error) {
	var stage string
	if err := ctx.GetInput(&stage); err != nil {
		stage = "unknown"
	}

	time.Sleep(time.Second * 2)

	return fmt.Sprintf("Completed %s at %s", stage, time.Now().Format(time.RFC3339)), nil
}

func DataProcessingActivity(ctx workflow.ActivityContext) (any, error) {
	var input map[string]interface{}
	if err := ctx.GetInput(&input); err != nil {
		return nil, fmt.Errorf("failed to get input: %w", err)
	}

	processed := make(map[string]interface{})
	for k, v := range input {
		processed[fmt.Sprintf("processed_%s", k)] = v
	}

	processed["processedAt"] = time.Now().Format(time.RFC3339)

	return processed, nil
}

func ParentWorkflow(ctx *workflow.WorkflowContext) (any, error) {
	var input map[string]interface{}
	if err := ctx.GetInput(&input); err != nil {
		input = make(map[string]interface{})
	}

	childInput1 := map[string]interface{}{
		"parentID": ctx.ID(),
		"step":     1,
		"data":     input,
	}
	var childResult1 map[string]interface{}
	if err := ctx.CallChildWorkflow(ChildWorkflow, workflow.WithChildWorkflowInput(childInput1)).Await(&childResult1); err != nil {
		return nil, fmt.Errorf("first child workflow failed: %w", err)
	}

	childInput2 := map[string]interface{}{
		"parentID":     ctx.ID(),
		"step":         2,
		"previousData": childResult1,
	}
	var childResult2 map[string]interface{}
	if err := ctx.CallChildWorkflow(ChildWorkflow, workflow.WithChildWorkflowInput(childInput2)).Await(&childResult2); err != nil {
		return nil, fmt.Errorf("second child workflow failed: %w", err)
	}

	return map[string]interface{}{
		"status":       "completed",
		"parentID":     ctx.ID(),
		"childResult1": childResult1,
		"childResult2": childResult2,
	}, nil
}

func ChildWorkflow(ctx *workflow.WorkflowContext) (any, error) {
	var input map[string]interface{}
	if err := ctx.GetInput(&input); err != nil {
		return nil, fmt.Errorf("failed to get input: %w", err)
	}

	ctx.CreateTimer(time.Second).Await(nil)

	var activityResult string
	if err := ctx.CallActivity(SimpleActivity, workflow.WithActivityInput(input)).Await(&activityResult); err != nil {
		return nil, fmt.Errorf("child activity failed: %w", err)
	}

	return map[string]interface{}{
		"childID":        ctx.ID(),
		"parentID":       input["parentID"],
		"step":           input["step"],
		"processed":      true,
		"activityResult": activityResult,
	}, nil
}

func NestedParentWorkflow(ctx *workflow.WorkflowContext) (any, error) {

	nestedInput := map[string]interface{}{
		"level":    1,
		"maxLevel": 3,
		"rootID":   ctx.ID(),
	}

	var nestedResult map[string]interface{}
	if err := ctx.CallChildWorkflow(RecursiveChildWorkflow, workflow.WithChildWorkflowInput(nestedInput)).Await(&nestedResult); err != nil {
		return nil, fmt.Errorf("nested child workflow failed: %w", err)
	}

	return map[string]interface{}{
		"status":       "completed",
		"rootID":       ctx.ID(),
		"nestedResult": nestedResult,
	}, nil
}

func RecursiveChildWorkflow(ctx *workflow.WorkflowContext) (any, error) {
	var input struct {
		Level    int                    `json:"level"`
		MaxLevel int                    `json:"maxLevel"`
		RootID   string                 `json:"rootID"`
		Data     map[string]interface{} `json:"data"`
	}

	if err := ctx.GetInput(&input); err != nil {
		return nil, fmt.Errorf("failed to get input: %w", err)
	}

	result := map[string]interface{}{
		"instanceID": ctx.ID(),
		"level":      input.Level,
		"rootID":     input.RootID,
	}

	if input.Level < input.MaxLevel {
		childInput := map[string]interface{}{
			"level":    input.Level + 1,
			"maxLevel": input.MaxLevel,
			"rootID":   input.RootID,
			"data":     input.Data,
		}

		var childResult map[string]interface{}
		if err := ctx.CallChildWorkflow(RecursiveChildWorkflow, workflow.WithChildWorkflowInput(childInput)).Await(&childResult); err != nil {
			return nil, fmt.Errorf("recursive child at level %d failed: %w", input.Level+1, err)
		}
		result["childResult"] = childResult
	} else {
		var activityResult string
		if err := ctx.CallActivity(SimpleActivity, workflow.WithActivityInput(input.Data)).Await(&activityResult); err != nil {
			return nil, fmt.Errorf("activity at max level failed: %w", err)
		}
		result["finalActivity"] = activityResult
	}

	return result, nil
}

func FanOutWorkflow(ctx *workflow.WorkflowContext) (any, error) {
	var input struct {
		ParallelCount int                    `json:"parallelCount"`
		Data          map[string]interface{} `json:"data"`
	}

	input.ParallelCount = 3
	ctx.GetInput(&input)

	if input.ParallelCount <= 0 {
		input.ParallelCount = 3
	}
	if input.ParallelCount > 10 {
	}

	var childTasks []workflow.Task
	for i := 0; i < input.ParallelCount; i++ {
		childInput := map[string]interface{}{
			"parentID": ctx.ID(),
			"index":    i,
			"data":     input.Data,
		}
		task := ctx.CallChildWorkflow(ChildWorkflow, workflow.WithChildWorkflowInput(childInput))
		childTasks = append(childTasks, task)
	}

	results := make([]map[string]interface{}, 0, len(childTasks))
	for i, task := range childTasks {
		var result map[string]interface{}
		if err := task.Await(&result); err != nil {
			result = map[string]interface{}{
				"index": i,
				"error": err.Error(),
			}
		}
		results = append(results, result)
	}

	return map[string]interface{}{
		"status":        "completed",
		"parentID":      ctx.ID(),
		"parallelCount": input.ParallelCount,
		"results":       results,
	}, nil
}
