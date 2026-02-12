package pricing

import (
	"fmt"

	"github.com/expr-lang/expr"
	"github.com/sam8helloworld/tms-poc/internal/shared"
	"github.com/shopspring/decimal"
)

// ExpressionStrategy: Stage 2 - Dynamic (Expression)
// "max(weight, volume*167) * rate" などの複雑な式に対応
// github.com/expr-lang/expr を使用した式評価
type ExpressionStrategy struct {
	Formula  string
	Currency string
}

func (s *ExpressionStrategy) Type() string { return "EXPRESSION" }

func (s *ExpressionStrategy) Calculate(ctx ShipmentContext) (shared.Money, error) {
	// ShipmentContextから式評価用の環境を構築
	env := buildExpressionEnv(ctx)

	// 式をコンパイル＆評価
	program, err := expr.Compile(s.Formula, expr.Env(env))
	if err != nil {
		return shared.Money{}, shared.NewDomainError(
			shared.ErrInvalidArgument,
			fmt.Sprintf("failed to compile formula: %s", err),
		)
	}

	output, err := expr.Run(program, env)
	if err != nil {
		return shared.Money{}, shared.NewDomainError(
			shared.ErrBusinessRuleViolation,
			fmt.Sprintf("failed to evaluate formula: %s", err),
		)
	}

	// 評価結果をdecimal.Decimalに変換
	amount, err := toDecimalFromOutput(output)
	if err != nil {
		return shared.Money{}, shared.NewDomainError(
			shared.ErrInvalidArgument,
			fmt.Sprintf("formula result is not a number: %s", err),
		)
	}

	// 負数は許容しない
	if amount.IsNegative() {
		amount = decimal.Zero
	}

	return shared.NewMoney(amount, s.Currency)
}

// buildExpressionEnv: ShipmentContextから式評価用の環境を構築
func buildExpressionEnv(ctx ShipmentContext) map[string]interface{} {
	env := make(map[string]interface{})

	// 基本的な数値フィールド（decimal.Decimalをfloat64に変換）
	env["quantity"] = decimalToFloat(ctx.Quantity)
	env["weight"] = decimalToFloat(ctx.WeightKG)
	env["volume"] = decimalToFloat(ctx.VolumeM3)

	// 課金重量（容積重量と実重量の大きい方）
	// ShipmentContextには直接含まれていないが、計算可能
	chargeableWeight := ctx.WeightKG
	volumetricWeight := ctx.VolumeM3.Mul(decimal.NewFromInt(1000)) // m3 * 1000 = kg
	if volumetricWeight.GreaterThan(chargeableWeight) {
		chargeableWeight = volumetricWeight
	}
	env["chargeable_weight"] = decimalToFloat(chargeableWeight)

	// 動的属性をマージ（文字列属性は数値変換を試みる）
	for key, val := range ctx.Attributes {
		// 文字列を数値に変換できる場合は数値として格納
		if strVal, ok := val.(string); ok {
			if numVal, err := decimal.NewFromString(strVal); err == nil {
				env[key] = decimalToFloat(numVal)
			} else {
				env[key] = strVal
			}
		} else {
			env[key] = val
		}
	}

	return env
}

// decimalToFloat: decimal.Decimalをfloat64に変換
func decimalToFloat(d decimal.Decimal) float64 {
	f, _ := d.Float64()
	return f
}

// toDecimalFromOutput: expr評価結果をdecimal.Decimalに変換
func toDecimalFromOutput(output interface{}) (decimal.Decimal, error) {
	switch v := output.(type) {
	case int:
		return decimal.NewFromInt(int64(v)), nil
	case int32:
		return decimal.NewFromInt(int64(v)), nil
	case int64:
		return decimal.NewFromInt(v), nil
	case float32:
		return decimal.NewFromFloat(float64(v)), nil
	case float64:
		return decimal.NewFromFloat(v), nil
	case string:
		return decimal.NewFromString(v)
	default:
		return decimal.Zero, fmt.Errorf("unsupported type: %T", v)
	}
}
