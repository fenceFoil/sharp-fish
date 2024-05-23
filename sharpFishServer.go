package main

import (
	"crypto/sha1"
	_ "embed"
	"encoding/binary"
	"encoding/json"
	"log"
	"math"
	"math/rand/v2"
	"net/http"
	"os"
	"strings"
	"text/template"
)

//go:embed fishSvgTemplate.go.tmpl
var fishSvgTemplate string

func coordsToString(coords []float64) string {
	jsonArray, err := json.Marshal(coords)
	if err != nil {
		panic(err)
	}
	return strings.Trim(string(jsonArray), "[]")
}

type FishPoints struct {
	MainHue     float64
	AccentHue   float64
	BodyCoords  string
	MouthCoords string
	EyeCoords   string
	TailCoords  string
}

func fishDiamondPoints(centerX, centerY, width, height float64, bellyUpRatio float64) string {
	return coordsToString([]float64{
		centerX + width/2, centerY,
		centerX, centerY - height/2,
		centerX - width/2, centerY,
		centerX, centerY + (height / 2 / bellyUpRatio),
	})
}

func generateFish(rand *rand.Rand) FishPoints {
	fish := FishPoints{
		MainHue:   float64(rand.Int32N(360)),
		AccentHue: float64(rand.Int32N(360)),
	}

	if math.Abs(fish.AccentHue-fish.MainHue) < 5 {
		fish.AccentHue = float64(int(fish.AccentHue+10) % 360)
	}

	// Constants
	centerX := 300.0
	centerY := 300.0
	fishLength := 300.0

	// Body stats: heightRatio, bellyUpRatio
	heightRatio := rand.Float64()*1.2 + 0.5
	bellyUpRatio := 0.0
	if rand.Float64() > 0.1 {
		bellyUpRatio = rand.Float64()*0.8 + 0.7
	} else {
		bellyUpRatio = rand.Float64()*4.5 + 0.5
	}

	// Really really really tall fish look broken
	if heightRatio/bellyUpRatio > 2.0 {
		// Cap belly up ratio
		bellyUpRatio = heightRatio / 2
	}

	// Mouth stats: mouthSizeRatio, mouthOpenRatio
	mouthSizeRatio := rand.Float64() * 0.75
	if rand.Float64() > 0.5 {
		mouthSizeRatio = rand.Float64() * 0.3
	}
	// Little mouths can open real wide
	mouthOpenRatio := (rand.Float64()*0.9 + 0.1) * (1 / mouthSizeRatio)

	// Eye stats: eyeSize
	eyeSize := rand.Float64()*0.4 + 1.1

	// Tail stats: tailConcavity, tailInsetRatio, tailHeightRatio, tailLengthRatio
	tailConcavity := rand.Float64() * 0.7
	tailInsetRatio := rand.Float64()*0.3 + 0.05
	tailHeightRatio := rand.Float64()*0.95 + 0.05
	tailLengthRatio := rand.Float64()*1 + 0.5

	fishHeight := fishLength * heightRatio
	fish.BodyCoords = fishDiamondPoints(centerX, centerY, fishLength, fishHeight, bellyUpRatio)

	mouthHeight := fishLength * mouthSizeRatio * mouthOpenRatio
	mouthWidth := fishLength * mouthSizeRatio
	fish.MouthCoords = coordsToString([]float64{
		centerX - fishLength/2 - 1, centerY,
		centerX - fishLength/2, centerY - mouthHeight/2,
		centerX - fishLength/2 + mouthWidth, centerY,
		centerX - fishLength/2, centerY + mouthHeight/2,
	})

	fish.EyeCoords = fishDiamondPoints(centerX-fishLength*0.18, centerY-fishHeight*0.2, fishLength/10*eyeSize, fishLength/10*eyeSize, 1)

	fish.TailCoords = coordsToString([]float64{
		centerX + fishLength/2 - (tailInsetRatio * fishLength), centerY,
		centerX + fishLength/2 - (tailInsetRatio * fishLength) + (fishLength / 2 * tailLengthRatio), centerY - (fishHeight / 2 * tailHeightRatio),
		centerX + fishLength/2 - (tailInsetRatio * fishLength) + (fishLength/2*tailLengthRatio)*(1-tailConcavity), centerY,
		centerX + fishLength/2 - (tailInsetRatio * fishLength) + fishLength/2*tailLengthRatio, centerY + (fishHeight/2)*tailHeightRatio,
	})

	return fish
}

// Ways to make fish: name -> seed, series of digits (10 values for each knob of fish), total random.

func fishRequestHandler(w http.ResponseWriter, req *http.Request) {
	tmpl, err := template.New("fishTemplate").Parse(fishSvgTemplate)
	if err != nil {
		panic(err)
	}

	seed1, seed2 := rand.Uint64(), rand.Uint64()
	// If it's not a empty path, seed the fish!
	if len(req.URL.Path[1:]) > 0 {
		hash := sha1.Sum([]byte(req.URL.Path[1:]))
		seed1 = binary.BigEndian.Uint64(hash[0:8])
		seed2 = binary.BigEndian.Uint64(hash[8:16])
	}
	// TODO: Check if it's a sequence of X digits. Use them as params if true

	err = tmpl.Execute(w, generateFish(rand.New(rand.NewPCG(seed1, seed2))))
	if err != nil {
		panic(err)
	}
}

func CORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Access-Control-Allow-Origin", "*")
		w.Header().Add("Access-Control-Allow-Credentials", "true")
		w.Header().Add("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		w.Header().Add("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")

		if r.Method == "OPTIONS" {
			http.Error(w, "No Content", http.StatusNoContent)
			return
		}

		next(w, r)
	}
}

func main() {
	http.HandleFunc("/", CORS(fishRequestHandler))

	// Accept host and port on command line
	host := "localhost:18927"
	if len(os.Args) > 1 {
		host = os.Args[1]
	}
	log.Fatal(http.ListenAndServe(host, nil))
}
