/*
 * Cadence-lint - The Cadence linter
 *
 * Copyright 2019-2022 Dapper Labs, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package analyzers

import (
	"fmt"
	"regexp"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/tools/analysis"
)

var ReferenceOperatorAnalyzer = (func() *analysis.Analyzer {

	elementFilter := []ast.Element{
		(*ast.ReferenceExpression)(nil),
	}

	invalidOperatorRegexp := regexp.MustCompile(`.*\bas[?!].*`)

	return &analysis.Analyzer{
		Description: "Detects invalid operators in reference expressions. These will get rejected in a future release.",
		Requires: []*analysis.Analyzer{
			analysis.InspectorAnalyzer,
		},
		Run: func(pass *analysis.Pass) interface{} {
			inspector := pass.ResultOf[analysis.InspectorAnalyzer].(*ast.Inspector)

			location := pass.Program.Location
			code := pass.Program.Code
			report := pass.Report

			inspector.Preorder(
				elementFilter,
				func(element ast.Element) {

					referenceExpression, ok := element.(*ast.ReferenceExpression)
					if !ok {
						return
					}

					// Operator is incorrectly parsed, but not stored in AST.
					// Analyze code instead

					startOffset := referenceExpression.Expression.EndPosition(nil).Offset + 1
					endOffset := referenceExpression.Type.StartPosition().Offset - 1

					if !invalidOperatorRegexp.MatchString(code[startOffset:endOffset]) {
						return
					}

					report(
						analysis.Diagnostic{
							Location: location,
							Range:    ast.NewRangeFromPositioned(nil, element),
							Category: UpdateCategory,
							Message:  "incorrect reference operator used",
							SecondaryMessage: fmt.Sprintf(
								"use the '%s' operator",
								ast.OperationCast.Symbol(),
							),
						},
					)
				},
			)

			return nil
		},
	}
})()

func init() {
	registerAnalyzer(
		"reference-operator",
		ReferenceOperatorAnalyzer,
	)
}
