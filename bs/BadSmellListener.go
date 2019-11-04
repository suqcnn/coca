package call

import (
	"github.com/antlr/antlr4/runtime/Go/antlr"
	. "github.com/phodal/coca/bs/models"
	. "github.com/phodal/coca/language/java"
	"reflect"
	"strings"
)

var imports []string
var clzs []string
var currentPkg string
var currentClz string
var methods []JFullMethod
var methodCalls []JFullMethodCall
var currentType string

var fields = make(map[string]string)
var localVars = make(map[string]string)
var formalParameters = make(map[string]string)

func NewBadSmellListener() *BadSmellListener {
	currentClz = ""
	currentPkg = ""
	methods = nil
	methodCalls = nil
	return &BadSmellListener{}
}

type BadSmellListener struct {
	BaseJavaParserListener
}

func (s *BadSmellListener) getNodeInfo() *JFullClassNode {
	return &JFullClassNode{currentPkg, currentClz, currentType, "", methods, methodCalls}
}

func (s *BadSmellListener) EnterPackageDeclaration(ctx *PackageDeclarationContext) {
	currentPkg = ctx.QualifiedName().GetText()
}

func (s *BadSmellListener) EnterImportDeclaration(ctx *ImportDeclarationContext) {
	importText := ctx.QualifiedName().GetText()
	imports = append(imports, importText)
}

func (s *BadSmellListener) EnterClassDeclaration(ctx *ClassDeclarationContext) {
	currentType = "Class"
	currentClz = ctx.IDENTIFIER().GetText()
}

func (s *BadSmellListener) EnterInterfaceDeclaration(ctx *InterfaceDeclarationContext) {
	currentType = "Interface"
	currentClz = ctx.IDENTIFIER().GetText()
}

func (s *BadSmellListener) EnterInterfaceMethodDeclaration(ctx *InterfaceMethodDeclarationContext) {
	startLine := ctx.GetStart().GetLine()
	startLinePosition := ctx.IDENTIFIER().GetSymbol().GetColumn()
	stopLine := ctx.GetStop().GetLine()
	name := ctx.IDENTIFIER().GetText()
	stopLinePosition := startLinePosition + len(name)
	methodBody := ctx.MethodBody().GetText()

	typeType := ctx.TypeTypeOrVoid().GetText()
	var params []JFullParameter = nil
	parameters := ctx.FormalParameters()
	if parameters != nil {

	}

	method := &JFullMethod{
		name,
		typeType,
		startLine,
		startLinePosition,
		stopLine,
		stopLinePosition,
		methodBody,
		params,
	}

	methods = append(methods, *method)
}

func (s *BadSmellListener) EnterFormalParameter(ctx *FormalParameterContext) {
	formalParameters[ctx.VariableDeclaratorId().GetText()] = ctx.TypeType().GetText()
}

func (s *BadSmellListener) EnterFieldDeclaration(ctx *FieldDeclarationContext) {
	declarators := ctx.VariableDeclarators()
	variableName := declarators.GetParent().GetChild(0).(antlr.ParseTree).GetText()
	fields[variableName] = ctx.TypeType().GetText()
}

func (s *BadSmellListener) EnterLocalVariableDeclaration(ctx *LocalVariableDeclarationContext) {
	typ := ctx.GetChild(0).(antlr.ParseTree).GetText()
	variableName := ctx.GetChild(1).GetChild(0).GetChild(0).(antlr.ParseTree).GetText()
	localVars[variableName] = typ
}

func (s *BadSmellListener) EnterMethodDeclaration(ctx *MethodDeclarationContext) {
	startLine := ctx.GetStart().GetLine()
	startLinePosition := ctx.IDENTIFIER().GetSymbol().GetColumn()
	stopLine := ctx.GetStop().GetLine()
	name := ctx.IDENTIFIER().GetText()
	stopLinePosition := startLinePosition + len(name)
	//XXX: find the start position of {, not public

	typeType := ctx.TypeTypeOrVoid().GetText()
	methodBody := ctx.MethodBody().GetText()

	parameters := ctx.FormalParameters()
	if parameters != nil {

	}

	method := &JFullMethod{
		name,
		typeType,
		startLine,
		startLinePosition,
		stopLine,
		stopLinePosition,
		methodBody,
		nil,
	}
	methods = append(methods, *method)
}

func (s *BadSmellListener) EnterMethodCall(ctx *MethodCallContext) {
	var targetCtx = ctx.GetParent().GetChild(0).(antlr.ParseTree).GetText()
	var targetType = parseTargetType(targetCtx)
	callee := ctx.GetChild(0).(antlr.ParseTree).GetText()

	startLine := ctx.GetStart().GetLine()
	startLinePosition := ctx.GetStart().GetColumn()
	stopLine := ctx.GetStop().GetLine()
	stopLinePosition := startLinePosition + len(callee)

	//typeType := ctx.GetChild(0).(antlr.ParseTree).TypeTypeOrVoid().GetText()

	fullType := warpTargetFullType(targetType)
	if fullType != "" {
		jMethodCall := &JFullMethodCall{removeTarget(fullType), "", targetType, callee, startLine, startLinePosition, stopLine, stopLinePosition}
		methodCalls = append(methodCalls, *jMethodCall)
	} else {
		if ctx.GetText() == targetType {
			methodName := ctx.IDENTIFIER().GetText()
			jMethodCall := &JFullMethodCall{currentPkg, "", currentClz, methodName, startLine, startLinePosition, stopLine, stopLinePosition}
			methodCalls = append(methodCalls, *jMethodCall)
		}
	}
}

func (s *BadSmellListener) EnterExpression(ctx *ExpressionContext) {
	// lambda BlogPO::of
	if ctx.COLONCOLON() != nil {
		text := ctx.Expression(0).GetText()
		methodName := ctx.IDENTIFIER().GetText()
		targetType := parseTargetType(text)
		fullType := warpTargetFullType(targetType)

		startLine := ctx.GetStart().GetLine()
		startLinePosition := ctx.GetStart().GetColumn()
		stopLine := ctx.GetStop().GetLine()
		stopLinePosition := startLinePosition + len(text)

		jMethodCall := &JFullMethodCall{removeTarget(fullType), "", targetType, methodName, startLine, startLinePosition, stopLine, stopLinePosition}
		methodCalls = append(methodCalls, *jMethodCall)
	}
}

func (s *BadSmellListener) appendClasses(classes []string) {
	clzs = classes
}

func removeTarget(fullType string) string {
	split := strings.Split(fullType, ".")
	return strings.Join(split[:len(split)-1], ".")
}

func parseTargetType(targetCtx string) string {
	targetVar := targetCtx
	targetType := targetVar

	//TODO: update this reflect
	typeOf := reflect.TypeOf(targetCtx).String()
	if strings.HasSuffix(typeOf, "MethodCallContext") {
		targetType = currentClz;
	} else {
		fieldType := fields[targetVar]
		formalType := formalParameters[targetVar]
		localVarType := localVars[targetVar]
		if fieldType != "" {
			targetType = fieldType
		} else if formalType != "" {
			targetType = formalType;
		} else if localVarType != "" {
			targetType = localVarType;
		}
	}

	return targetType
}

func warpTargetFullType(targetType string) string {
	if strings.EqualFold(currentClz, targetType) {
		return currentPkg + "." + targetType
	}

	for index := range imports {
		imp := imports[index]
		if strings.HasSuffix(imp, targetType) {
			return imp
		}
	}

	//maybe the same package
	for _, clz := range clzs {
		if strings.HasSuffix(clz, "."+targetType) {
			return clz
		}
	}

	//1. current package, 2. import by *
	return ""
}