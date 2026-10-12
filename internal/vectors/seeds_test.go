package vectors

import (
	"math"
	"math/rand"
	"testing"
)

// pvSeedYears bounds the years TestSeedsDistinct covers; every vector
// has fewer (TestSeedsCoverVectors).
const pvSeedYears = 1024

// pvEffectiveSeed is the seed math/rand actually uses: the seed modulo
// 2^31-1, with 0 replaced (math/rand rngSource.Seed).
func pvEffectiveSeed(s int64) int64 {
	s %= math.MaxInt32
	if s < 0 {
		s += math.MaxInt32
	}
	if s == 0 {
		s = 89482311
	}
	return s
}

// TestSeedsDistinct: no two (year, seed variant) pairs share a stream
// over years 0..pvSeedYears-1 and every variant.
func TestSeedsDistinct(t *testing.T) {
	seen := map[int64][2]int{}
	for y := 0; y < pvSeedYears; y++ {
		for k := 0; k < Seeds; k++ {
			s := pvEffectiveSeed(pvSeed(y, k))
			if o, ok := seen[s]; ok {
				t.Fatalf("year %d variant %d shares a seed with year %d variant %d", y, k, o[0], o[1])
			}
			seen[s] = [2]int{y, k}
		}
	}
}

// TestRunRandsDistinct: the sources runVectorWith draws from (pvRun.rand)
// differ in their first draws for every (year, seed variant) pair of the
// first 64 years, so no year and variant reuses another's stream.
func TestRunRandsDistinct(t *testing.T) {
	seen := map[[4]int][2]int{}
	for y := 0; y < 64; y++ {
		for k := 0; k < Seeds; k++ {
			r := pvRun{seed: k}.rand(y)
			var d [4]int
			for i := range d {
				d[i] = r.Intn(1 << 30)
			}
			if o, ok := seen[d]; ok {
				t.Fatalf("year %d variant %d draws as year %d variant %d", y, k, o[0], o[1])
			}
			seen[d] = [2]int{y, k}
		}
	}
}

// TestSeedsCoverVectors: every vector's years are within the range
// TestSeedsDistinct covers.
func TestSeedsCoverVectors(t *testing.T) {
	vs, err := ReadDir("../../engine/testdata/vectors")
	if err != nil {
		t.Fatal(err)
	}
	if len(vs) == 0 {
		t.Fatal("no vectors")
	}
	for _, v := range vs {
		if v.Years >= pvSeedYears {
			t.Errorf("%s: %d years, TestSeedsDistinct covers %d", v.ID, v.Years, pvSeedYears)
		}
	}
}

// TestSeedVariantsIndependent: within a year, the seed variants' streams
// show no pairwise dependence at matching draw positions, early and
// deep: a chi-square test on rand(200) in 10x10 bins and a correlation,
// for every pair of variants. The limits are far above chance (81
// degrees of freedom: p < 1e-6 above 170; |r| above 6/sqrt(n)), so a
// failure means a real dependence, not an unlucky draw.
func TestSeedVariantsIndependent(t *testing.T) {
	if testing.Short() {
		t.Skip("streams of 20000 draws")
	}
	const n = 20000
	for _, y := range []int{1, 2, 3, 6, 29} {
		var v [Seeds][]int
		for k := range v {
			r := rand.New(rand.NewSource(pvSeed(y, k)))
			v[k] = make([]int, n)
			for i := range v[k] {
				v[k][i] = r.Intn(200)
			}
		}
		for _, w := range [][2]int{{0, 2000}, {2000, n}} {
			m := float64(w[1] - w[0])
			for a := 0; a < Seeds; a++ {
				for b := a + 1; b < Seeds; b++ {
					var tab [10][10]float64
					var sx, sy, sxx, syy, sxy float64
					for i := w[0]; i < w[1]; i++ {
						x, z := float64(v[a][i]), float64(v[b][i])
						tab[v[a][i]/20][v[b][i]/20]++
						sx, sy, sxx, syy, sxy = sx+x, sy+z, sxx+x*x, syy+z*z, sxy+x*z
					}
					chi, e := 0.0, m/100
					for i := range tab {
						for j := range tab[i] {
							d := tab[i][j] - e
							chi += d * d / e
						}
					}
					r := (m*sxy - sx*sy) / math.Sqrt((m*sxx-sx*sx)*(m*syy-sy*sy))
					if chi > 170 || math.Abs(r) > 6/math.Sqrt(m) {
						t.Errorf("year %d variants %d, %d, draws %d..%d: chi2 %.1f, r %.4f", y, a, b, w[0], w[1], chi, r)
					}
				}
			}
		}
	}
}
