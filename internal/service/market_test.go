package service

import (
	"testing"
)

func TestCalculateARALimit(t *testing.T) {
	if limit := CalculateARALimit(150); limit != 35.0 {
		t.Errorf("Expected limit 35.0, got %v", limit)
	}
	if limit := CalculateARALimit(1000); limit != 25.0 {
		t.Errorf("Expected limit 25.0, got %v", limit)
	}
	if limit := CalculateARALimit(6000); limit != 20.0 {
		t.Errorf("Expected limit 20.0, got %v", limit)
	}
}

func TestCheckIsARA(t *testing.T) {
	isAra, limit := CheckIsARA(34.5, 100)
	if !isAra || limit != 35.0 {
		t.Errorf("Expected isAra true and limit 35.0, got %v, %v", isAra, limit)
	}

	isAra, limit = CheckIsARA(24.8, 1000)
	if !isAra || limit != 25.0 {
		t.Errorf("Expected isAra true and limit 25.0, got %v, %v", isAra, limit)
	}

	isAra, _ = CheckIsARA(5.0, 1000)
	if isAra {
		t.Errorf("Expected isAra false for 5.0%%, got true")
	}
}

func TestFormatNumberWithDots(t *testing.T) {
	if res := FormatNumberWithDots(1000); res != "1.000" {
		t.Errorf("Expected 1.000, got %s", res)
	}
	if res := FormatNumberWithDots(12345678); res != "12.345.678" {
		t.Errorf("Expected 12.345.678, got %s", res)
	}
	if res := FormatNumberWithDots(50); res != "50" {
		t.Errorf("Expected 50, got %s", res)
	}
}

func TestFetchTopGainersLive(t *testing.T) {
	gainers, err := FetchTopGainers(5)
	if err != nil {
		t.Logf("Live test error (could be network related): %v", err)
		return
	}

	if len(gainers) == 0 {
		t.Errorf("Expected at least 1 gainer, got 0")
	}

	for i, g := range gainers {
		t.Logf("#%d: %s (%s) Price: %.2f Prev: %.2f Change: +%.2f (%.2f%%) Vol: %d ARA: %v",
			i+1, g.DisplaySymbol, g.Name, g.Price, g.PreviousClose, g.Change, g.ChangePercent, g.Volume, g.IsARA)
	}
}

func TestFetchTopLosersLive(t *testing.T) {
	losers, err := FetchTopLosers(5)
	if err != nil {
		t.Logf("Live test error (could be network related): %v", err)
		return
	}

	if len(losers) == 0 {
		t.Errorf("Expected at least 1 loser, got 0")
	}

	for i, l := range losers {
		t.Logf("#%d: %s (%s) Price: %.2f Prev: %.2f Change: %.2f (%.2f%%) Vol: %d ARB: %v",
			i+1, l.DisplaySymbol, l.Name, l.Price, l.PreviousClose, l.Change, l.ChangePercent, l.Volume, l.IsARB)
	}
}
