package toml

import (
	"testing"

	pbSettings "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/stretchr/testify/suite"
)

// EvaluatorSourceProcessor defines the interface for evaluator source processing functions
type EvaluatorSourceProcessor interface {
	ProcessSource(evaluator interface{}, config map[string]any)
	CreateEvaluator() interface{}
	GetSource(evaluator interface{}) interface{}
	CreateCodeSource(code string) interface{}
	CreateUriSource(uri string) interface{}
	GetCodeFromSource(source interface{}) (string, bool)
	GetUriFromSource(source interface{}) (string, bool)
	GetExampleCode() string
	GetExampleUri() string
	GetIrrelevantConfigKey() string
}

// StarlarkSourceProcessor implements EvaluatorSourceProcessor for Starlark
type StarlarkSourceProcessor struct{}

func (p *StarlarkSourceProcessor) ProcessSource(evaluator interface{}, config map[string]any) {
	if evaluator == nil {
		processStarlarkSource(nil, config)
		return
	}
	processStarlarkSource(evaluator.(*pbSettings.StarlarkEvaluator), config)
}

func (p *StarlarkSourceProcessor) CreateEvaluator() interface{} {
	return &pbSettings.StarlarkEvaluator{}
}

func (p *StarlarkSourceProcessor) GetSource(evaluator interface{}) interface{} {
	return evaluator.(*pbSettings.StarlarkEvaluator).Source
}

func (p *StarlarkSourceProcessor) CreateCodeSource(code string) interface{} {
	return &pbSettings.StarlarkEvaluator_Code{Code: code}
}

func (p *StarlarkSourceProcessor) CreateUriSource(uri string) interface{} {
	return &pbSettings.StarlarkEvaluator_Uri{Uri: uri}
}

func (p *StarlarkSourceProcessor) GetCodeFromSource(source interface{}) (string, bool) {
	if codeSource, ok := source.(*pbSettings.StarlarkEvaluator_Code); ok {
		return codeSource.Code, true
	}
	return "", false
}

func (p *StarlarkSourceProcessor) GetUriFromSource(source interface{}) (string, bool) {
	if uriSource, ok := source.(*pbSettings.StarlarkEvaluator_Uri); ok {
		return uriSource.Uri, true
	}
	return "", false
}

func (p *StarlarkSourceProcessor) GetExampleCode() string {
	return "result = 'hello'"
}

func (p *StarlarkSourceProcessor) GetExampleUri() string {
	return "file://script.star"
}

func (p *StarlarkSourceProcessor) GetIrrelevantConfigKey() string {
	return "timeout"
}

// ExtismSourceProcessor implements EvaluatorSourceProcessor for Extism
type ExtismSourceProcessor struct{}

func (p *ExtismSourceProcessor) ProcessSource(evaluator interface{}, config map[string]any) {
	if evaluator == nil {
		processExtismSource(nil, config)
		return
	}
	processExtismSource(evaluator.(*pbSettings.ExtismEvaluator), config)
}

func (p *ExtismSourceProcessor) CreateEvaluator() interface{} {
	return &pbSettings.ExtismEvaluator{}
}

func (p *ExtismSourceProcessor) GetSource(evaluator interface{}) interface{} {
	return evaluator.(*pbSettings.ExtismEvaluator).Source
}

func (p *ExtismSourceProcessor) CreateCodeSource(code string) interface{} {
	return &pbSettings.ExtismEvaluator_Code{Code: code}
}

func (p *ExtismSourceProcessor) CreateUriSource(uri string) interface{} {
	return &pbSettings.ExtismEvaluator_Uri{Uri: uri}
}

func (p *ExtismSourceProcessor) GetCodeFromSource(source interface{}) (string, bool) {
	if codeSource, ok := source.(*pbSettings.ExtismEvaluator_Code); ok {
		return codeSource.Code, true
	}
	return "", false
}

func (p *ExtismSourceProcessor) GetUriFromSource(source interface{}) (string, bool) {
	if uriSource, ok := source.(*pbSettings.ExtismEvaluator_Uri); ok {
		return uriSource.Uri, true
	}
	return "", false
}

func (p *ExtismSourceProcessor) GetExampleCode() string {
	return "base64encodedwasm"
}

func (p *ExtismSourceProcessor) GetExampleUri() string {
	return "file://plugin.wasm"
}

func (p *ExtismSourceProcessor) GetIrrelevantConfigKey() string {
	return "entrypoint"
}

// RisorSourceProcessor implements EvaluatorSourceProcessor for Risor
type RisorSourceProcessor struct{}

func (p *RisorSourceProcessor) ProcessSource(evaluator interface{}, config map[string]any) {
	if evaluator == nil {
		processRisorSource(nil, config)
		return
	}
	processRisorSource(evaluator.(*pbSettings.RisorEvaluator), config)
}

func (p *RisorSourceProcessor) CreateEvaluator() interface{} {
	return &pbSettings.RisorEvaluator{}
}

func (p *RisorSourceProcessor) GetSource(evaluator interface{}) interface{} {
	return evaluator.(*pbSettings.RisorEvaluator).Source
}

func (p *RisorSourceProcessor) CreateCodeSource(code string) interface{} {
	return &pbSettings.RisorEvaluator_Code{Code: code}
}

func (p *RisorSourceProcessor) CreateUriSource(uri string) interface{} {
	return &pbSettings.RisorEvaluator_Uri{Uri: uri}
}

func (p *RisorSourceProcessor) GetCodeFromSource(source interface{}) (string, bool) {
	if codeSource, ok := source.(*pbSettings.RisorEvaluator_Code); ok {
		return codeSource.Code, true
	}
	return "", false
}

func (p *RisorSourceProcessor) GetUriFromSource(source interface{}) (string, bool) {
	if uriSource, ok := source.(*pbSettings.RisorEvaluator_Uri); ok {
		return uriSource.Uri, true
	}
	return "", false
}

func (p *RisorSourceProcessor) GetExampleCode() string {
	return "print('hello')"
}

func (p *RisorSourceProcessor) GetExampleUri() string {
	return "file://script.risor"
}

func (p *RisorSourceProcessor) GetIrrelevantConfigKey() string {
	return "timeout"
}

// EvaluatorSourceTestSuite is a test suite for evaluator source processing functions
type EvaluatorSourceTestSuite struct {
	suite.Suite
	processor EvaluatorSourceProcessor
}

// TestNilEvaluator tests nil evaluator handling
func (s *EvaluatorSourceTestSuite) TestNilEvaluator() {
	config := map[string]any{
		"code": s.processor.GetExampleCode(),
	}

	// Should not panic when evaluator is nil
	s.processor.ProcessSource(nil, config)
}

// TestWithCode tests code source processing
func (s *EvaluatorSourceTestSuite) TestWithCode() {
	eval := s.processor.CreateEvaluator()
	config := map[string]any{
		"code": s.processor.GetExampleCode(),
	}

	s.processor.ProcessSource(eval, config)

	source := s.processor.GetSource(eval)
	s.Require().NotNil(source, "Source should be set")

	code, ok := s.processor.GetCodeFromSource(source)
	s.Require().True(ok, "Source should be code type")
	s.Equal(s.processor.GetExampleCode(), code, "Code should be set correctly")
}

// TestWithUri tests URI source processing
func (s *EvaluatorSourceTestSuite) TestWithUri() {
	eval := s.processor.CreateEvaluator()
	config := map[string]any{
		"uri": s.processor.GetExampleUri(),
	}

	s.processor.ProcessSource(eval, config)

	source := s.processor.GetSource(eval)
	s.Require().NotNil(source, "Source should be set")

	uri, ok := s.processor.GetUriFromSource(source)
	s.Require().True(ok, "Source should be URI type")
	s.Equal(s.processor.GetExampleUri(), uri, "URI should be set correctly")
}

// TestCodeTakesPrecedence tests code precedence over URI
func (s *EvaluatorSourceTestSuite) TestCodeTakesPrecedence() {
	eval := s.processor.CreateEvaluator()
	config := map[string]any{
		"code": s.processor.GetExampleCode(),
		"uri":  s.processor.GetExampleUri(),
	}

	s.processor.ProcessSource(eval, config)

	source := s.processor.GetSource(eval)
	s.Require().NotNil(source, "Source should be set")

	code, ok := s.processor.GetCodeFromSource(source)
	s.Require().True(ok, "Source should be code type when both present")
	s.Equal(s.processor.GetExampleCode(), code, "Code should be set correctly")
}

// TestNoSource tests behavior when no source is present
func (s *EvaluatorSourceTestSuite) TestNoSource() {
	eval := s.processor.CreateEvaluator()
	config := map[string]any{
		s.processor.GetIrrelevantConfigKey(): "30s",
	}

	s.processor.ProcessSource(eval, config)

	source := s.processor.GetSource(eval)
	s.Nil(source, "Source should remain nil when no source present")
}

// TestEmptyValues tests behavior with empty source values
func (s *EvaluatorSourceTestSuite) TestEmptyValues() {
	eval := s.processor.CreateEvaluator()
	config := map[string]any{
		"code": "",
		"uri":  "",
	}

	s.processor.ProcessSource(eval, config)

	source := s.processor.GetSource(eval)
	s.Nil(source, "Source should remain nil for empty values")
}

// TestStarlarkSourceProcessing tests Starlark evaluator source processing
func TestStarlarkSourceProcessing(t *testing.T) {
	suite.Run(t, &EvaluatorSourceTestSuite{
		processor: &StarlarkSourceProcessor{},
	})
}

// TestExtismSourceProcessing tests Extism evaluator source processing
func TestExtismSourceProcessing(t *testing.T) {
	suite.Run(t, &EvaluatorSourceTestSuite{
		processor: &ExtismSourceProcessor{},
	})
}

// TestRisorSourceProcessing tests Risor evaluator source processing
func TestRisorSourceProcessing(t *testing.T) {
	suite.Run(t, &EvaluatorSourceTestSuite{
		processor: &RisorSourceProcessor{},
	})
}
