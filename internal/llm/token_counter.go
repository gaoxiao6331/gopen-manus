package llm

import (
	"fmt"
	"math"

	"github.com/pkoukk/tiktoken-go"
)

const (
	// Token constants
	baseMessageTokens    = 4
	formatTokens         = 2
	lowDetailImageTokens = 85
	highDetailTileTokens = 170

	// Image processing constants
	maxSize                   = 2048
	highDetailTargetShortSide = 768
	tileSize                  = 512
)

// TokenCounter is used to count tokens for text and images.
type TokenCounter struct {
	tokenizer *tiktoken.Tiktoken
}

// NewTokenCounter creates a new TokenCounter.
func NewTokenCounter(model string) (*TokenCounter, error) {
	tkm, err := tiktoken.EncodingForModel(model)
	if err != nil {
		tkm, err = tiktoken.GetEncoding("cl100k_base")
		if err != nil {
			return nil, fmt.Errorf("failed to get tiktoken encoding: %w", err)
		}
	}
	return &TokenCounter{tokenizer: tkm}, nil
}

// CountText calculates tokens for a text string.
func (tc *TokenCounter) CountText(text string) int {
	if text == "" {
		return 0
	}
	return len(tc.tokenizer.Encode(text, nil, nil))
}

// CountImage calculates tokens for an image based on detail level and dimensions.
func (tc *TokenCounter) CountImage(width, height int, detail string) int {
	if detail == "low" {
		return lowDetailImageTokens
	}

	// For "medium" or "high" detail
	return tc.calculateHighDetailTokens(width, height)
}

func (tc *TokenCounter) calculateHighDetailTokens(width, height int) int {
	// Step 1: Scale to fit in maxSize x maxSize square
	if width > maxSize || height > maxSize {
		scale := float64(maxSize) / float64(max(width, height))
		width = int(float64(width) * scale)
		height = int(float64(height) * scale)
	}

	// Step 2: Scale so shortest side is highDetailTargetShortSide
	scale := float64(highDetailTargetShortSide) / float64(min(width, height))
	scaledWidth := int(float64(width) * scale)
	scaledHeight := int(float64(height) * scale)

	// Step 3: Count number of 512px tiles
	tilesX := math.Ceil(float64(scaledWidth) / tileSize)
	tilesY := math.Ceil(float64(scaledHeight) / tileSize)
	totalTiles := tilesX * tilesY

	// Step 4: Calculate final token count
	return int(totalTiles*highDetailTileTokens) + lowDetailImageTokens
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
