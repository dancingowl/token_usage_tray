package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math"
	"math/big"
	"strconv"
	"strings"
	"time"
)

// Credits are a separate balance, never a percentage of a usage window.
type CreditBalance struct {
	Balance   *float64  `json:"balance,omitempty"`
	Unlimited bool      `json:"unlimited,omitempty"`
	CheckedAt time.Time `json:"checked_at"`
	Account   string    `json:"account,omitempty"`
}

func creditAccountFingerprint(account string) string {
	if account == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(account))
	return hex.EncodeToString(sum[:])
}

func validCreditBalance(balance float64) bool {
	return balance >= 0 && !math.IsNaN(balance) && !math.IsInf(balance, 0)
}

// Keep the optional payload raw so a malformed credit field cannot reject quotas.
func parseCreditBalance(raw json.RawMessage, account string, checkedAt time.Time) *CreditBalance {
	c := &CreditBalance{CheckedAt: checkedAt, Account: creditAccountFingerprint(account)}
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil || fields == nil {
		return c
	}
	if value, ok := fields["unlimited"]; ok {
		if string(value) == "null" || json.Unmarshal(value, &c.Unlimited) != nil {
			return c
		}
	}
	if c.Unlimited {
		return c
	}
	value := fields["balance"]
	var balance float64
	if len(value) == 0 || string(value) == "null" {
		return c
	}
	if json.Unmarshal(value, &balance) != nil {
		var str string
		if json.Unmarshal(value, &str) != nil {
			return c
		}
		var err error
		balance, err = strconv.ParseFloat(strings.TrimSpace(str), 64)
		if err != nil {
			return c
		}
	}
	if validCreditBalance(balance) {
		c.Balance = &balance
	}
	return c
}

func creditBalanceFreshAt(c *CreditBalance, now time.Time) bool {
	return c != nil && !c.CheckedAt.IsZero() && !c.CheckedAt.After(now) && now.Sub(c.CheckedAt) <= historyFreshness
}

func creditBalanceTextAt(u AgentUsage, now time.Time) string {
	c := u.Credits
	if !creditBalanceFreshAt(c, now) {
		return "확인 불가"
	}
	if c.Unlimited {
		return "무제한"
	}
	if c.Balance == nil || !validCreditBalance(*c.Balance) {
		return "확인 불가"
	}
	return formatCreditBalance(*c.Balance) + " 크레딧"
}

func formatCreditBalance(balance float64) string {
	if !validCreditBalance(balance) {
		return ""
	}
	if balance == 0 {
		return "0.00"
	}
	// Match the browser's decimal half-up display instead of binary tie-to-even.
	// Integer cents also avoid overflowing a large finite balance by multiplying it.
	whole, fraction, _ := strings.Cut(strconv.FormatFloat(balance, 'f', -1, 64), ".")
	fraction += "000"
	cents, _ := new(big.Int).SetString(whole+fraction[:2], 10)
	if fraction[2] >= '5' {
		cents.Add(cents, big.NewInt(1))
	}
	digits := cents.String()
	for len(digits) < 3 {
		digits = "0" + digits
	}
	whole, fraction = digits[:len(digits)-2], digits[len(digits)-2:]
	for i := len(whole) - 3; i > 0; i -= 3 {
		whole = whole[:i] + "," + whole[i:]
	}
	return whole + "." + fraction
}
