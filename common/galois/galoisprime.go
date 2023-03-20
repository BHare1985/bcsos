package galois

import (
	"log"
	"math/big"

	"github.com/holiman/uint256"
)

const ALIGN31 = 31
const ALIGN32 = 32
const ALIGN08 = 8

type GFP struct {
	Size  uint
	Prime *uint256.Int
}

// PRIME = 2^256 - 351*2^32 + 1

func NewGFP() *GFP {
	p := big.NewInt(1)
	p1 := big.NewInt(1)
	p1.Lsh(p1, 32)
	p1.Mul(p1, big.NewInt(351))
	p.Lsh(p, 256)
	p.Sub(p, p1)
	p.Add(p, big.NewInt(1))
	size := p.BitLen()
	// p.Set(big.NewInt(257))

	// log.Printf("Prime : %x", p)
	prime, _ := uint256.FromBig(p)

	gf := GFP{uint(size), prime}
	return &gf
}

func (gf *GFP) Cmp(x, y *uint256.Int) int {
	return x.Cmp(y)
}

func (gf *GFP) IntFromBytes(x []byte) *uint256.Int {
	s := new(uint256.Int)
	s.SetBytes(x)
	return s.Mod(s, gf.Prime)
}

func (gf *GFP) Add(x, y *uint256.Int) *uint256.Int {
	s := new(uint256.Int)
	return s.AddMod(x, y, gf.Prime)
}

func (gf *GFP) Sub(x, y *uint256.Int) *uint256.Int {
	s := new(uint256.Int)
	if x.Lt(y) {
		s.Sub(y, x)
		return s.Sub(gf.Prime, s)
	} else {
		return s.Sub(x, y)
	}
}

func (gf *GFP) Mul(x, y *uint256.Int) *uint256.Int {
	s := new(uint256.Int)
	return s.MulMod(x, y, gf.Prime)
}
func (gf *GFP) Mul2(x, y *uint256.Int) *uint256.Int {
	s := x.ToBig()
	s.Mul(s, y.ToBig())
	s.Mod(s, gf.Prime.ToBig())
	o, _ := uint256.FromBig(s)
	return o
}

func (gf *GFP) Div(x, y *uint256.Int) *uint256.Int {
	return gf.Mul(x, gf.Inv(y))
}

func (gf *GFP) OrOne(x *uint256.Int) *uint256.Int {
	s := new(uint256.Int)
	return s.Or(x, uint256.NewInt(1))
}

func (gf *GFP) Exp(x, y *uint256.Int) *uint256.Int {
	a := x.ToBig()
	b := y.ToBig()
	m := gf.Prime.ToBig()
	a.Exp(a, b, m)
	s, _ := uint256.FromBig(a)
	return s
}

// Extended Euclidian Algorithm to calculate Inverse
func (gf *GFP) InvE(x *uint256.Int) *uint256.Int {
	if x.BitLen() == 0 {
		return uint256.NewInt(0)
	}
	lm := uint256.NewInt(1)
	hm := uint256.NewInt(0)
	low := x.Clone()
	high := gf.Prime.Clone()
	r := high.Clone()

	for low.Gt(uint256.NewInt(1)) {
		r.Div(high, low)
		nm := gf.Sub(hm, gf.Mul(lm, r))
		nw := gf.Sub(high, gf.Mul(low, r))
		lm, low, hm, high = nm, nw, lm, low
	}

	return lm.Mod(lm, gf.Prime)
}

func (gf *GFP) Inv(x *uint256.Int) *uint256.Int {
	if x.BitLen() == 0 {
		return uint256.NewInt(0)
	}
	inv := gf.Prime.Clone()
	inv.Sub(inv, uint256.NewInt(2))
	return gf.Exp(x, inv)
}

func (gf *GFP) MultInv(xs []*uint256.Int) []*uint256.Int {
	output := make([]*uint256.Int, len(xs)+1)
	output[0] = uint256.NewInt(1)

	for i := 1; i < len(output); i++ {
		if xs[i-1].IsZero() {
			output[i] = output[i-1]
		} else {
			output[i] = gf.Mul(output[i-1], xs[i-1])
		}
	}

	inv := gf.Inv(output[len(output)-1])
	for i := len(xs); 0 < i; i-- {
		if xs[i-1].IsZero() {
			output[i-1] = uint256.NewInt(0)
		} else {
			output[i-1] = gf.Mul(output[i-1], inv)
		}
		if !xs[i-1].IsZero() {
			inv = gf.Mul(inv, xs[i-1])
		}
	}
	return output[:len(xs)]
}

// Evaluate Polynomial at a point x
func (gf *GFP) EvalPolyAt(cs []*uint256.Int, x *uint256.Int) *uint256.Int {
	y := uint256.NewInt(0)
	pox := uint256.NewInt(1)

	for _, c := range cs {
		y = gf.Add(gf.Mul(c, pox), y)
		pox = gf.Mul(pox, x)
	}

	return y
}

func (gf *GFP) ZPoly(xs []*uint256.Int) []*uint256.Int {
	cs := make([]*uint256.Int, 1, len(xs)+1)
	// cs = append(cs, uint256.NewInt(1))
	cs[0] = uint256.NewInt(1)
	for j, x := range xs {
		cs = append([]*uint256.Int{uint256.NewInt(0)}, cs...)
		for i := 0; i < j+1; i++ {
			t := gf.Mul(cs[i+1], x)
			cs[i] = gf.Sub(cs[i], t)
		}
	}

	return cs
}

// div polys
// D(x) = (x-1)(x-2)(x-3)....(x-n)/(x-k)
func (gf *GFP) DivPolys(a, b []*uint256.Int) []*uint256.Int {
	if len(a) < len(b) {
		return nil
	}

	var out []*uint256.Int
	cs := make([]*uint256.Int, len(a))
	for i := 0; i < len(a); i++ {
		cs[i] = a[i]
	}

	apos := len(cs) - 1
	bpos := len(b) - 1
	diff := apos - bpos

	for diff >= 0 {
		qout := gf.Div(cs[apos], b[bpos])
		out = append([]*uint256.Int{qout}, out...)
		for i := bpos; i >= 0; i-- {
			cs[diff+i] = gf.Sub(cs[diff+i], gf.Mul(b[i], qout))
		}
		apos -= 1
		diff -= 1
	}
	return out
}

func (gf *GFP) MulPolys(a, b []*uint256.Int) []*uint256.Int {
	out := make([]*uint256.Int, len(a)+len(b)-1)
	for i := 0; i < len(out); i++ {
		out[i] = uint256.NewInt(0)
	}

	for i := 0; i < len(a); i++ {
		for j := 0; j < len(b); j++ {
			out[i+j] = gf.Add(out[i+j], gf.Mul(a[i], b[j]))
		}
	}

	return out
}

func (gf *GFP) LagrangeInterp(xs, ys []*uint256.Int) []*uint256.Int {
	zp := gf.ZPoly(xs)
	if len(zp) != len(ys)+1 {
		return nil
	}

	lp := make([]*uint256.Int, len(ys))
	for i := 0; i < len(ys); i++ {
		lp[i] = uint256.NewInt(0)
	}

	for i, x := range xs {
		ps := make([]*uint256.Int, 2)
		ps[0], ps[1] = gf.Sub(gf.Prime, x), uint256.NewInt(1)

		// Get divid polynomial
		// dp = (x-x1)(x-x2).....(x-xn) / (x-xk)
		dp := gf.DivPolys(zp, ps)
		// dps = append([]*uint256.Int{dp}, dps...)
		// Evaluate each divided polynomial
		// denom = (xk-x1)(xk-x2)....(xk-xn)  without (xk-xk)
		denom := gf.EvalPolyAt(dp, x)
		// invdenom = 1/denom
		invdenom := gf.Inv(denom)
		// yk = yk * 1/denom
		yk := gf.Mul(ys[i], invdenom)
		// Add all coeficient of each x^n
		for j := range ys {
			lp[j] = gf.Add(lp[j], gf.Mul(dp[j], yk))
		}
	}

	return lp
}

func (gf *GFP) LagrangeInterp_4(xs, ys []*uint256.Int) []*uint256.Int {
	x01 := gf.Mul(xs[0], xs[1])
	x02 := gf.Mul(xs[0], xs[2])
	x03 := gf.Mul(xs[0], xs[3])
	x12 := gf.Mul(xs[1], xs[2])
	x13 := gf.Mul(xs[1], xs[3])
	x23 := gf.Mul(xs[2], xs[3])
	m := gf.Prime

	add3 := func(a, b, c *uint256.Int) *uint256.Int {
		return gf.Add(gf.Add(a, b), c)
	}

	eq0 := []*uint256.Int{gf.Mul(gf.Sub(m, x12), xs[3]), add3(x12, x13, x23), gf.Sub(m, add3(xs[1], xs[2], xs[3])), uint256.NewInt(1)}
	eq1 := []*uint256.Int{gf.Mul(gf.Sub(m, x02), xs[3]), add3(x02, x03, x23), gf.Sub(m, add3(xs[0], xs[2], xs[3])), uint256.NewInt(1)}
	eq2 := []*uint256.Int{gf.Mul(gf.Sub(m, x01), xs[3]), add3(x01, x03, x13), gf.Sub(m, add3(xs[0], xs[1], xs[3])), uint256.NewInt(1)}
	eq3 := []*uint256.Int{gf.Mul(gf.Sub(m, x01), xs[2]), add3(x01, x02, x12), gf.Sub(m, add3(xs[0], xs[1], xs[2])), uint256.NewInt(1)}

	eval_deg4 := func(p []*uint256.Int, x *uint256.Int) *uint256.Int {
		x2 := gf.Mul(x, x)
		x3 := gf.Mul(x2, x)

		o := gf.Add(p[0], gf.Mul(p[1], x))
		o = gf.Add(o, gf.Mul(p[2], x2))
		o = gf.Add(o, gf.Mul(p[3], x3))
		return o
	}

	e0 := eval_deg4(eq0, xs[0])
	e1 := eval_deg4(eq1, xs[1])
	e2 := eval_deg4(eq2, xs[2])
	e3 := eval_deg4(eq3, xs[3])
	e01 := gf.Mul(e0, e1)
	e23 := gf.Mul(e2, e3)
	invall := gf.Inv(gf.Mul(e01, e23))
	inv_y0 := gf.Mul(gf.Mul(gf.Mul(ys[0], invall), e1), e23)
	inv_y1 := gf.Mul(gf.Mul(gf.Mul(ys[1], invall), e0), e23)
	inv_y2 := gf.Mul(gf.Mul(gf.Mul(ys[2], invall), e01), e3)
	inv_y3 := gf.Mul(gf.Mul(gf.Mul(ys[3], invall), e01), e2)

	lp := make([]*uint256.Int, len(ys))
	for i := 0; i < len(ys); i++ {
		lp[i] = gf.Mul(eq0[i], inv_y0)
		lp[i] = gf.Add(lp[i], gf.Mul(eq1[i], inv_y1))
		lp[i] = gf.Add(lp[i], gf.Mul(eq2[i], inv_y2))
		lp[i] = gf.Add(lp[i], gf.Mul(eq3[i], inv_y3))
	}

	return lp
}

func (gf *GFP) ExtRootUnity2(x *uint256.Int, inv bool) (int, []*uint256.Int) {
	maxc := 65537
	if gf.Size < 16 {
		maxc = int(1<<gf.Size) + 1
	}
	var cmpos int

	roots := make([]*uint256.Int, maxc)
	if inv {
		roots[maxc-1] = uint256.NewInt(1)
		roots[maxc-2] = x
		cmpos = maxc - 2
	} else {
		roots[0] = uint256.NewInt(1)
		roots[1] = x
		cmpos = 1
	}

	one := uint256.NewInt(1)
	i := 2
	for ; one.Cmp(roots[cmpos]) != 0; i++ {
		if i < maxc {
			if inv {
				roots[maxc-i-1] = gf.Mul(x, roots[cmpos])
			} else {
				roots[i] = gf.Mul(x, roots[cmpos])
			}
		} else {
			return -1, roots
		}
		if !inv {
			cmpos = i
		} else {
			cmpos = maxc - i - 1
		}
	}

	if inv {
		return i, roots[maxc-i:]
	} else {
		// return i - 1, roots[:len(roots)-1]
		return i, roots
	}

}

func (gf *GFP) ExtRootUnity(root *uint256.Int, inv bool) (int, []*uint256.Int) {
	maxc := 65536*1 + 1
	if gf.Size < 16 {
		maxc = int(1<<gf.Size) + 1
	}

	x := new(uint256.Int)
	if inv {
		x.Set(gf.Inv(root))
	} else {
		x.Set(root)
	}

	roots := make([]*uint256.Int, maxc)
	roots[0] = uint256.NewInt(1)
	roots[1] = x

	one := uint256.NewInt(1)
	i := 2
	for ; one.Cmp(roots[i-1]) != 0; i++ {
		if i < maxc {
			roots[i] = gf.Mul(x, roots[i-1])
		} else {
			return -1, roots
		}
	}
	// return i - 1, roots[:i-1]
	return i, roots[:i]
}

// FFT algorithm with root of unity
// xs should be root of unit : x^n = 1
func (gf *GFP) fft_org(xs, ys []*uint256.Int, w *uint256.Int) []*uint256.Int {
	l := len(xs)
	os := make([]*uint256.Int, l) // outputs

	for i := 0; i < l; i++ {
		sum := uint256.NewInt(0)
		for j := 0; j < len(ys); j++ {
			m := gf.Mul(ys[j], xs[(i*j)%l])
			sum = gf.Add(sum, m)
		}
		os[i] = gf.Mul(sum, w)
	}
	return os
}

func (gf *GFP) _fft(xs, ys []*uint256.Int, w *uint256.Int) []*uint256.Int {
	l := len(xs)
	os := make([]*uint256.Int, l) // outputs

	for i := 0; i < l; i++ {
		sum := uint256.NewInt(0)
		for j := 0; j < len(ys); j++ {
			m := gf.Mul(ys[j], xs[(i*j)%l])
			sum = gf.Add(sum, m)
		}
		os[i] = gf.Mul(sum, w)
	}
	return os
}
func (gf *GFP) fft(xs, ys []*uint256.Int, w *uint256.Int) []*uint256.Int {
	if len(ys) <= 4 {
		return gf._fft(xs, ys, w)
	}

	exs := make([]*uint256.Int, (len(ys)+1)>>1)
	eys := make([]*uint256.Int, (len(ys)+1)>>1)
	oys := make([]*uint256.Int, (len(ys)+1)>>1)

	for i := 0; i < len(ys); i++ {
		if i%2 == 0 {
			exs[i>>1] = xs[i]
			eys[i>>1] = ys[i]
		} else {
			oys[i>>1] = ys[i]
		}
	}

	L := gf.fft(exs, eys, w)
	R := gf.fft(exs, oys, w)

	os := make([]*uint256.Int, len(ys))
	for i := 0; i < len(L); i++ {
		yt := gf.Mul(R[i], xs[i])
		os[i] = gf.Add(L[i], yt)
		os[i+len(L)] = gf.Sub(L[i], yt)
	}
	return os
}

// DFT evaluates a polynomial at xs(root of unity)
// cs is coefficients of a polynominal : [c0, c1, c2, c3 ... cn-1]
// xs is root of unity, so x^n = 1 : [x^0, x^1, x^2, .... x^n-1]
func (gf *GFP) DFT(cs []*uint256.Int, root *uint256.Int) []*uint256.Int {
	size, xs := gf.ExtRootUnity(root, false)
	if size == -1 {
		log.Printf("Wrong root of unity !!!")
		return nil
	}
	w := uint256.NewInt(1) // No inverse
	vs := make([]*uint256.Int, size-1)
	// for i := 0; i < len(cs); i++ {
	// 	vs[i] = cs[i]
	// }
	copy(vs, cs)
	for i := len(cs); i < len(vs); i++ {
		vs[i] = uint256.NewInt(0)
	}
	return gf.fft(xs[:size-1], vs, w)
}

// IDFT generates a polynomial with points [(x0, y0), (x1, y1)....(xn-1, yn-1)]
// xs is root of unity, so x^n = 1 : [x^0, x^1, x^2, .... x^n-1]
// ys : [y0, y1, y2, y3 ... yn-1]
// Output is coefficients of a polynominal : [c0, c1, c2, c3 ... cn-1]
func (gf *GFP) IDFT(ys []*uint256.Int, root *uint256.Int) []*uint256.Int {
	size, xs := gf.ExtRootUnity(root, true)
	if size != len(ys)+1 {
		log.Printf("The length of xs and ys should be the same : %v, %v", size, len(ys))
		return nil
	}

	w := gf.Inv(uint256.NewInt(uint64(size - 1))) // Divid by inverse
	return gf.fft(xs[:size-1], ys, w)
}

func (gf *GFP) loadUint256FromStream(s []byte, align int) []*uint256.Int {
	ls := len(s)
	lpad := ls % align
	if lpad != 0 {
		for i := 0; i < align-lpad; i++ { // Add Padding
			s = append(s, 0x0)
		}
	}
	ls = len(s) / align

	visu := make([]*uint256.Int, ls)
	for i := 0; i < ls; i++ {
		x := s[i*align : i*align+align]
		visu[i] = uint256.NewInt(0)
		visu[i].SetBytes(x)
	}

	return visu
}

func (gf *GFP) LoadUint256FromStream31(s []byte) []*uint256.Int {
	return gf.loadUint256FromStream(s, ALIGN31)
}

func (gf *GFP) LoadUint256FromStream32(s []byte) []*uint256.Int {
	return gf.loadUint256FromStream(s, ALIGN32)
}

func (gf *GFP) LoadUint256FromKey(key []byte) []*uint256.Int {
	return gf.loadUint256FromStream(key, ALIGN08)
}
