package main

import (
	"code.google.com/p/goprotobuf/proto"
	"common"
	"fmt"
	"log"
	"net"
	"strconv"
	"sync"
)

const (
	apiPortEnvVariableName = "GAUGE_API_PORT"
	API_STATIC_PORT        = 8889
)

func makeListOfAvailableSteps(runnerConn net.Conn) {
	addStepValuesToAvailableSteps(getStepsFromRunner(runnerConn))
	specFiles := findSpecsFilesIn(common.SpecsDirectoryName)
	dictionary, _ := createConceptsDictionary(true)
	availableSpecs = parseSpecFiles(specFiles, dictionary)
	findAvailableStepsInSpecs(availableSpecs)
}

func parseSpecFiles(specFiles []string, dictionary *conceptDictionary) []*specification {
	specs := make([]*specification, 0)
	for _, file := range specFiles {
		specContent, err := common.ReadFileContents(file)
		if err != nil {
			continue
		}
		parser := new(specParser)
		specification, result := parser.parse(specContent, dictionary)

		if result.ok {
			specs = append(specs, specification)
		}
	}
	return specs
}

func findAvailableStepsInSpecs(specs []*specification) {
	for _, spec := range specs {
		addStepsToAvailableSteps(spec.contexts)
		for _, scenario := range spec.scenarios {
			addStepsToAvailableSteps(scenario.steps)
		}
	}
}

func addStepsToAvailableSteps(steps []*step) {
	for _, step := range steps {
		if _, ok := availableStepsMap[step.value]; !ok {
			availableStepsMap[step.value] = true
		}
	}
}

func addStepValuesToAvailableSteps(stepValues []string) {
	for _, step := range stepValues {
		addToAvailableSteps(step)
	}
}

func addToAvailableSteps(step string) {
	if _, ok := availableStepsMap[step]; !ok {
		availableStepsMap[step] = true
	}
}

func getAvailableStepNames() []string {
	stepNames := make([]string, 0)
	for stepName, _ := range availableStepsMap {
		stepNames = append(stepNames, stepName)
	}
	return stepNames
}

func getStepsFromRunner(runnerConnection net.Conn) []string {
	steps := make([]string, 0)
	if runnerConnection == nil {
		var connErr error
		runnerConnection, connErr = startRunnerAndMakeConnection(getProjectManifest())
		if connErr == nil {
			steps = append(steps, requestForSteps(runnerConnection)...)
			killRunner(runnerConnection)
		}
	} else {
		steps = append(steps, requestForSteps(runnerConnection)...)
	}
	return steps
}

func requestForSteps(connection net.Conn) []string {
	message, err := getResponse(connection, createGetStepNamesRequest())
	if err == nil {
		allStepsResponse := message.GetStepNamesResponse()
		return allStepsResponse.GetSteps()
	}
	return make([]string, 0)
}

func killRunner(connection net.Conn) error {
	id := common.GetUniqueId()
	message := &Message{MessageId: &id, MessageType: Message_KillProcessRequest.Enum(),
		KillProcessRequest: &KillProcessRequest{}}

	return writeMessage(connection, message)
}

func createGetStepNamesRequest() *Message {
	return &Message{MessageType: Message_StepNamesRequest.Enum(), StepNamesRequest: &StepNamesRequest{}}
}

func startAPIService(port int, apiChannel chan bool, wg *sync.WaitGroup) {
	defer wg.Done()
	gaugeListener, err := newGaugeListener(port)
	if err != nil {
		fmt.Printf("[Error] Failed to start API. %s\n", err.Error())
		apiChannel <- false
	}
	if port == 0 {
		if err := common.SetEnvVariable(apiPortEnvVariableName, strconv.Itoa(gaugeListener.tcpListener.Addr().(*net.TCPAddr).Port)); err != nil {
			fmt.Printf("Failed to set Env variable %s. %s", apiPortEnvVariableName, err.Error())
			apiChannel <- false
			return
		}
	}
	apiChannel <- true
	gaugeListener.acceptConnections(&GaugeApiMessageHandler{})
}

type GaugeApiMessageHandler struct{}

func (handler *GaugeApiMessageHandler) messageReceived(bytesRead []byte, conn net.Conn) {
	apiMessage := &APIMessage{}
	err := proto.Unmarshal(bytesRead, apiMessage)
	if err != nil {
		log.Printf("[Warning] Failed to read proto message: %s\n", err.Error())
	} else {
		messageType := apiMessage.GetMessageType()
		switch messageType {
		case APIMessage_GetProjectRootRequest:
			handler.respondToProjectRootRequest(apiMessage, conn)
			break
		case APIMessage_GetAllStepsRequest:
			handler.respondToGetAllStepsRequest(apiMessage, conn)
			break
		case APIMessage_GetAllSpecsRequest:
			handler.respondToGetAllSpecsRequest(apiMessage, conn)
			break
		}
	}
}

func (handler *GaugeApiMessageHandler) respondToProjectRootRequest(message *APIMessage, conn net.Conn) {
	root, err := common.GetProjectRoot()
	if err != nil {
		fmt.Printf("[Warning] Failed to find project root while responding to API request. %s\n", err.Error())
		root = ""
	}
	projectRootResponse := &GetProjectRootResponse{ProjectRoot: proto.String(root)}
	responseApiMessage := &APIMessage{MessageType: APIMessage_GetProjectRootResponse.Enum(), MessageId: message.MessageId, ProjectRootResponse: projectRootResponse}
	handler.sendMessage(responseApiMessage, conn)
}

func (handler *GaugeApiMessageHandler) respondToGetAllStepsRequest(message *APIMessage, conn net.Conn) {
	getAllStepsResponse := &GetAllStepsResponse{Steps: getAvailableStepNames()}
	responseApiMessage := &APIMessage{MessageType: APIMessage_GetAllStepResponse.Enum(), MessageId: message.MessageId, AllStepsResponse: getAllStepsResponse}
	handler.sendMessage(responseApiMessage, conn)
}

func (handler *GaugeApiMessageHandler) respondToGetAllSpecsRequest(message *APIMessage, conn net.Conn) {
	getAllSpecsResponse := handler.createGetAllSpecsResponseMessageFor(availableSpecs)
	responseApiMessage := &APIMessage{MessageType: APIMessage_GetAllSpecsResponse.Enum(), MessageId: message.MessageId, AllSpecsResponse: getAllSpecsResponse}
	handler.sendMessage(responseApiMessage, conn)
}

func (handler *GaugeApiMessageHandler) createGetAllSpecsResponseMessageFor(specs []*specification) *GetAllSpecsResponse {
	protoSpecs := make([]*ProtoSpec, 0)
	for _, spec := range specs {
		protoSpecs = append(protoSpecs, convertToProtoSpec(spec))
	}
	return &GetAllSpecsResponse{Specs: protoSpecs}
}

func convertToProtoSpec(spec *specification) *ProtoSpec {
	protoItems := make([]*ProtoItem, 0)
	for _, item := range spec.items {
		protoItems = append(protoItems, convertToProtoItem(item))
	}
	return &ProtoSpec{Items: protoItems}
}

func convertToProtoItem(item item) *ProtoItem {
	switch item.kind() {
	case scenarioKind:
		return convertToProtoScenarioItem(item.(*scenario))
	case stepKind:
		return convertToProtoStepItem(item.(*step))
	case commentKind:
		return convertToProtoCommentItem(item.(*comment))
	case tagKind:
		return convertToProtoTagsItem(item.(*tags))
	case tableKind:
		return convertToProtoTableItem(item.(*table))
	}
	return nil
}

func convertToProtoStepItem(step *step) *ProtoItem {
	if step.isConcept {
		return convertToProtoConcept(step)
	}
	return &ProtoItem{ItemType: ProtoItem_Step.Enum(), Step: convertToProtoStep(step)}
}

func convertToProtoScenarioItem(scenario *scenario) *ProtoItem {
	scenarioItems := make([]*ProtoItem, 0)
	for _, item := range scenario.items {
		scenarioItems = append(scenarioItems, convertToProtoItem(item))
	}
	protoScenario := &ProtoScenario{ScenarioItems: scenarioItems}
	return &ProtoItem{ItemType: ProtoItem_Scenario.Enum(), Scenario: protoScenario}
}

func convertToProtoConcept(concept *step) *ProtoItem {
	protoConcept := &ProtoConcept{ConceptStep: convertToProtoStep(concept), Steps: convertToProtoSteps(concept.conceptSteps)}
	protoConceptItem := &ProtoItem{ItemType: ProtoItem_Concept.Enum(), Concept: protoConcept}
	return protoConceptItem
}

func convertToProtoStep(step *step) *ProtoStep {
	result := getProtoStepResult(step.executionResults)
	return &ProtoStep{Text: proto.String(step.lineText), Parameters: convertToProtoParameters(step.args), Result: result}
}

func getProtoStepResult(executionResults []*stepExecutionResult) []*ProtoStepExecutionResult {
	protoResults := make([]*ProtoStepExecutionResult, 0)
	for _, result := range executionResults {
		protoResult := &ProtoStepExecutionResult{IsPassed: proto.Bool(result.isPassed), StackTrace: proto.String(result.stackTrace), Argument: result.argument}
		protoResults = append(protoResults, protoResult)
	}
	return protoResults
}

func convertToProtoSteps(steps []*step) []*ProtoStep {
	protoSteps := make([]*ProtoStep, 0)
	for _, step := range steps {
		protoSteps = append(protoSteps, convertToProtoStep(step))
	}
	return protoSteps
}

func convertToProtoCommentItem(comment *comment) *ProtoItem {
	return &ProtoItem{ItemType: ProtoItem_Comment.Enum(), Comment: &ProtoComment{Text: proto.String(comment.value)}}
}

func convertToProtoTagsItem(tags *tags) *ProtoItem {
	return &ProtoItem{ItemType: ProtoItem_Tags.Enum(), Tags: &ProtoTags{Tags: tags.values}}
}

func convertToProtoTableItem(table *table) *ProtoItem {
	return &ProtoItem{ItemType: ProtoItem_Table.Enum(), Table: convertToProtoTableParam(table)}
}

func convertToProtoParameters(args []*stepArg) []*Parameter {
	params := make([]*Parameter, 0)
	for _, arg := range args {
		params = append(params, convertToProtoParameter(arg))
	}
	return params
}

func convertToProtoParameter(arg *stepArg) *Parameter {
	switch arg.argType {
	case static:
		return &Parameter{ParameterType: Parameter_Static.Enum(), Value: proto.String(arg.value)}
	case dynamic:
		return &Parameter{ParameterType: Parameter_Dynamic.Enum(), Value: proto.String(arg.value)}
	case tableArg:
		return &Parameter{ParameterType: Parameter_Table.Enum(), Table: convertToProtoTableParam(&arg.table)}
	}
	return nil
}

func convertToProtoTableParam(table *table) *ProtoTable {
	protoTableParam := &ProtoTable{Rows: make([]*ProtoTableRow, 0)}
	protoTableParam.Headers = &ProtoTableRow{Cells: table.headers}
	for _, row := range table.getRows() {
		protoTableParam.Rows = append(protoTableParam.Rows, &ProtoTableRow{Cells: row})
	}
	return protoTableParam
}

func (handler *GaugeApiMessageHandler) sendMessage(message *APIMessage, conn net.Conn) {
	if err := writeMessage(conn, message); err != nil {
		fmt.Printf("[Warning] Failed to respond to API request. %s\n", err.Error())
	}
}
