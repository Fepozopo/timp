package stdimg

import (
	"image"
	"math"
	"math/rand"
	"runtime"
	"sort"
	"sync"
)

// AddNoise adds noise to src. typ may be "GAUSSIAN", "UNIFORM", or "POISSON".
// amount controls strength (stddev for gaussian, max deviation for uniform, scale for poisson).
// seed allows deterministic output for tests (seed==0 uses a fixed seed).
func AddNoise(src *image.NRGBA, typ string, amount float64, seed int64) *image.NRGBA {
	if src == nil {
		return nil
	}
	if amount <= 0 {
		return CloneNRGBA(src)
	}

	rng := rand.New(rand.NewSource(seed))
	if seed == 0 {
		// choose a deterministic non-zero seed for reproducibility when seed==0
		rng = rand.New(rand.NewSource(1))
	}

	w := src.Bounds().Dx()
	h := src.Bounds().Dy()
	out := image.NewNRGBA(src.Rect)
	typ = upper(typ)
	var cdfs [][]float64
	if typ == "POISSON" {
		cdfs = buildPoissonCDFs(amount)
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := src.PixOffset(x+src.Rect.Min.X, y+src.Rect.Min.Y)
			r := float64(src.Pix[i+0])
			g := float64(src.Pix[i+1])
			b := float64(src.Pix[i+2])
			a := float64(src.Pix[i+3])
			var nr, ng, nb float64
			switch typ {
			case "UNIFORM":
				deltaR := (rng.Float64()*2 - 1) * amount
				deltaG := (rng.Float64()*2 - 1) * amount
				deltaB := (rng.Float64()*2 - 1) * amount
				nr = r + deltaR
				ng = g + deltaG
				nb = b + deltaB
			case "POISSON":
				// vectorized sampling via precomputed CDF for each channel value
				chR := int(r)
				if chR < 0 {
					chR = 0
				}
				if chR > 255 {
					chR = 255
				}
				cdfR := cdfs[chR]
				u := rng.Float64()
				k := sort.SearchFloat64s(cdfR, u)
				sampleR := float64(k)
				nr = sampleR * (255.0 / amount)
				chG := int(g)
				if chG < 0 {
					chG = 0
				}
				if chG > 255 {
					chG = 255
				}
				cdfG := cdfs[chG]
				u2 := rng.Float64()
				k2 := sort.SearchFloat64s(cdfG, u2)
				sampleG := float64(k2)
				ng = sampleG * (255.0 / amount)
				chB := int(b)
				if chB < 0 {
					chB = 0
				}
				if chB > 255 {
					chB = 255
				}
				cdfB := cdfs[chB]
				u3 := rng.Float64()
				k3 := sort.SearchFloat64s(cdfB, u3)
				sampleB := float64(k3)
				nb = sampleB * (255.0 / amount)
			default:
				// GAUSSIAN
				nr = r + gaussianSample(rng, amount)
				ng = g + gaussianSample(rng, amount)
				nb = b + gaussianSample(rng, amount)
			}

			out.Pix[i+0] = uint8(clampFloatToUint8(nr))
			out.Pix[i+1] = uint8(clampFloatToUint8(ng))
			out.Pix[i+2] = uint8(clampFloatToUint8(nb))
			out.Pix[i+3] = uint8(clampFloatToUint8(a))
		}
	}
	return out
}

// gaussianSample returns a normal(0,std) sample using Box-Muller
func gaussianSample(rng *rand.Rand, std float64) float64 {
	if std <= 0 {
		return 0
	}
	u1 := rng.Float64()
	u2 := rng.Float64()
	z0 := math.Sqrt(-2*math.Log(u1)) * math.Cos(2*math.Pi*u2)
	return z0 * std
}

// buildPoissonCDFs precomputes Poisson CDF arrays for channel values 0..255.
// For each channel value ch we compute lambda = (ch/255)*amount and build the CDF
// over k=0..K such that cumulative probability approaches 1. Returns a slice
// of length 256 where each entry is the CDF slice for that channel.
func buildPoissonCDFs(amount float64) [][]float64 {
	cdfs := make([][]float64, 256)
	// parallel worker pool
	workers := runtime.GOMAXPROCS(0)
	if workers < 1 {
		workers = 1
	}
	jobs := make(chan int, 256)
	var wg sync.WaitGroup
	wg.Add(workers)
	for w := 0; w < workers; w++ {
		go func() {
			defer wg.Done()
			for ch := range jobs {
				lambda := (float64(ch) / 255.0) * amount
				if lambda <= 0 {
					cdfs[ch] = []float64{1.0}
					continue
				}
				// pump PMF until cumulative nearly 1
				cdf := make([]float64, 0, 32)
				p := math.Exp(-lambda) // p0
				cum := p
				cdf = append(cdf, cum)
				k := 1
				// upper bound heuristic
				upper := int(math.Ceil(lambda + 10*math.Sqrt(lambda) + 10))
				if upper < 32 {
					upper = 32
				}
				for cum < 1-1e-12 && k <= upper {
					p = p * lambda / float64(k)
					cum += p
					// guard
					if cum > 1 {
						cum = 1
					}
					cdf = append(cdf, cum)
					k++
				}
				cdfs[ch] = cdf
			}
		}()
	}
	for ch := 0; ch < 256; ch++ {
		jobs <- ch
	}
	close(jobs)
	wg.Wait()
	return cdfs
}

func upper(s string) string {
	// simple ASCII upper
	b := []byte(s)
	for i := range b {
		if b[i] >= 'a' && b[i] <= 'z' {
			b[i] = b[i] - 'a' + 'A'
		}
	}
	return string(b)
}
