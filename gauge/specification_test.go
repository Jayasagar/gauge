package main

import . "launchpad.net/gocheck"

func (s *MySuite) TestThrowsErrorForMultipleSpecHeading(c *C) {
	tokens := []*token{
		&token{kind: specKind, value: "Spec Heading", lineNo: 1},
		&token{kind: scenarioKind, value: "Scenario Heading", lineNo: 2},
		&token{kind: stepKind, value: "Example step", lineNo: 3},
		&token{kind: specKind, value: "Another Heading", lineNo: 4},
	}

	_, result := new(specParser).createSpecification(tokens, new(conceptDictionary))

	c.Assert(result.ok, Equals, false)

	c.Assert(result.error.message, Equals, "Parse error: Multiple spec headings found in same file")
	c.Assert(result.error.lineNo, Equals, 4)
}

func (s *MySuite) TestThrowsErrorForScenarioWithoutSpecHeading(c *C) {
	tokens := []*token{
		&token{kind: scenarioKind, value: "Scenario Heading", lineNo: 1},
		&token{kind: stepKind, value: "Example step", lineNo: 2},
	}

	_, result := new(specParser).createSpecification(tokens, new(conceptDictionary))

	c.Assert(result.ok, Equals, false)

	c.Assert(result.error.message, Equals, "Parse error: Scenario should be defined after the spec heading")
	c.Assert(result.error.lineNo, Equals, 1)
}

func (s *MySuite) TestThrowsErrorForDuplicateScenariosWithinTheSameSpec(c *C) {
	tokens := []*token{
		&token{kind: specKind, value: "Spec Heading", lineNo: 1},
		&token{kind: scenarioKind, value: "Scenario Heading", lineNo: 2},
		&token{kind: stepKind, value: "Example step", lineNo: 3},
		&token{kind: scenarioKind, value: "Scenario Heading", lineNo: 4},
	}

	_, result := new(specParser).createSpecification(tokens, new(conceptDictionary))

	c.Assert(result.ok, Equals, false)

	c.Assert(result.error.message, Equals, "Parse error: Duplicate scenario definitions are not allowed in the same specification")
	c.Assert(result.error.lineNo, Equals, 4)
}

func (s *MySuite) TestSpecWithHeadingAndSimpleSteps(c *C) {
	tokens := []*token{
		&token{kind: specKind, value: "Spec Heading", lineNo: 1},
		&token{kind: scenarioKind, value: "Scenario Heading", lineNo: 2},
		&token{kind: stepKind, value: "Example step", lineNo: 3},
	}

	spec, result := new(specParser).createSpecification(tokens, new(conceptDictionary))

	c.Assert(len(spec.items), Equals, 1)
	c.Assert(spec.items[0], Equals, spec.scenarios[0])
	scenarioItems := (spec.items[0]).(*scenario).items
	c.Assert(scenarioItems[0], Equals, spec.scenarios[0].steps[0])

	c.Assert(result.ok, Equals, true)
	c.Assert(spec.heading.lineNo, Equals, 1)
	c.Assert(spec.heading.value, Equals, "Spec Heading")

	c.Assert(len(spec.scenarios), Equals, 1)
	c.Assert(spec.scenarios[0].heading.lineNo, Equals, 2)
	c.Assert(spec.scenarios[0].heading.value, Equals, "Scenario Heading")
	c.Assert(len(spec.scenarios[0].steps), Equals, 1)
	c.Assert(spec.scenarios[0].steps[0].value, Equals, "Example step")
}

func (s *MySuite) TestStepsAndComments(c *C) {
	tokens := []*token{
		&token{kind: specKind, value: "Spec Heading", lineNo: 1},
		&token{kind: commentKind, value: "A comment with some text and **bold** characters", lineNo: 2},
		&token{kind: scenarioKind, value: "Scenario Heading", lineNo: 3},
		&token{kind: commentKind, value: "Another comment", lineNo: 4},
		&token{kind: stepKind, value: "Example step", lineNo: 5},
		&token{kind: commentKind, value: "Third comment", lineNo: 6},
	}

	spec, result := new(specParser).createSpecification(tokens, new(conceptDictionary))
	c.Assert(len(spec.items), Equals, 2)
	c.Assert(spec.items[0], Equals, spec.comments[0])
	c.Assert(spec.items[1], Equals, spec.scenarios[0])

	scenarioItems := (spec.items[1]).(*scenario).items
	c.Assert(3, Equals, len(scenarioItems))
	c.Assert(scenarioItems[0], Equals, spec.scenarios[0].comments[0])
	c.Assert(scenarioItems[1], Equals, spec.scenarios[0].steps[0])
	c.Assert(scenarioItems[2], Equals, spec.scenarios[0].comments[1])

	c.Assert(result.ok, Equals, true)
	c.Assert(spec.heading.value, Equals, "Spec Heading")

	c.Assert(len(spec.comments), Equals, 1)
	c.Assert(spec.comments[0].lineNo, Equals, 2)
	c.Assert(spec.comments[0].value, Equals, "A comment with some text and **bold** characters")

	c.Assert(len(spec.scenarios), Equals, 1)
	scenario := spec.scenarios[0]

	c.Assert(2, Equals, len(scenario.comments))
	c.Assert(scenario.comments[0].lineNo, Equals, 4)
	c.Assert(scenario.comments[0].value, Equals, "Another comment")

	c.Assert(scenario.comments[1].lineNo, Equals, 6)
	c.Assert(scenario.comments[1].value, Equals, "Third comment")

	c.Assert(scenario.heading.value, Equals, "Scenario Heading")
	c.Assert(len(scenario.steps), Equals, 1)
}

func (s *MySuite) TestStepsWithParam(c *C) {
	tokens := []*token{
		&token{kind: specKind, value: "Spec Heading", lineNo: 1},
		&token{kind: tableHeader, args: []string{"id"}, lineNo: 2},
		&token{kind: tableRow, args: []string{"1"}, lineNo: 3},
		&token{kind: scenarioKind, value: "Scenario Heading", lineNo: 4},
		&token{kind: stepKind, value: "enter {static} with {dynamic}", lineNo: 5, args: []string{"user", "id"}},
		&token{kind: stepKind, value: "sample \\{static\\}", lineNo: 6},
	}

	spec, result := new(specParser).createSpecification(tokens, new(conceptDictionary))
	c.Assert(result.ok, Equals, true)
	step := spec.scenarios[0].steps[0]
	c.Assert(step.value, Equals, "enter {} with {}")
	c.Assert(step.lineNo, Equals, 5)
	c.Assert(len(step.args), Equals, 2)
	c.Assert(step.args[0].value, Equals, "user")
	c.Assert(step.args[0].argType, Equals, static)
	c.Assert(step.args[1].value, Equals, "id")
	c.Assert(step.args[1].argType, Equals, dynamic)
	c.Assert(step.args[1].name, Equals, "id")

	escapedStep := spec.scenarios[0].steps[1]
	c.Assert(escapedStep.value, Equals, "sample \\{static\\}")
	c.Assert(len(escapedStep.args), Equals, 0)
}

func (s *MySuite) TestStepsWithKeywords(c *C) {
	tokens := []*token{
		&token{kind: specKind, value: "Spec Heading", lineNo: 1},
		&token{kind: scenarioKind, value: "Scenario Heading", lineNo: 2},
		&token{kind: stepKind, value: "sample {static} and {dynamic}", lineNo: 3, args: []string{"name"}},
	}

	_, result := new(specParser).createSpecification(tokens, new(conceptDictionary))

	c.Assert(result, NotNil)
	c.Assert(result.ok, Equals, false)
	c.Assert(result.error.message, Equals, "Step text should not have '{static}' or '{dynamic}' or '{special}'")
	c.Assert(result.error.lineNo, Equals, 3)
}

func (s *MySuite) TestContextWithKeywords(c *C) {
	tokens := []*token{
		&token{kind: specKind, value: "Spec Heading", lineNo: 1},
		&token{kind: stepKind, value: "sample {static} and {dynamic}", lineNo: 3, args: []string{"name"}},
		&token{kind: scenarioKind, value: "Scenario Heading", lineNo: 2},
	}

	_, result := new(specParser).createSpecification(tokens, new(conceptDictionary))

	c.Assert(result, NotNil)
	c.Assert(result.ok, Equals, false)
	c.Assert(result.error.message, Equals, "Step text should not have '{static}' or '{dynamic}' or '{special}'")
	c.Assert(result.error.lineNo, Equals, 3)
}

func (s *MySuite) TestSpecWithDataTable(c *C) {
	tokens := []*token{
		&token{kind: specKind, value: "Spec Heading"},
		&token{kind: commentKind, value: "Comment before data table"},
		&token{kind: tableHeader, args: []string{"id", "name"}},
		&token{kind: tableRow, args: []string{"1", "foo"}},
		&token{kind: tableRow, args: []string{"2", "bar"}},
		&token{kind: commentKind, value: "Comment before data table"},
	}

	spec, result := new(specParser).createSpecification(tokens, new(conceptDictionary))

	c.Assert(len(spec.items), Equals, 3)
	c.Assert(spec.items[0], Equals, spec.comments[0])
	c.Assert(spec.items[1], DeepEquals, &spec.dataTable)
	c.Assert(spec.items[2], Equals, spec.comments[1])

	c.Assert(result.ok, Equals, true)
	c.Assert(spec.dataTable, NotNil)
	c.Assert(len(spec.dataTable.get("id")), Equals, 2)
	c.Assert(len(spec.dataTable.get("name")), Equals, 2)
	c.Assert(spec.dataTable.get("id")[0].value, Equals, "1")
	c.Assert(spec.dataTable.get("id")[0].cellType, Equals, static)
	c.Assert(spec.dataTable.get("id")[1].value, Equals, "2")
	c.Assert(spec.dataTable.get("id")[1].cellType, Equals, static)
	c.Assert(spec.dataTable.get("name")[0].value, Equals, "foo")
	c.Assert(spec.dataTable.get("name")[0].cellType, Equals, static)
	c.Assert(spec.dataTable.get("name")[1].value, Equals, "bar")
	c.Assert(spec.dataTable.get("name")[1].cellType, Equals, static)
}

func (s *MySuite) TestStepWithInlineTable(c *C) {
	tokens := []*token{
		&token{kind: specKind, value: "Spec Heading", lineNo: 1},
		&token{kind: scenarioKind, value: "Scenario Heading", lineNo: 2},
		&token{kind: stepKind, value: "Step with inline table", lineNo: 3},
		&token{kind: tableHeader, args: []string{"id", "name"}},
		&token{kind: tableRow, args: []string{"1", "foo"}},
		&token{kind: tableRow, args: []string{"2", "bar"}},
	}

	spec, result := new(specParser).createSpecification(tokens, new(conceptDictionary))

	c.Assert(result.ok, Equals, true)
	step := spec.scenarios[0].steps[0]

	c.Assert(step.args[0].argType, Equals, tableArg)
	inlineTable := step.args[0].table
	c.Assert(inlineTable, NotNil)

	c.Assert(step.value, Equals, "Step with inline table {}")
	c.Assert(len(inlineTable.get("id")), Equals, 2)
	c.Assert(len(inlineTable.get("name")), Equals, 2)
	c.Assert(inlineTable.get("id")[0].value, Equals, "1")
	c.Assert(inlineTable.get("id")[0].cellType, Equals, static)
	c.Assert(inlineTable.get("id")[1].value, Equals, "2")
	c.Assert(inlineTable.get("id")[1].cellType, Equals, static)
	c.Assert(inlineTable.get("name")[0].value, Equals, "foo")
	c.Assert(inlineTable.get("name")[0].cellType, Equals, static)
	c.Assert(inlineTable.get("name")[1].value, Equals, "bar")
	c.Assert(inlineTable.get("name")[1].cellType, Equals, static)
}

func (s *MySuite) TestStepWithInlineTableWithDynamicParam(c *C) {
	tokens := []*token{
		&token{kind: specKind, value: "Spec Heading", lineNo: 1},
		&token{kind: tableHeader, args: []string{"type1", "type2"}},
		&token{kind: tableRow, args: []string{"1", "2"}},
		&token{kind: scenarioKind, value: "Scenario Heading", lineNo: 2},
		&token{kind: stepKind, value: "Step with inline table", lineNo: 3},
		&token{kind: tableHeader, args: []string{"id", "name"}},
		&token{kind: tableRow, args: []string{"1", "<type1>"}},
		&token{kind: tableRow, args: []string{"2", "<type2>"}},
	}

	spec, result := new(specParser).createSpecification(tokens, new(conceptDictionary))

	c.Assert(result.ok, Equals, true)
	step := spec.scenarios[0].steps[0]

	c.Assert(step.args[0].argType, Equals, tableArg)
	inlineTable := step.args[0].table
	c.Assert(inlineTable, NotNil)

	c.Assert(step.value, Equals, "Step with inline table {}")
	c.Assert(len(inlineTable.get("id")), Equals, 2)
	c.Assert(len(inlineTable.get("name")), Equals, 2)
	c.Assert(inlineTable.get("id")[0].value, Equals, "1")
	c.Assert(inlineTable.get("id")[0].cellType, Equals, static)
	c.Assert(inlineTable.get("id")[1].value, Equals, "2")
	c.Assert(inlineTable.get("id")[1].cellType, Equals, static)
	c.Assert(inlineTable.get("name")[0].value, Equals, "type1")
	c.Assert(inlineTable.get("name")[0].cellType, Equals, dynamic)
	c.Assert(inlineTable.get("name")[1].value, Equals, "type2")
	c.Assert(inlineTable.get("name")[1].cellType, Equals, dynamic)
}

func (s *MySuite) TestStepWithInlineTableWithUnResolvableDynamicParam(c *C) {
	tokens := []*token{
		&token{kind: specKind, value: "Spec Heading", lineNo: 1},
		&token{kind: tableHeader, args: []string{"type1", "type2"}},
		&token{kind: tableRow, args: []string{"1", "2"}},
		&token{kind: scenarioKind, value: "Scenario Heading", lineNo: 2},
		&token{kind: stepKind, value: "Step with inline table", lineNo: 3},
		&token{kind: tableHeader, args: []string{"id", "name"}},
		&token{kind: tableRow, args: []string{"1", "<invalid>"}},
		&token{kind: tableRow, args: []string{"2", "<type2>"}},
	}

	_, result := new(specParser).createSpecification(tokens, new(conceptDictionary))
	c.Assert(result.ok, Equals, false)
	c.Assert(result.error.message, Equals, "Dynamic param <invalid> could not be resolved")

}

func (s *MySuite) TestContextWithInlineTable(c *C) {
	tokens := []*token{
		&token{kind: specKind, value: "Spec Heading"},
		&token{kind: stepKind, value: "Context with inline table"},
		&token{kind: tableHeader, args: []string{"id", "name"}},
		&token{kind: tableRow, args: []string{"1", "foo"}},
		&token{kind: tableRow, args: []string{"2", "bar"}},
		&token{kind: tableRow, args: []string{"3", "not a <dynamic>"}},
		&token{kind: scenarioKind, value: "Scenario Heading"},
	}

	spec, result := new(specParser).createSpecification(tokens, new(conceptDictionary))
	c.Assert(len(spec.items), Equals, 2)
	c.Assert(spec.items[0], DeepEquals, spec.contexts[0])
	c.Assert(spec.items[1], Equals, spec.scenarios[0])

	c.Assert(result.ok, Equals, true)
	context := spec.contexts[0]

	c.Assert(context.args[0].argType, Equals, tableArg)
	inlineTable := context.args[0].table

	c.Assert(inlineTable, NotNil)
	c.Assert(context.value, Equals, "Context with inline table {}")
	c.Assert(len(inlineTable.get("id")), Equals, 3)
	c.Assert(len(inlineTable.get("name")), Equals, 3)
	c.Assert(inlineTable.get("id")[0].value, Equals, "1")
	c.Assert(inlineTable.get("id")[0].cellType, Equals, static)
	c.Assert(inlineTable.get("id")[1].value, Equals, "2")
	c.Assert(inlineTable.get("id")[1].cellType, Equals, static)
	c.Assert(inlineTable.get("id")[2].value, Equals, "3")
	c.Assert(inlineTable.get("id")[2].cellType, Equals, static)
	c.Assert(inlineTable.get("name")[0].value, Equals, "foo")
	c.Assert(inlineTable.get("name")[0].cellType, Equals, static)
	c.Assert(inlineTable.get("name")[1].value, Equals, "bar")
	c.Assert(inlineTable.get("name")[1].cellType, Equals, static)
	c.Assert(inlineTable.get("name")[2].value, Equals, "not a <dynamic>")
	c.Assert(inlineTable.get("name")[2].cellType, Equals, static)
}

func (s *MySuite) TestErrorWhenDataTableHasOnlyHeader(c *C) {
	tokens := []*token{
		&token{kind: specKind, value: "Spec Heading"},
		&token{kind: tableHeader, args: []string{"id", "name"}, lineNo: 3},
		&token{kind: scenarioKind, value: "Scenario Heading"},
	}

	_, result := new(specParser).createSpecification(tokens, new(conceptDictionary))

	c.Assert(result.ok, Equals, false)
	c.Assert(result.error.message, Equals, "Data table should have at least 1 data row")
	c.Assert(result.error.lineNo, Equals, 3)
}

func (s *MySuite) TestWarningWhenParsingMultipleDataTable(c *C) {
	tokens := []*token{
		&token{kind: specKind, value: "Spec Heading"},
		&token{kind: commentKind, value: "Comment before data table"},
		&token{kind: tableHeader, args: []string{"id", "name"}},
		&token{kind: tableRow, args: []string{"1", "foo"}},
		&token{kind: tableRow, args: []string{"2", "bar"}},
		&token{kind: commentKind, value: "Comment before data table"},
		&token{kind: tableHeader, args: []string{"phone"}, lineNo: 7},
		&token{kind: tableRow, args: []string{"1"}},
		&token{kind: tableRow, args: []string{"2"}},
	}

	_, result := new(specParser).createSpecification(tokens, new(conceptDictionary))

	c.Assert(result.ok, Equals, true)
	c.Assert(len(result.warnings), Equals, 1)
	c.Assert(result.warnings[0].String(), Equals, "line no: 7, Multiple data table present, ignoring table")

}

func (s *MySuite) TestWarningWhenParsingTableOccursWithoutStep(c *C) {
	tokens := []*token{
		&token{kind: specKind, value: "Spec Heading", lineNo: 1},
		&token{kind: scenarioKind, value: "Scenario Heading", lineNo: 2},
		&token{kind: tableHeader, args: []string{"id", "name"}, lineNo: 3},
		&token{kind: tableRow, args: []string{"1", "foo"}, lineNo: 4},
		&token{kind: tableRow, args: []string{"2", "bar"}, lineNo: 5},
		&token{kind: stepKind, value: "Step", lineNo: 6},
		&token{kind: commentKind, value: "comment in between", lineNo: 7},
		&token{kind: tableHeader, args: []string{"phone"}, lineNo: 8},
		&token{kind: tableRow, args: []string{"1"}},
		&token{kind: tableRow, args: []string{"2"}},
	}

	_, result := new(specParser).createSpecification(tokens, new(conceptDictionary))

	c.Assert(result.ok, Equals, true)
	c.Assert(len(result.warnings), Equals, 2)
	c.Assert(result.warnings[0].String(), Equals, "line no: 3, Table not associated with a step, ignoring table")
	c.Assert(result.warnings[1].String(), Equals, "line no: 8, Table not associated with a step, ignoring table")

}

func (s *MySuite) TestAddSpecTags(c *C) {
	tokens := []*token{
		&token{kind: specKind, value: "Spec Heading", lineNo: 1},
		&token{kind: tagKind, args: []string{"tag1", "tag2"}, lineNo: 2},
		&token{kind: scenarioKind, value: "Scenario Heading", lineNo: 3},
	}

	spec, result := new(specParser).createSpecification(tokens, new(conceptDictionary))

	c.Assert(result.ok, Equals, true)

	c.Assert(len(spec.tags.values), Equals, 2)
	c.Assert(spec.tags.values[0], Equals, "tag1")
	c.Assert(spec.tags.values[1], Equals, "tag2")
}

func (s *MySuite) TestAddSpecTagsAndScenarioTags(c *C) {
	tokens := []*token{
		&token{kind: specKind, value: "Spec Heading", lineNo: 1},
		&token{kind: tagKind, args: []string{"tag1", "tag2"}, lineNo: 2},
		&token{kind: scenarioKind, value: "Scenario Heading", lineNo: 3},
		&token{kind: tagKind, args: []string{"tag3", "tag4"}, lineNo: 2},
	}

	spec, result := new(specParser).createSpecification(tokens, new(conceptDictionary))

	c.Assert(result.ok, Equals, true)

	c.Assert(len(spec.tags.values), Equals, 2)
	c.Assert(spec.tags.values[0], Equals, "tag1")
	c.Assert(spec.tags.values[1], Equals, "tag2")

	tags := spec.scenarios[0].tags
	c.Assert(len(tags.values), Equals, 2)
	c.Assert(tags.values[0], Equals, "tag3")
	c.Assert(tags.values[1], Equals, "tag4")
}

func (s *MySuite) TestErrorOnAddingDynamicParamterWithoutADataTable(c *C) {
	tokens := []*token{
		&token{kind: specKind, value: "Spec Heading", lineNo: 1},
		&token{kind: scenarioKind, value: "Scenario Heading", lineNo: 2},
		&token{kind: stepKind, value: "Step with a {dynamic}", args: []string{"foo"}, lineNo: 3, lineText: "*Step with a <foo>"},
	}

	_, result := new(specParser).createSpecification(tokens, new(conceptDictionary))

	c.Assert(result.ok, Equals, false)
	c.Assert(result.error.message, Equals, "Dynamic parameter <foo> could not be resolved")
	c.Assert(result.error.lineNo, Equals, 3)

}

func (s *MySuite) TestErrorOnAddingDynamicParamterWithoutDataTableHeaderValue(c *C) {
	tokens := []*token{
		&token{kind: specKind, value: "Spec Heading", lineNo: 1},
		&token{kind: tableHeader, args: []string{"id, name"}, lineNo: 2},
		&token{kind: tableRow, args: []string{"123, hello"}, lineNo: 3},
		&token{kind: scenarioKind, value: "Scenario Heading", lineNo: 4},
		&token{kind: stepKind, value: "Step with a {dynamic}", args: []string{"foo"}, lineNo: 5, lineText: "*Step with a <foo>"},
	}

	_, result := new(specParser).createSpecification(tokens, new(conceptDictionary))

	c.Assert(result.ok, Equals, false)
	c.Assert(result.error.message, Equals, "Dynamic parameter <foo> could not be resolved")
	c.Assert(result.error.lineNo, Equals, 5)

}

func (s *MySuite) TestLookupaddArg(c *C) {
	lookup := new(argLookup)
	lookup.addArgName("param1")
	lookup.addArgName("param2")

	c.Assert(lookup.paramIndexMap["param1"], Equals, 0)
	c.Assert(lookup.paramIndexMap["param2"], Equals, 1)
	c.Assert(len(lookup.paramValue), Equals, 2)
	c.Assert(lookup.paramValue[0].name, Equals, "param1")
	c.Assert(lookup.paramValue[1].name, Equals, "param2")

}

func (s *MySuite) TestLookupContainsArg(c *C) {
	lookup := new(argLookup)
	lookup.addArgName("param1")
	lookup.addArgName("param2")

	c.Assert(lookup.containsArg("param1"), Equals, true)
	c.Assert(lookup.containsArg("param2"), Equals, true)
	c.Assert(lookup.containsArg("param3"), Equals, false)
}

func (s *MySuite) TestaddArgValue(c *C) {
	lookup := new(argLookup)
	lookup.addArgName("param1")
	lookup.addArgValue("param1", &stepArg{value: "value1", argType: static})
	lookup.addArgName("param2")
	lookup.addArgValue("param2", &stepArg{value: "value2", argType: dynamic})

	c.Assert(lookup.getArg("param1").value, Equals, "value1")
	c.Assert(lookup.getArg("param2").value, Equals, "value2")
}

func (s *MySuite) TestPanicForInvalidArg(c *C) {
	lookup := new(argLookup)

	c.Assert(func() { lookup.addArgValue("param1", &stepArg{value: "value1", argType: static}) }, Panics, "Accessing an invalid parameter (param1)")
	c.Assert(func() { lookup.getArg("param1") }, Panics, "Accessing an invalid parameter (param1)")
}

func (s *MySuite) TestGetLookupCopy(c *C) {
	originalLookup := new(argLookup)
	originalLookup.addArgName("param1")
	originalLookup.addArgValue("param1", &stepArg{value: "oldValue", argType: dynamic})

	copiedLookup := originalLookup.getCopy()
	copiedLookup.addArgValue("param1", &stepArg{value: "new value", argType: static})

	c.Assert(copiedLookup.getArg("param1").value, Equals, "new value")
	c.Assert(originalLookup.getArg("param1").value, Equals, "oldValue")
}

func (s *MySuite) TestGetLookupFromTableRow(c *C) {
	dataTable := new(table)
	dataTable.addHeaders([]string{"id", "name"})
	dataTable.addRowValues([]string{"1", "admin"})
	dataTable.addRowValues([]string{"2", "root"})

	emptyLookup := new(argLookup).fromDataTableRow(new(table), 0)
	lookup1 := new(argLookup).fromDataTableRow(dataTable, 0)
	lookup2 := new(argLookup).fromDataTableRow(dataTable, 1)

	c.Assert(emptyLookup.paramIndexMap, IsNil)

	c.Assert(lookup1.getArg("id").value, Equals, "1")
	c.Assert(lookup1.getArg("id").argType, Equals, static)
	c.Assert(lookup1.getArg("name").value, Equals, "admin")
	c.Assert(lookup1.getArg("name").argType, Equals, static)

	c.Assert(lookup2.getArg("id").value, Equals, "2")
	c.Assert(lookup2.getArg("id").argType, Equals, static)
	c.Assert(lookup2.getArg("name").value, Equals, "root")
	c.Assert(lookup2.getArg("name").argType, Equals, static)
}

func (s *MySuite) TestCreateStepFromSimpleConcept(c *C) {
	tokens := []*token{
		&token{kind: specKind, value: "Spec Heading", lineNo: 1},
		&token{kind: scenarioKind, value: "Scenario Heading", lineNo: 2},
		&token{kind: stepKind, value: "concept step", lineNo: 3},
	}

	conceptDictionary := new(conceptDictionary)
	firstStep := &step{value: "step 1"}
	secondStep := &step{value: "step 2"}
	conceptStep := &step{value: "concept step", isConcept: true, conceptSteps: []*step{firstStep, secondStep}}
	conceptDictionary.add([]*step{conceptStep}, "file.cpt")
	spec, result := new(specParser).createSpecification(tokens, conceptDictionary)
	c.Assert(result.ok, Equals, true)

	c.Assert(len(spec.scenarios[0].steps), Equals, 1)
	specConceptStep := spec.scenarios[0].steps[0]
	c.Assert(specConceptStep.isConcept, Equals, true)
	c.Assert(specConceptStep.conceptSteps[0], Equals, firstStep)
	c.Assert(specConceptStep.conceptSteps[1], Equals, secondStep)
}

func (s *MySuite) TestCreateStepFromConceptWithParameters(c *C) {
	tokens := []*token{
		&token{kind: specKind, value: "Spec Heading", lineNo: 1},
		&token{kind: scenarioKind, value: "Scenario Heading", lineNo: 2},
		&token{kind: stepKind, value: "create user {static}", args: []string{"foo"}, lineNo: 3},
		&token{kind: stepKind, value: "create user {static}", args: []string{"bar"}, lineNo: 4},
	}

	concepts, _ := new(conceptParser).parse("#create user <username> \n * enter user <username> \n *select \"finish\"")
	conceptsDictionary := new(conceptDictionary)
	conceptsDictionary.add(concepts, "file.cpt")

	spec, result := new(specParser).createSpecification(tokens, conceptsDictionary)
	c.Assert(result.ok, Equals, true)

	c.Assert(len(spec.scenarios[0].steps), Equals, 2)

	firstConceptStep := spec.scenarios[0].steps[0]
	c.Assert(firstConceptStep.isConcept, Equals, true)
	c.Assert(firstConceptStep.conceptSteps[0].value, Equals, "enter user {}")
	c.Assert(firstConceptStep.conceptSteps[0].args[0].value, Equals, "username")
	c.Assert(firstConceptStep.conceptSteps[1].value, Equals, "select {}")
	c.Assert(firstConceptStep.conceptSteps[1].args[0].value, Equals, "finish")
	c.Assert(len(firstConceptStep.lookup.paramValue), Equals, 1)
	c.Assert(firstConceptStep.lookup.paramValue[0].name, Equals, "username")
	c.Assert(firstConceptStep.lookup.paramValue[0].stepArg.value, Equals, "foo")

	secondConceptStep := spec.scenarios[0].steps[1]
	c.Assert(secondConceptStep.isConcept, Equals, true)
	c.Assert(secondConceptStep.conceptSteps[0].value, Equals, "enter user {}")
	c.Assert(secondConceptStep.conceptSteps[0].args[0].value, Equals, "username")
	c.Assert(secondConceptStep.conceptSteps[1].value, Equals, "select {}")
	c.Assert(secondConceptStep.conceptSteps[1].args[0].value, Equals, "finish")
	c.Assert(len(secondConceptStep.lookup.paramValue), Equals, 1)
	c.Assert(secondConceptStep.lookup.paramValue[0].name, Equals, "username")
	c.Assert(secondConceptStep.lookup.paramValue[0].stepArg.value, Equals, "bar")

}

func (s *MySuite) TestCreateStepFromConceptWithDynamicParameters(c *C) {
	tokens := []*token{
		&token{kind: specKind, value: "Spec Heading", lineNo: 1},
		&token{kind: tableHeader, args: []string{"id", "description"}, lineNo: 2},
		&token{kind: tableRow, args: []string{"123", "Admin fellow"}, lineNo: 3},
		&token{kind: scenarioKind, value: "Scenario Heading", lineNo: 4},
		&token{kind: stepKind, value: "create user {dynamic} and {dynamic}", args: []string{"id", "description"}, lineNo: 5},
		&token{kind: stepKind, value: "create user {static} and {static}", args: []string{"456", "Regular fellow"}, lineNo: 6},
	}

	concepts, _ := new(conceptParser).parse("#create user <user-id> and <user-description> \n * enter user <user-id> and <user-description> \n *select \"finish\"")
	conceptsDictionary := new(conceptDictionary)
	conceptsDictionary.add(concepts, "file.cpt")
	spec, result := new(specParser).createSpecification(tokens, conceptsDictionary)
	c.Assert(result.ok, Equals, true)

	c.Assert(len(spec.items), Equals, 2)
	c.Assert(spec.items[0], DeepEquals, &spec.dataTable)
	c.Assert(spec.items[1], Equals, spec.scenarios[0])

	scenarioItems := (spec.items[1]).(*scenario).items
	c.Assert(scenarioItems[0], Equals, spec.scenarios[0].steps[0])
	c.Assert(scenarioItems[1], DeepEquals, spec.scenarios[0].steps[1])

	c.Assert(len(spec.scenarios[0].steps), Equals, 2)

	firstConcept := spec.scenarios[0].steps[0]
	c.Assert(firstConcept.isConcept, Equals, true)
	c.Assert(firstConcept.conceptSteps[0].value, Equals, "enter user {} and {}")
	c.Assert(firstConcept.conceptSteps[0].args[0].argType, Equals, dynamic)
	c.Assert(firstConcept.conceptSteps[0].args[0].value, Equals, "user-id")
	c.Assert(firstConcept.conceptSteps[0].args[0].name, Equals, "user-id")
	c.Assert(firstConcept.conceptSteps[0].args[1].argType, Equals, dynamic)
	c.Assert(firstConcept.conceptSteps[0].args[1].value, Equals, "user-description")
	c.Assert(firstConcept.conceptSteps[0].args[1].name, Equals, "user-description")
	c.Assert(firstConcept.conceptSteps[1].value, Equals, "select {}")
	c.Assert(firstConcept.conceptSteps[1].args[0].value, Equals, "finish")
	c.Assert(firstConcept.conceptSteps[1].args[0].argType, Equals, static)

	c.Assert(len(firstConcept.lookup.paramValue), Equals, 2)
	c.Assert(firstConcept.lookup.paramValue[0].name, Equals, "user-id")
	c.Assert(firstConcept.lookup.paramValue[0].stepArg.value, Equals, "id")
	c.Assert(firstConcept.lookup.paramValue[0].stepArg.argType, Equals, dynamic)
	c.Assert(firstConcept.lookup.paramValue[1].name, Equals, "user-description")
	c.Assert(firstConcept.lookup.paramValue[1].stepArg.value, Equals, "description")
	c.Assert(firstConcept.lookup.paramValue[1].stepArg.argType, Equals, dynamic)

	secondConcept := spec.scenarios[0].steps[1]
	c.Assert(secondConcept.isConcept, Equals, true)
	c.Assert(secondConcept.conceptSteps[0].value, Equals, "enter user {} and {}")
	c.Assert(secondConcept.conceptSteps[0].args[0].argType, Equals, dynamic)
	c.Assert(secondConcept.conceptSteps[0].args[0].value, Equals, "user-id")
	c.Assert(secondConcept.conceptSteps[0].args[0].name, Equals, "user-id")
	c.Assert(secondConcept.conceptSteps[0].args[1].argType, Equals, dynamic)
	c.Assert(secondConcept.conceptSteps[0].args[1].value, Equals, "user-description")
	c.Assert(secondConcept.conceptSteps[0].args[1].name, Equals, "user-description")
	c.Assert(secondConcept.conceptSteps[1].value, Equals, "select {}")
	c.Assert(secondConcept.conceptSteps[1].args[0].value, Equals, "finish")
	c.Assert(secondConcept.conceptSteps[1].args[0].argType, Equals, static)

	c.Assert(len(secondConcept.lookup.paramValue), Equals, 2)
	c.Assert(secondConcept.lookup.paramValue[0].name, Equals, "user-id")
	c.Assert(secondConcept.lookup.paramValue[0].stepArg.value, Equals, "456")
	c.Assert(secondConcept.lookup.paramValue[0].stepArg.argType, Equals, static)
	c.Assert(secondConcept.lookup.paramValue[1].name, Equals, "user-description")
	c.Assert(secondConcept.lookup.paramValue[1].stepArg.value, Equals, "Regular fellow")
	c.Assert(secondConcept.lookup.paramValue[1].stepArg.argType, Equals, static)

}

func (s *MySuite) TestCreateStepFromConceptWithInlineTable(c *C) {
	tokens := []*token{
		&token{kind: specKind, value: "Spec Heading", lineNo: 1},
		&token{kind: scenarioKind, value: "Scenario Heading", lineNo: 4},
		&token{kind: stepKind, value: "create users", lineNo: 3},
		&token{kind: tableHeader, args: []string{"id", "description"}, lineNo: 4},
		&token{kind: tableRow, args: []string{"123", "Admin"}, lineNo: 5},
		&token{kind: tableRow, args: []string{"456", "normal fellow"}, lineNo: 6},
	}

	concepts, _ := new(conceptParser).parse("#create users <table> \n * enter details from <table> \n *select \"finish\"")
	conceptsDictionary := new(conceptDictionary)
	conceptsDictionary.add(concepts, "file.cpt")
	spec, result := new(specParser).createSpecification(tokens, conceptsDictionary)
	c.Assert(result.ok, Equals, true)

	steps := spec.scenarios[0].steps
	c.Assert(len(steps), Equals, 1)
	c.Assert(steps[0].isConcept, Equals, true)
	c.Assert(steps[0].value, Equals, "create users {}")
	c.Assert(len(steps[0].args), Equals, 1)
	c.Assert(steps[0].args[0].argType, Equals, tableArg)
	c.Assert(len(steps[0].conceptSteps), Equals, 2)
}

func (s *MySuite) TestCreateStepFromConceptWithInlineTableHavingDynamicParam(c *C) {
	tokens := []*token{
		&token{kind: specKind, value: "Spec Heading", lineNo: 1},
		&token{kind: tableHeader, args: []string{"id", "description"}, lineNo: 2},
		&token{kind: tableRow, args: []string{"123", "Admin"}, lineNo: 3},
		&token{kind: tableRow, args: []string{"456", "normal fellow"}, lineNo: 4},
		&token{kind: scenarioKind, value: "Scenario Heading", lineNo: 5},
		&token{kind: stepKind, value: "create users", lineNo: 6},
		&token{kind: tableHeader, args: []string{"user-id", "description", "name"}, lineNo: 7},
		&token{kind: tableRow, args: []string{"<id>", "<description>", "root"}, lineNo: 8},
		&token{kind: stepKind, value: "create users", lineNo: 9},
		&token{kind: tableHeader, args: []string{"user-id", "description", "name"}, lineNo: 10},
		&token{kind: tableRow, args: []string{"1", "normal", "wheel"}, lineNo: 11},
	}

	concepts, _ := new(conceptParser).parse("#create users <id> \n * enter details from <id> \n *select \"finish\"")
	conceptsDictionary := new(conceptDictionary)
	conceptsDictionary.add(concepts, "file.cpt")
	spec, result := new(specParser).createSpecification(tokens, conceptsDictionary)
	c.Assert(result.ok, Equals, true)

	steps := spec.scenarios[0].steps
	c.Assert(len(steps), Equals, 2)
	c.Assert(steps[0].isConcept, Equals, true)
	c.Assert(steps[1].isConcept, Equals, true)
	c.Assert(steps[0].value, Equals, "create users {}")
	c.Assert(len(steps[0].args), Equals, 1)
	c.Assert(steps[0].args[0].argType, Equals, tableArg)
	table := steps[0].args[0].table
	c.Assert(table.get("user-id")[0].value, Equals, "id")
	c.Assert(table.get("user-id")[0].cellType, Equals, dynamic)
	c.Assert(table.get("description")[0].value, Equals, "description")
	c.Assert(table.get("description")[0].cellType, Equals, dynamic)
	c.Assert(table.get("name")[0].value, Equals, "root")
	c.Assert(table.get("name")[0].cellType, Equals, static)
	c.Assert(len(steps[0].conceptSteps), Equals, 2)
}

func (s *MySuite) TestPopulateFragmentsForSimpleStep(c *C) {
	step := &step{value: "This is a simple step"}

	step.populateFragments()

	c.Assert(len(step.fragments), Equals, 1)
	fragment := step.fragments[0]
	c.Assert(fragment.GetText(), Equals, "This is a simple step")
	c.Assert(fragment.GetFragmentType(), Equals, Fragment_Text)
}

func (s *MySuite) TestPopulateFragmentsForStepWithParameters(c *C) {
	arg1 := &stepArg{value: "first", argType: static}
	arg2 := &stepArg{value: "second", argType: dynamic}
	argTable := new(table)
	headers := []string{"header1", "header2"}
	row1 := []string{"row1", "row2"}
	argTable.addHeaders(headers)
	argTable.addRowValues(row1)
	arg3 := &stepArg{table: *argTable, argType: tableArg}
	stepArgs := []*stepArg{arg1, arg2, arg3}
	step := &step{value: "{} step with {}, {} works", args: stepArgs}

	step.populateFragments()

	c.Assert(len(step.fragments), Equals, 6)
	fragment1 := step.fragments[0]
	c.Assert(fragment1.GetFragmentType(), Equals, Fragment_Parameter)
	c.Assert(fragment1.GetParameter().GetValue(), Equals, "first")
	c.Assert(fragment1.GetParameter().GetParameterType(), Equals, Parameter_Static)

	fragment2 := step.fragments[1]
	c.Assert(fragment2.GetText(), Equals, " step with ")
	c.Assert(fragment2.GetFragmentType(), Equals, Fragment_Text)

	fragment3 := step.fragments[2]
	c.Assert(fragment3.GetFragmentType(), Equals, Fragment_Parameter)
	c.Assert(fragment3.GetParameter().GetValue(), Equals, "second")
	c.Assert(fragment3.GetParameter().GetParameterType(), Equals, Parameter_Dynamic)

	fragment4 := step.fragments[3]
	c.Assert(fragment4.GetText(), Equals, ", ")
	c.Assert(fragment4.GetFragmentType(), Equals, Fragment_Text)

	fragment5 := step.fragments[4]
	c.Assert(fragment5.GetFragmentType(), Equals, Fragment_Parameter)
	c.Assert(fragment5.GetParameter().GetParameterType(), Equals, Parameter_Table)
	protoTable := fragment5.GetParameter().GetTable()
	c.Assert(protoTable.GetHeaders().GetCells(), DeepEquals, headers)
	c.Assert(len(protoTable.GetRows()), Equals, 1)
	c.Assert(protoTable.GetRows()[0].GetCells(), DeepEquals, row1)
}
