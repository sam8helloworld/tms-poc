package shared

import "github.com/shopspring/decimal"

type Money struct {
	Amount   decimal.Decimal
	Currency string
}

// NewMoney: 通貨必須バリデーション付きファクトリ
func NewMoney(amount decimal.Decimal, currency string) (Money, error) {
	if currency == "" {
		return Money{}, NewDomainError(ErrInvalidArgument, "currency is required")
	}
	return Money{Amount: amount, Currency: currency}, nil
}

// ZeroMoney: 指定通貨のゼロ金額を生成
func ZeroMoney(currency string) Money {
	return Money{Amount: decimal.Zero, Currency: currency}
}

// Add: 通貨一致チェック付き加算
func (m Money) Add(other Money) (Money, error) {
	if m.Currency != other.Currency {
		return Money{}, NewDomainError(ErrCurrencyMismatch, "cannot add different currencies").
			WithDetail("left", m.Currency).
			WithDetail("right", other.Currency)
	}
	return Money{Amount: m.Amount.Add(other.Amount), Currency: m.Currency}, nil
}

// Sub: 通貨一致チェック付き減算
func (m Money) Sub(other Money) (Money, error) {
	if m.Currency != other.Currency {
		return Money{}, NewDomainError(ErrCurrencyMismatch, "cannot subtract different currencies").
			WithDetail("left", m.Currency).
			WithDetail("right", other.Currency)
	}
	return Money{Amount: m.Amount.Sub(other.Amount), Currency: m.Currency}, nil
}

// Multiply: スカラー乗算
func (m Money) Multiply(factor decimal.Decimal) Money {
	return Money{Amount: m.Amount.Mul(factor), Currency: m.Currency}
}

// IsZero: ゼロかどうか判定
func (m Money) IsZero() bool {
	return m.Amount.IsZero()
}

// IsPositive: 正の値かどうか判定
func (m Money) IsPositive() bool {
	return m.Amount.GreaterThan(decimal.Zero)
}

// IsNegative: 負の値かどうか判定
func (m Money) IsNegative() bool {
	return m.Amount.LessThan(decimal.Zero)
}

// GreaterThan: 他のMoneyより大きいか判定
func (m Money) GreaterThan(other Money) bool {
	return m.Amount.GreaterThan(other.Amount)
}

// Equals: 等価判定（金額と通貨が一致）
func (m Money) Equals(other Money) bool {
	return m.Currency == other.Currency && m.Amount.Equal(other.Amount)
}
