/*
Copyright 2023 The Tekton Authors

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

package resultref

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

const (
	resultExpressionFormat     = "tasks.<taskName>.results.<resultName>"
	stepResultExpressionFormat = "steps.<stepName>.results.<resultName>"
	// Result expressions of the form <resultName>.<attribute> will be treated as object results.
	// If a string result name contains a dot, brackets should be used to differentiate it from an object result.
	// https://github.com/tektoncd/community/blob/main/teps/0075-object-param-and-result-types.md#collisions-with-builtin-variable-replacement
	objectResultExpressionFormat     = "tasks.<taskName>.results.<objectResultName>.<individualAttribute>"
	objectStepResultExpressionFormat = "steps.<stepName>.results.<objectResultName>.<individualAttribute>"
	// ResultStepPart Constant used to define the "steps" part of a step result reference
	ResultStepPart = "steps"
	// ResultTaskPart Constant used to define the "tasks" part of a pipeline result reference
	ResultTaskPart = "tasks"
	// ResultFinallyPart Constant used to define the "finally" part of a pipeline result reference
	ResultFinallyPart = "finally"
	// ResultResultPart Constant used to define the "results" part of a pipeline result reference
	ResultResultPart = "results"

	// arrayIndexing will match all `[int]` and `[*]` for parseExpression
	arrayIndexing          = `\[([0-9])*\*?\]`
	stepResultUsagePattern = `\$\(steps\..*\.results\..*\)`
)

// arrayIndexingRegex is used to match `[int]` and `[*]`
var arrayIndexingRegex = regexp.MustCompile(arrayIndexing)

// StepResultRegex compiles the regex pattern for the usage of step results.
var StepResultRegex = regexp.MustCompile(stepResultUsagePattern)

// LooksLikeResultRef attempts to check if the given string looks like it contains any
// result references. Returns true if it does, false otherwise
func LooksLikeResultRef(expression string) bool {
	subExpressions := strings.Split(expression, ".")
	return len(subExpressions) >= 4 && (subExpressions[0] == ResultTaskPart || subExpressions[0] == ResultFinallyPart) && subExpressions[2] == ResultResultPart
}

// looksLikeStepResultRef attempts to check if the given string looks like it contains any
// step result references. Returns true if it does, false otherwise
func looksLikeStepResultRef(expression string) bool {
	subExpressions := strings.Split(expression, ".")
	return len(subExpressions) >= 4 && subExpressions[0] == ResultStepPart && subExpressions[2] == ResultResultPart
}

// ParsedResult captures the task/step name, result name, type,
// array idx (in case of array result) and
// object key (in case of an object result).
// This is generated by parsing expressions that use
// $(tasks.taskName.results.resultName...) or $(steps.stepName.results.resultName...)
type ParsedResult struct {
	ResourceName string
	ResultName   string
	ResultType   string
	ArrayIdx     *int
	ObjectKey    string
}

// parseExpression parses "task name", "result name", "array index" (iff it's an array result) and "object key name" (iff it's an object result)
// 1. Reference string result
// - Input: tasks.myTask.results.aStringResult
// - Output: "myTask", "aStringResult", nil, "", nil
// 2. Reference Object value with key:
// - Input: tasks.myTask.results.anObjectResult.key1
// - Output: "myTask", "anObjectResult", nil, "key1", nil
// 3. Reference array elements with array indexing :
// - Input: tasks.myTask.results.anArrayResult[1]
// - Output: "myTask", "anArrayResult", 1, "", nil
// 4. Referencing whole array or object result:
// - Input: tasks.myTask.results.Result[*]
// - Output: "myTask", "Result", nil, "", nil
// Invalid Case:
// - Input: tasks.myTask.results.resultName.foo.bar
// - Output: "", "", nil, "", error
// TODO: may use regex for each type to handle possible reference formats
func parseExpression(substitutionExpression string) (ParsedResult, error) {
	if LooksLikeResultRef(substitutionExpression) || looksLikeStepResultRef(substitutionExpression) {
		subExpressions := strings.Split(substitutionExpression, ".")
		// For string result: tasks.<taskName>.results.<stringResultName>
		// For string step result: steps.<stepName>.results.<stringResultName>
		// For array result: tasks.<taskName>.results.<arrayResultName>[index]
		// For array step result: steps.<stepName>.results.<arrayResultName>[index]
		if len(subExpressions) == 4 {
			resultName, stringIdx := ParseResultName(subExpressions[3])
			if stringIdx != "" {
				if stringIdx == "*" {
					pr := ParsedResult{
						ResourceName: subExpressions[1],
						ResultName:   resultName,
						ResultType:   "array",
					}
					return pr, nil
				}
				intIdx, _ := strconv.Atoi(stringIdx)
				pr := ParsedResult{
					ResourceName: subExpressions[1],
					ResultName:   resultName,
					ResultType:   "array",
					ArrayIdx:     &intIdx,
				}
				return pr, nil
			}
			pr := ParsedResult{
				ResourceName: subExpressions[1],
				ResultName:   resultName,
				ResultType:   "string",
			}
			return pr, nil
		} else if len(subExpressions) == 5 {
			// For object type result: tasks.<taskName>.results.<objectResultName>.<individualAttribute>
			// For object type step result: steps.<stepName>.results.<objectResultName>.<individualAttribute>
			pr := ParsedResult{
				ResourceName: subExpressions[1],
				ResultName:   subExpressions[3],
				ResultType:   "object",
				ObjectKey:    subExpressions[4],
			}
			return pr, nil
		}
	}
	return ParsedResult{}, fmt.Errorf("must be one of the form 1). %q; 2). %q; 3). %q; 4). %q", resultExpressionFormat, objectResultExpressionFormat, stepResultExpressionFormat, objectStepResultExpressionFormat)
}

// ParseTaskExpression parses the input string and searches for the use of task result usage.
func ParseTaskExpression(substitutionExpression string) (ParsedResult, error) {
	if LooksLikeResultRef(substitutionExpression) {
		return parseExpression(substitutionExpression)
	}
	return ParsedResult{}, fmt.Errorf("must be one of the form 1). %q; 2). %q", resultExpressionFormat, objectResultExpressionFormat)
}

// ParseStepExpression parses the input string and searches for the use of step result usage.
func ParseStepExpression(substitutionExpression string) (ParsedResult, error) {
	if looksLikeStepResultRef(substitutionExpression) {
		return parseExpression(substitutionExpression)
	}
	return ParsedResult{}, fmt.Errorf("must be one of the form 1). %q; 2). %q", stepResultExpressionFormat, objectStepResultExpressionFormat)
}

// ParseResultName parse the input string to extract resultName and result index.
// Array indexing:
// Input:  anArrayResult[1]
// Output: anArrayResult, "1"
// Array star reference:
// Input:  anArrayResult[*]
// Output: anArrayResult, "*"
func ParseResultName(resultName string) (string, string) {
	stringIdx := strings.TrimSuffix(strings.TrimPrefix(arrayIndexingRegex.FindString(resultName), "["), "]")
	resultName = arrayIndexingRegex.ReplaceAllString(resultName, "")
	return resultName, stringIdx
}
