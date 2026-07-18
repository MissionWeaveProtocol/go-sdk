package missionweaveprotocol

import (
	"errors"
	"math/big"
)

var (
	ed25519Field = new(big.Int).Sub(
		new(big.Int).Lsh(big.NewInt(1), 255),
		big.NewInt(19),
	)
	ed25519Order, _ = new(big.Int).SetString(
		"1000000000000000000000000000000014def9dea2f79cd65812631a5cf5d3ed",
		16,
	)
	ed25519D = func() *big.Int {
		inverse := new(big.Int).ModInverse(big.NewInt(121666), ed25519Field)
		value := new(big.Int).Mul(big.NewInt(-121665), inverse)
		return value.Mod(value, ed25519Field)
	}()
	ed25519SqrtM1 = func() *big.Int {
		exponent := new(big.Int).Sub(ed25519Field, big.NewInt(1))
		exponent.Div(exponent, big.NewInt(4))
		return new(big.Int).Exp(big.NewInt(2), exponent, ed25519Field)
	}()
)

type ed25519Point struct {
	x *big.Int
	y *big.Int
	z *big.Int
	t *big.Int
}

func strictEd25519Point(encoded []byte, allowIdentity bool) error {
	if len(encoded) != 32 {
		return errors.New("does not encode a 32-byte Ed25519 point")
	}
	compressed := append([]byte(nil), encoded...)
	xSign := compressed[31] >> 7
	compressed[31] &= 0x7f
	y := littleEndianInteger(compressed)
	if y.Cmp(ed25519Field) >= 0 {
		return errors.New("is not a canonical Ed25519 point encoding")
	}

	ySquared := fieldMultiply(y, y)
	numerator := fieldSubtract(ySquared, big.NewInt(1))
	denominator := fieldAdd(fieldMultiply(ed25519D, ySquared), big.NewInt(1))
	denominatorInverse := new(big.Int).ModInverse(denominator, ed25519Field)
	if denominatorInverse == nil {
		return errors.New("does not decode to an Edwards25519 point")
	}
	xSquared := fieldMultiply(numerator, denominatorInverse)
	exponent := new(big.Int).Add(ed25519Field, big.NewInt(3))
	exponent.Div(exponent, big.NewInt(8))
	x := new(big.Int).Exp(xSquared, exponent, ed25519Field)
	if fieldSubtract(fieldMultiply(x, x), xSquared).Sign() != 0 {
		x = fieldMultiply(x, ed25519SqrtM1)
	}
	if fieldSubtract(fieldMultiply(x, x), xSquared).Sign() != 0 {
		return errors.New("does not decode to an Edwards25519 point")
	}
	if x.Sign() == 0 && xSign == 1 {
		return errors.New("uses a noncanonical negative-zero encoding")
	}
	if byte(x.Bit(0)) != xSign {
		x = fieldSubtract(ed25519Field, x)
	}
	point := ed25519Point{
		x: x,
		y: y,
		z: big.NewInt(1),
		t: fieldMultiply(x, y),
	}
	identity := ed25519PointIsIdentity(point)
	if identity && !allowIdentity {
		return errors.New("encodes the Ed25519 identity point")
	}
	if !ed25519PointIsIdentity(ed25519ScalarMultiply(point, ed25519Order)) {
		return errors.New("is not in the prime-order Ed25519 subgroup")
	}
	return nil
}

func ed25519PointAdd(left, right ed25519Point) ed25519Point {
	a := fieldMultiply(fieldSubtract(left.y, left.x), fieldSubtract(right.y, right.x))
	b := fieldMultiply(fieldAdd(left.y, left.x), fieldAdd(right.y, right.x))
	c := fieldMultiply(big.NewInt(2), fieldMultiply(ed25519D, fieldMultiply(left.t, right.t)))
	d := fieldMultiply(big.NewInt(2), fieldMultiply(left.z, right.z))
	e := fieldSubtract(b, a)
	f := fieldSubtract(d, c)
	g := fieldAdd(d, c)
	h := fieldAdd(b, a)
	return ed25519Point{
		x: fieldMultiply(e, f),
		y: fieldMultiply(g, h),
		z: fieldMultiply(f, g),
		t: fieldMultiply(e, h),
	}
}

func ed25519ScalarMultiply(point ed25519Point, scalar *big.Int) ed25519Point {
	result := ed25519Point{x: big.NewInt(0), y: big.NewInt(1), z: big.NewInt(1), t: big.NewInt(0)}
	addend := point
	for bit := 0; bit < scalar.BitLen(); bit++ {
		if scalar.Bit(bit) == 1 {
			result = ed25519PointAdd(result, addend)
		}
		addend = ed25519PointAdd(addend, addend)
	}
	return result
}

func ed25519PointIsIdentity(point ed25519Point) bool {
	return new(big.Int).Mod(new(big.Int).Set(point.x), ed25519Field).Sign() == 0 &&
		fieldSubtract(point.y, point.z).Sign() == 0
}

func fieldAdd(left, right *big.Int) *big.Int {
	value := new(big.Int).Add(left, right)
	return value.Mod(value, ed25519Field)
}

func fieldSubtract(left, right *big.Int) *big.Int {
	value := new(big.Int).Sub(left, right)
	return value.Mod(value, ed25519Field)
}

func fieldMultiply(left, right *big.Int) *big.Int {
	value := new(big.Int).Mul(left, right)
	return value.Mod(value, ed25519Field)
}

func littleEndianInteger(encoded []byte) *big.Int {
	reversed := make([]byte, len(encoded))
	for index := range encoded {
		reversed[len(encoded)-1-index] = encoded[index]
	}
	return new(big.Int).SetBytes(reversed)
}
