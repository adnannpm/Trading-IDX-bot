package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

type StockQuote struct {
	Symbol               string  `json:"symbol"`
	DisplaySymbol        string  `json:"display_symbol"`
	Name                 string  `json:"name"`
	Price                float64 `json:"price"`
	Change               float64 `json:"change"`
	ChangePercent        float64 `json:"change_percent"`
	Open                 float64 `json:"open"`
	High                 float64 `json:"high"`
	Low                  float64 `json:"low"`
	PreviousClose        float64 `json:"previous_close"`
	Volume               int64   `json:"volume"`
	AvgVolume10Day       int64   `json:"avg_volume_10d"`
	VolumeSpikeRatio     float64 `json:"volume_spike_ratio"`
	IsVolumeSpike        bool    `json:"is_volume_spike"`
	Currency             string  `json:"currency"`
	FiftyTwoWeekHigh     float64 `json:"fifty_two_week_high"`
	FiftyTwoWeekLow      float64 `json:"fifty_two_week_low"`
	Is52WeekHighBreakout bool    `json:"is_52w_high_breakout"`
	MarketState          string  `json:"market_state"`
	LastTradeTime        int64   `json:"last_trade_time"`
	IsARA                bool    `json:"is_ara"`
	ARALimitPct          float64 `json:"ara_limit_pct"`
	IsARB                bool    `json:"is_arb"`
	ARBLimitPct          float64 `json:"arb_limit_pct"`
}

type yahooScreenerItem struct {
	Symbol                     string  `json:"symbol"`
	ShortName                  string  `json:"shortName"`
	LongName                   string  `json:"longName"`
	RegularMarketPrice         float64 `json:"regularMarketPrice"`
	RegularMarketChange        float64 `json:"regularMarketChange"`
	RegularMarketChangePercent float64 `json:"regularMarketChangePercent"`
	RegularMarketOpen          float64 `json:"regularMarketOpen"`
	RegularMarketDayHigh       float64 `json:"regularMarketDayHigh"`
	RegularMarketDayLow        float64 `json:"regularMarketDayLow"`
	RegularMarketPreviousClose float64 `json:"regularMarketPreviousClose"`
	RegularMarketVolume        int64   `json:"regularMarketVolume"`
	AverageDailyVolume10Day    int64   `json:"averageDailyVolume10Day"`
	AverageDailyVolume3Month   int64   `json:"averageDailyVolume3Month"`
	Currency                   string  `json:"currency"`
	MarketState                string  `json:"marketState"`
	RegularMarketTime          int64   `json:"regularMarketTime"`
	FiftyTwoWeekHigh           float64 `json:"fiftyTwoWeekHigh"`
	FiftyTwoWeekLow            float64 `json:"fiftyTwoWeekLow"`
}

type yahooScreenerResponse struct {
	Finance struct {
		Result []struct {
			Quotes []yahooScreenerItem `json:"quotes"`
		} `json:"result"`
		Error interface{} `json:"error"`
	} `json:"finance"`
}

type yahooChartResponse struct {
	Chart struct {
		Result []struct {
			Meta struct {
				Symbol               string  `json:"symbol"`
				ShortName            string  `json:"shortName"`
				LongName             string  `json:"longName"`
				RegularMarketPrice   float64 `json:"regularMarketPrice"`
				ChartPreviousClose   float64 `json:"chartPreviousClose"`
				PreviousClose        float64 `json:"previousClose"`
				RegularMarketOpen    float64 `json:"regularMarketOpen"`
				RegularMarketDayHigh float64 `json:"regularMarketDayHigh"`
				RegularMarketDayLow  float64 `json:"regularMarketDayLow"`
				RegularMarketVolume  int64   `json:"regularMarketVolume"`
				Currency             string  `json:"currency"`
				MarketState          string  `json:"marketState"`
				RegularMarketTime    int64   `json:"regularMarketTime"`
			} `json:"meta"`
		} `json:"result"`
	} `json:"chart"`
}

type yahooSession struct {
	client    *http.Client
	cookieStr string
	crumb     string
	expiresAt time.Time
}

var (
	sessionLock   sync.Mutex
	cachedSession *yahooSession
	UserAgent     = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36"
)

func ToDisplaySymbol(symbol string) string {
	if symbol == "^JKSE" {
		return "IHSG"
	}
	clean := strings.TrimSpace(symbol)
	clean = strings.TrimSuffix(strings.ToUpper(clean), ".JK")
	return clean
}

func NormalizeSymbol(symbol string) string {
	clean := strings.ToUpper(strings.TrimSpace(symbol))
	if clean == "IHSG" || clean == "JKSE" || clean == "^JKSE" {
		return "^JKSE"
	}
	if !strings.HasSuffix(clean, ".JK") {
		return clean + ".JK"
	}
	return clean
}

func CalculateARALimit(prevPrice float64) float64 {
	if prevPrice < 200 {
		return 35.0
	} else if prevPrice < 5000 {
		return 25.0
	}
	return 20.0
}

func CheckIsARA(changePct float64, prevPrice float64) (bool, float64) {
	limit := CalculateARALimit(prevPrice)
	if changePct >= limit-0.85 {
		return true, limit
	}
	if changePct >= 9.5 && changePct <= 10.5 {
		return true, 10.0
	}
	return false, limit
}

func CheckIsARB(changePct float64, prevPrice float64) (bool, float64) {
	limit := CalculateARALimit(prevPrice)
	if changePct <= -(limit - 0.85) {
		return true, limit
	}
	if changePct <= -9.5 && changePct >= -10.5 {
		return true, 10.0
	}
	return false, limit
}

func CheckVolumeSpike(vol, avg10d int64, changePct float64) (bool, float64) {
	if avg10d <= 0 || vol <= 0 {
		return false, 0
	}
	ratio := float64(vol) / float64(avg10d)
	if ratio >= 1.5 && changePct >= 4.0 && changePct <= 15.0 {
		return true, ratio
	}
	return false, ratio
}

func Check52WeekHighBreakout(price, high, fiftyTwoWeekHigh, changePct float64) bool {
	if fiftyTwoWeekHigh <= 0 || price <= 0 || changePct <= 0 {
		return false
	}
	if (high >= fiftyTwoWeekHigh || price >= fiftyTwoWeekHigh*0.995) && changePct >= 2.0 {
		return true
	}
	return false
}

func getYahooSession() (*yahooSession, error) {
	sessionLock.Lock()
	defer sessionLock.Unlock()

	now := time.Now()
	if cachedSession != nil && cachedSession.expiresAt.After(now) {
		return cachedSession, nil
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("gagal membuat cookie jar: %w", err)
	}

	client := &http.Client{
		Jar:     jar,
		Timeout: 10 * time.Second,
	}

	reqCookie, err := http.NewRequest("GET", "https://fc.yahoo.com", nil)
	if err != nil {
		return nil, err
	}
	reqCookie.Header.Set("User-Agent", UserAgent)
	respCookie, err := client.Do(reqCookie)
	if respCookie != nil && respCookie.Body != nil {
		io.Copy(io.Discard, respCookie.Body)
		respCookie.Body.Close()
	}

	reqCrumb, err := http.NewRequest("GET", "https://query2.finance.yahoo.com/v1/test/getcrumb", nil)
	if err != nil {
		return nil, err
	}
	reqCrumb.Header.Set("User-Agent", UserAgent)

	respCrumb, err := client.Do(reqCrumb)
	if err != nil {
		reqCrumb.URL, _ = url.Parse("https://query1.finance.yahoo.com/v1/test/getcrumb")
		respCrumb, err = client.Do(reqCrumb)
		if err != nil {
			return nil, fmt.Errorf("gagal request crumb Yahoo Finance: %w", err)
		}
	}
	defer respCrumb.Body.Close()

	crumbBytes, err := io.ReadAll(respCrumb.Body)
	if err != nil {
		return nil, fmt.Errorf("gagal membaca crumb response: %w", err)
	}
	crumb := strings.TrimSpace(string(crumbBytes))
	if crumb == "" || strings.Contains(crumb, "<html") || len(crumb) > 60 {
		return nil, fmt.Errorf("crumb tidak valid dari Yahoo: %s (status: %d)", crumb, respCrumb.StatusCode)
	}

	cachedSession = &yahooSession{
		client:    client,
		crumb:     crumb,
		expiresAt: now.Add(30 * time.Minute),
	}

	return cachedSession, nil
}

func queryScreener(sess *yahooSession, sortType string, size int) ([]StockQuote, error) {
	endpoint := fmt.Sprintf("https://query1.finance.yahoo.com/v1/finance/screener?crumb=%s", url.QueryEscape(sess.crumb))

	bodyPayload := map[string]interface{}{
		"size":        size,
		"offset":      0,
		"sortField":   "percentchange",
		"sortType":    sortType,
		"quoteType":   "EQUITY",
		"topOperator": "AND",
		"query": map[string]interface{}{
			"operator": "AND",
			"operands": []map[string]interface{}{
				{"operator": "EQ", "operands": []string{"region", "id"}},
			},
		},
	}

	bodyBytes, err := json.Marshal(bodyPayload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", UserAgent)

	resp, err := sess.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("yahoo screener mengembalikan HTTP %d", resp.StatusCode)
	}

	var screenerResp yahooScreenerResponse
	if err := json.NewDecoder(resp.Body).Decode(&screenerResp); err != nil {
		return nil, fmt.Errorf("gagal decode response screener: %w", err)
	}

	var results []StockQuote
	if len(screenerResp.Finance.Result) > 0 {
		for _, q := range screenerResp.Finance.Result[0].Quotes {
			displaySym := ToDisplaySymbol(q.Symbol)
			name := q.LongName
			if name == "" {
				name = q.ShortName
			}
			if name == "" {
				name = displaySym
			}

			price := q.RegularMarketPrice
			prevPrice := q.RegularMarketPreviousClose
			change := q.RegularMarketChange
			changePct := q.RegularMarketChangePercent

			if prevPrice == 0 && change != 0 {
				prevPrice = price - change
			} else if prevPrice == 0 {
				prevPrice = price
			}

			if change == 0 && prevPrice > 0 && price != prevPrice {
				change = price - prevPrice
			}

			if changePct == 0 && prevPrice > 0 && change != 0 {
				changePct = (change / prevPrice) * 100
			}

			if price <= 0 {
				continue
			}

			isAra, limitPct := CheckIsARA(changePct, prevPrice)
			isArb, arbLimitPct := CheckIsARB(changePct, prevPrice)
			isSpike, spikeRatio := CheckVolumeSpike(q.RegularMarketVolume, q.AverageDailyVolume10Day, changePct)
			is52w := Check52WeekHighBreakout(price, q.RegularMarketDayHigh, q.FiftyTwoWeekHigh, changePct)

			quote := StockQuote{
				Symbol:               q.Symbol,
				DisplaySymbol:        displaySym,
				Name:                 name,
				Price:                price,
				Change:               change,
				ChangePercent:        changePct,
				Open:                 q.RegularMarketOpen,
				High:                 q.RegularMarketDayHigh,
				Low:                  q.RegularMarketDayLow,
				PreviousClose:        prevPrice,
				Volume:               q.RegularMarketVolume,
				AvgVolume10Day:       q.AverageDailyVolume10Day,
				VolumeSpikeRatio:     spikeRatio,
				IsVolumeSpike:        isSpike,
				FiftyTwoWeekHigh:     q.FiftyTwoWeekHigh,
				FiftyTwoWeekLow:      q.FiftyTwoWeekLow,
				Is52WeekHighBreakout: is52w,
				Currency:             q.Currency,
				MarketState:          q.MarketState,
				LastTradeTime:        q.RegularMarketTime,
				IsARA:                isAra,
				ARALimitPct:          limitPct,
				IsARB:                isArb,
				ARBLimitPct:          arbLimitPct,
			}
			results = append(results, quote)
		}
	}

	return results, nil
}

func FetchTopGainers(limit int) ([]StockQuote, error) {
	if limit <= 0 {
		limit = 10
	}

	sess, err := getYahooSession()
	if err != nil {
		return nil, fmt.Errorf("gagal mendapatkan session Yahoo: %w", err)
	}

	fetchSize := 30
	if limit > fetchSize {
		fetchSize = limit + 10
	}

	quotes, err := queryScreener(sess, "DESC", fetchSize)
	if err != nil {
		sessionLock.Lock()
		cachedSession = nil
		sessionLock.Unlock()

		sess, err = getYahooSession()
		if err != nil {
			return nil, fmt.Errorf("retry get session gagal: %w", err)
		}
		quotes, err = queryScreener(sess, "DESC", fetchSize)
		if err != nil {
			return nil, fmt.Errorf("screener gagal setelah retry: %w", err)
		}
	}

	var gainers []StockQuote
	for _, q := range quotes {
		if q.ChangePercent > 0 {
			gainers = append(gainers, q)
		}
	}

	for i := 0; i < len(gainers)-1; i++ {
		for j := i + 1; j < len(gainers); j++ {
			if gainers[i].ChangePercent < gainers[j].ChangePercent {
				gainers[i], gainers[j] = gainers[j], gainers[i]
			}
		}
	}

	if len(gainers) > limit {
		gainers = gainers[:limit]
	}

	return gainers, nil
}

func FetchTopLosers(limit int) ([]StockQuote, error) {
	if limit <= 0 {
		limit = 10
	}

	sess, err := getYahooSession()
	if err != nil {
		return nil, fmt.Errorf("gagal mendapatkan session Yahoo: %w", err)
	}

	fetchSize := 35
	if limit > fetchSize {
		fetchSize = limit + 10
	}

	quotes, err := queryScreener(sess, "ASC", fetchSize)
	if err != nil {
		sessionLock.Lock()
		cachedSession = nil
		sessionLock.Unlock()

		sess, err = getYahooSession()
		if err != nil {
			return nil, fmt.Errorf("retry get session gagal: %w", err)
		}
		quotes, err = queryScreener(sess, "ASC", fetchSize)
		if err != nil {
			return nil, fmt.Errorf("screener ARB gagal setelah retry: %w", err)
		}
	}

	var losers []StockQuote
	for _, q := range quotes {
		if q.ChangePercent < 0 {
			losers = append(losers, q)
		}
	}

	for i := 0; i < len(losers)-1; i++ {
		for j := i + 1; j < len(losers); j++ {
			if losers[i].ChangePercent > losers[j].ChangePercent {
				losers[i], losers[j] = losers[j], losers[i]
			}
		}
	}

	if len(losers) > limit {
		losers = losers[:limit]
	}

	return losers, nil
}

func FetchMomentumStocks(limit int) ([]StockQuote, error) {
	if limit <= 0 {
		limit = 10
	}

	sess, err := getYahooSession()
	if err != nil {
		return nil, fmt.Errorf("gagal mendapatkan session Yahoo: %w", err)
	}

	fetchSize := 60
	quotes, err := queryScreener(sess, "DESC", fetchSize)
	if err != nil {
		sessionLock.Lock()
		cachedSession = nil
		sessionLock.Unlock()

		sess, err = getYahooSession()
		if err != nil {
			return nil, fmt.Errorf("retry get session gagal: %w", err)
		}
		quotes, err = queryScreener(sess, "DESC", fetchSize)
		if err != nil {
			return nil, fmt.Errorf("screener momentum gagal setelah retry: %w", err)
		}
	}

	var candidates []StockQuote
	seen := make(map[string]bool)

	for _, q := range quotes {
		if seen[q.Symbol] {
			continue
		}

		hasLiquidity := q.Volume >= 200_000

		if (q.IsVolumeSpike || q.Is52WeekHighBreakout) && hasLiquidity {
			candidates = append(candidates, q)
			seen[q.Symbol] = true
		}
	}

	for i := 0; i < len(candidates)-1; i++ {
		for j := i + 1; j < len(candidates); j++ {
			scoreI := 0
			if candidates[i].IsVolumeSpike {
				scoreI += 2
			}
			if candidates[i].Is52WeekHighBreakout {
				scoreI += 2
			}

			scoreJ := 0
			if candidates[j].IsVolumeSpike {
				scoreJ += 2
			}
			if candidates[j].Is52WeekHighBreakout {
				scoreJ += 2
			}

			if scoreI < scoreJ || (scoreI == scoreJ && candidates[i].VolumeSpikeRatio < candidates[j].VolumeSpikeRatio) {
				candidates[i], candidates[j] = candidates[j], candidates[i]
			}
		}
	}

	if len(candidates) > limit {
		candidates = candidates[:limit]
	}

	return candidates, nil
}

func FetchStockQuote(symbol string) (*StockQuote, error) {
	normalized := NormalizeSymbol(symbol)
	hosts := []string{"query1.finance.yahoo.com", "query2.finance.yahoo.com"}
	var lastErr error

	for _, host := range hosts {
		endpoint := fmt.Sprintf("https://%s/v8/finance/chart/%s?interval=5m&range=1d&includePrePost=false", host, url.PathEscape(normalized))
		req, err := http.NewRequest("GET", endpoint, nil)
		if err != nil {
			lastErr = err
			continue
		}
		req.Header.Set("User-Agent", UserAgent)
		req.Header.Set("Accept", "application/json")

		client := &http.Client{Timeout: 8 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("status code %d", resp.StatusCode)
			continue
		}

		var chartResp yahooChartResponse
		if err := json.NewDecoder(resp.Body).Decode(&chartResp); err != nil {
			lastErr = err
			continue
		}

		if len(chartResp.Chart.Result) == 0 {
			lastErr = fmt.Errorf("data chart kosong untuk %s", symbol)
			continue
		}

		meta := chartResp.Chart.Result[0].Meta
		price := meta.RegularMarketPrice
		prevClose := meta.ChartPreviousClose
		if prevClose == 0 {
			prevClose = meta.PreviousClose
		}
		change := price - prevClose
		changePct := float64(0)
		if prevClose > 0 {
			changePct = (change / prevClose) * 100
		}

		name := meta.LongName
		if name == "" {
			name = meta.ShortName
		}
		if name == "" {
			name = ToDisplaySymbol(normalized)
		}

		isAra, limitPct := CheckIsARA(changePct, prevClose)
		isArb, arbLimitPct := CheckIsARB(changePct, prevClose)

		return &StockQuote{
			Symbol:        meta.Symbol,
			DisplaySymbol: ToDisplaySymbol(meta.Symbol),
			Name:          name,
			Price:         price,
			Change:        change,
			ChangePercent: changePct,
			Open:          meta.RegularMarketOpen,
			High:          meta.RegularMarketDayHigh,
			Low:           meta.RegularMarketDayLow,
			PreviousClose: prevClose,
			Volume:        meta.RegularMarketVolume,
			Currency:      meta.Currency,
			MarketState:   meta.MarketState,
			LastTradeTime: meta.RegularMarketTime,
			IsARA:         isAra,
			ARALimitPct:   limitPct,
			IsARB:         isArb,
			ARBLimitPct:   arbLimitPct,
		}, nil
	}

	return nil, fmt.Errorf("gagal mengambil quote %s: %v", symbol, lastErr)
}

func FormatNumberWithDots(n int64) string {
	in := strconv.FormatInt(n, 10)
	var out []byte
	isNegative := false
	if len(in) > 0 && in[0] == '-' {
		isNegative = true
		in = in[1:]
	}
	l := len(in)
	for i, c := range in {
		if i > 0 && (l-i)%3 == 0 {
			out = append(out, '.')
		}
		out = append(out, byte(c))
	}
	if isNegative {
		return "-" + string(out)
	}
	return string(out)
}

func FormatVolume(vol int64) string {
	lots := vol / 100
	if vol >= 1_000_000_000 {
		return fmt.Sprintf("%.2fB lembar (%s lot)", float64(vol)/1_000_000_000, FormatNumberWithDots(lots))
	} else if vol >= 1_000_000 {
		return fmt.Sprintf("%.2fM lembar (%s lot)", float64(vol)/1_000_000, FormatNumberWithDots(lots))
	} else if vol >= 1_000 {
		return fmt.Sprintf("%.2fK lembar (%s lot)", float64(vol)/1_000, FormatNumberWithDots(lots))
	}
	return fmt.Sprintf("%d lembar (%s lot)", vol, FormatNumberWithDots(lots))
}
