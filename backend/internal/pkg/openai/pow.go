package openai

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// PowChallenge represents the PoW challenge from OpenAI
type PowChallenge struct {
	Seed       string `json:"seed"`
	Difficulty string `json:"difficulty"`
	UserAgent  string `json:"user_agent,omitempty"`
	ScriptHash string `json:"script,omitempty"`
}

// PowResult represents the solved PoW result
type PowResult struct {
	Seed       string `json:"seed"`
	Difficulty string `json:"difficulty"`
	Nonce      string `json:"nonce"`
	Answer     string `json:"answer"`
}

// SolvePow solves the PoW challenge using parallel computation
// difficulty format: "0xxxxx" where x count indicates difficulty level
func SolvePow(challenge *PowChallenge, timeoutSec int) (*PowResult, error) {
	if challenge == nil || challenge.Seed == "" {
		return nil, fmt.Errorf("invalid challenge")
	}

	difficulty := challenge.Difficulty
	if difficulty == "" {
		difficulty = "0" // default easy difficulty
	}

	// Count leading zeros required
	requiredZeros := 0
	for _, c := range difficulty {
		if c == '0' {
			requiredZeros++
		} else {
			break
		}
	}

	prefix := strings.Repeat("0", requiredZeros)
	numWorkers := runtime.NumCPU()
	timeout := time.Duration(timeoutSec) * time.Second

	var found atomic.Bool
	var result atomic.Value
	var wg sync.WaitGroup

	startTime := time.Now()
	ctx := make(chan struct{})

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			nonce := uint64(workerID)
			step := uint64(numWorkers)

			for !found.Load() {
				select {
				case <-ctx:
					return
				default:
				}

				if time.Since(startTime) > timeout {
					return
				}

				nonceStr := fmt.Sprintf("%d", nonce)
				data := challenge.Seed + nonceStr
				hash := sha256.Sum256([]byte(data))
				hashHex := hex.EncodeToString(hash[:])

				if strings.HasPrefix(hashHex, prefix) {
					if found.CompareAndSwap(false, true) {
						result.Store(&PowResult{
							Seed:       challenge.Seed,
							Difficulty: difficulty,
							Nonce:      nonceStr,
							Answer:     hashHex,
						})
						close(ctx)
					}
					return
				}
				nonce += step
			}
		}(i)
	}

	wg.Wait()

	if r := result.Load(); r != nil {
		return r.(*PowResult), nil
	}
	return nil, fmt.Errorf("pow solve timeout after %v", timeout)
}

// EncodePowResult encodes the PoW result for submission
func EncodePowResult(result *PowResult) (string, error) {
	if result == nil {
		return "", fmt.Errorf("nil result")
	}
	data, err := json.Marshal(result)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

// ParsePowChallenge parses PoW challenge from sentinel response
func ParsePowChallenge(data []byte) (*PowChallenge, error) {
	var challenge PowChallenge
	if err := json.Unmarshal(data, &challenge); err != nil {
		return nil, fmt.Errorf("parse challenge: %w", err)
	}
	return &challenge, nil
}

