package appcore

import (
	"fmt"
	"math"
	"math/big"
	"strconv"
)

// Convert the decimal values received over JSON, then round once, half up, to
// four decimal places. Binary float multiplication can misround decimal ties
// and the usual round(x*10000) can overflow even when x itself is finite.
func convertedAmount(amount, mid float64, multiply bool) (float64, error) {
	a, ok := new(big.Rat).SetString(strconv.FormatFloat(amount, 'f', -1, 64))
	if !ok || a.Sign() < 0 {
		return 0, fmt.Errorf("niepoprawna kwota")
	}
	r, ok := new(big.Rat).SetString(strconv.FormatFloat(mid, 'f', -1, 64))
	if !ok || r.Sign() <= 0 {
		return 0, fmt.Errorf("niepoprawny kurs")
	}
	if multiply {
		a.Mul(a, r)
	} else {
		a.Quo(a, r)
	}
	a.Mul(a, big.NewRat(currencyPrecision, 1))
	units, remainder := new(big.Int), new(big.Int)
	units.QuoRem(a.Num(), a.Denom(), remainder)
	if remainder.Lsh(remainder, 1).Cmp(a.Denom()) >= 0 {
		units.Add(units, big.NewInt(1))
	}
	result, _ := new(big.Rat).SetFrac(units, big.NewInt(currencyPrecision)).Float64()
	if math.IsInf(result, 0) || math.IsNaN(result) {
		return 0, fmt.Errorf("wynik przekracza obsługiwany zakres liczbowy; zmniejsz kwotę")
	}
	return result, nil
}
