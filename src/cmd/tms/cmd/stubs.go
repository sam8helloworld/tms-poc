package cmd

import (
	"fmt"
	"io"

	"github.com/sam8helloworld/tms-poc/internal/sourcing/domain/pricing"
)

// stubParserFactory: 開発用のTariffParserFactoryスタブ
type stubParserFactory struct{}

func (f *stubParserFactory) GetParser(format string) (pricing.TariffParser, error) {
	return nil, fmt.Errorf("[STUB] tariff parser not available for format: %s", format)
}

// stubValidator: 開発用のTariffDataValidatorスタブ
type stubValidator struct{}

func (v *stubValidator) Validate(data *pricing.ParsedTariffData) *pricing.ValidationResult {
	return &pricing.ValidationResult{IsValid: true}
}

// Verify interface compliance
var _ pricing.TariffParserFactory = (*stubParserFactory)(nil)
var _ pricing.TariffDataValidator = (*stubValidator)(nil)

// stubParser: TariffParserスタブ（GetParserから返されるが実際にはエラーになる）
type stubParser struct{}

func (p *stubParser) Parse(reader io.Reader) (*pricing.ParsedTariffData, error) {
	return nil, fmt.Errorf("[STUB] tariff parser not implemented")
}

func (p *stubParser) SupportedFormats() []string {
	return nil
}

var _ pricing.TariffParser = (*stubParser)(nil)
