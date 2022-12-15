package galois

import (
	"log"
	"math/big"
)

type gf256 struct {
}

// Irreducible Polynomial
// F(x) = x^256 + x^241 + x^178 + x^121 + 1
var P256 = []uint{256, 241, 178, 121}
var GFPRI = big.NewInt(1)
var GFMASK = big.NewInt(1)
var GFINV = big.NewInt(1)

const GF_SIZE = 256

func GF256() *gf256 {
	poly := gf256{}
	return &poly
}

func init() {
	GFMASK = GFMASK.Lsh(GFMASK, GF_SIZE)
	GFMASK = GFMASK.Sub(GFMASK, big.NewInt(1))

	GFINV = GFINV.Lsh(GFINV, GF_SIZE)
	GFINV = GFINV.Sub(GFINV, big.NewInt(2))

	for _, x := range P256 {
		p := big.NewInt(1)
		GFPRI = p.Add(p.Lsh(p, x), GFPRI)
	}
	log.Printf("Prime %v", GFPRI.BitLen())
}

func (table *gf256) Add256(x, y *big.Int) *big.Int {
	s := big.NewInt(0)
	return s.Xor(x, y)
}

func (table *gf256) Sub256(x, y *big.Int) *big.Int {
	s := big.NewInt(0)
	return s.Xor(x, y)
}

func (table *gf256) lsh256(x *big.Int) *big.Int {
	t := new(big.Int)
	t = t.Set(x)
	if x.BitLen() < GF_SIZE {
		t = t.Lsh(t, 1)
	} else {
		t = t.Lsh(t, 1)
		t = t.Xor(t, GFPRI)
		// t = t.And(t, GFMASK)
	}

	return t
}

func (table *gf256) Mul256(x, y *big.Int) *big.Int {
	r := big.NewInt(0)
	t := new(big.Int)
	t = t.Set(x)
	max := y.BitLen()

	for i := 0; i < max; i++ {
		if y.Bit(i) == 1 {
			r = r.Xor(r, t)
		}

		t = table.lsh256(t)
	}

	return r
}

// x**p = x, so x**(p-2) = x**(-1)
func (table *gf256) Div256(x, y *big.Int) *big.Int {
	if y.BitLen() == 0 {
		log.Printf("Div by zero")
		return big.NewInt(0)
	}

	inv := table.Exp256(y, GFINV)
	return table.Mul256(x, inv)
}

func (table *gf256) Exp256(x, y *big.Int) *big.Int {
	p := new(big.Int)
	p = p.Set(x)
	b := big.NewInt(1)
	max := y.BitLen()

	for i := 0; i < max; i++ {
		if y.Bit(i) != 0 {
			b = table.Mul256(b, p)
		}
		p = table.Mul256(p, p)
	}

	return b
}

// y/x = q*x + r
func (table *gf256) divid(x, y *big.Int) (*big.Int, *big.Int) {
	d := new(big.Int)
	d = d.Set(x)
	q := new(big.Int)
	q = q.Set(y)
	r := new(big.Int)
	// log.Printf("x : %x", x)
	nq := big.NewInt(0)

	for {
		lq := big.NewInt(1)
		dif := q.BitLen() - d.BitLen()
		if dif < 0 {
			break
		}
		r = r.Lsh(d, uint(dif))
		r = r.Xor(r, q)
		q = q.Set(r)
		nq = lq.Add(lq.Lsh(lq, uint(dif)), nq)
	}

	return nq, r
}

// Farmat's Little Theorem to calculate Inverse
func (table *gf256) InvF256(x *big.Int) *big.Int {
	// e := big.NewInt(0)
	// e = e.Lsh(big.NewInt(1), 256)
	// return table.Exp256(x, e.Sub(e, big.NewInt(2)))
	return table.Exp256(x, GFINV)
}

// Extended Euclidian Algorithm to calculate Inverse
func (table *gf256) Inv256(x *big.Int) *big.Int {
	b := new(big.Int)
	b = b.Set(x)
	a := new(big.Int)
	a = a.Set(GFPRI)
	t := big.NewInt(0)
	t1 := big.NewInt(0)
	t2 := big.NewInt(1)

	for b.BitLen() > 1 {
		q, r := table.divid(b, a)
		t = table.Sub256(t1, table.Mul256(q, t2))
		a.Set(b)
		b.Set(r)
		t1.Set(t2)
		t2.Set(t)
	}

	log.Printf("b : %x, x : %x, Inv : %x", b, x, t)
	return t
}
