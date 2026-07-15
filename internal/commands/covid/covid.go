package covid

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"mad-news-bot/internal/cache"
)

const covidURL = "https://www.worldometers.info/coronavirus/"

type stats struct {
	A  int `json:"a"`
	C  int `json:"c"`
	D  int `json:"d"`
	R  int `json:"r"`
	RA int `json:"ra"`
	RC int `json:"rc"`
	RD int `json:"rd"`
	RR int `json:"rr"`
	NC int `json:"nc"`
}

type snapshot struct {
	Date int64 `json:"date"`
	stats
}

func Format(ctx context.Context, redis *cache.Redis) (string, error) {
	res, err := scrape(ctx)
	if err != nil {
		return "", err
	}
	if redis != nil {
		_ = updateSnapshots(ctx, redis, res)
	}

	population := 7_800_000_000
	casesPercent := round2(100 / (float64(population) / float64(res.C)) * 100)
	deathsPercent := round2(100 / (float64(res.C) / float64(res.D)) * 100)

	return fmt.Sprintf(`[Данные по SARS-Cov19 на 13 Апреля 2024]

[Мир] Случаев: %s
[Мир] Смертей: %s

[Россия] Случаев: %s
[Россия] Смертей: %s

[Процент заболевших (от %s)] %.2f%%
[Смертность] %.2f%%

(Источник: worldometers)
`, formatInt(res.C), formatInt(res.D), formatInt(res.RC), formatInt(res.RD),
		formatInt(population), casesPercent, deathsPercent), nil
}

func scrape(ctx context.Context) (*stats, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, covidURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}

	res := &stats{}
	doc.Find(".maincounter-number span").EachWithBreak(func(i int, s *goquery.Selection) bool {
		if i > 2 {
			return false
		}
		text := strings.ReplaceAll(strings.TrimSpace(s.Text()), ",", "")
		val, _ := strconv.Atoi(text)
		switch i {
		case 0:
			res.C = val
		case 1:
			res.D = val
		case 2:
			res.R = val
		}
		return true
	})

	doc.Find("#main_table_countries_today tbody tr").Each(func(_ int, row *goquery.Selection) {
		country := strings.TrimSpace(row.Find("td:nth-child(2)").Text())
		if country == "World" {
			text := strings.ReplaceAll(strings.TrimSpace(row.Find("td:nth-child(4)").Text()), ",", "")
			res.NC, _ = strconv.Atoi(text)
			return
		}
		link := strings.TrimSpace(row.Find("td:nth-child(2) a").Text())
		if link != "Russia" {
			return
		}
		res.RC = parseInt(row.Find("td:nth-child(3)").Text())
		res.RD = parseInt(row.Find("td:nth-child(5)").Text())
		res.RR = parseInt(row.Find("td:nth-child(7)").Text())
	})

	res.A = res.C - res.D - res.R
	res.RA = res.RC - res.RD - res.RR
	return res, nil
}

func updateSnapshots(ctx context.Context, redis *cache.Redis, res *stats) error {
	now := time.Now().UnixMilli()
	snap := snapshot{Date: now, stats: *res}
	raw, _ := json.Marshal(snap)

	prev, err := redis.Get(ctx, "prev")
	if err != nil {
		return redis.SetEX(ctx, "prev", string(raw), 14*24*time.Hour)
	}

	var prevSnap snapshot
	if json.Unmarshal([]byte(prev), &prevSnap) != nil {
		return redis.SetEX(ctx, "prev", string(raw), 14*24*time.Hour)
	}

	daysLeft1 := int((now - prevSnap.Date) / (24 * 3600000))
	if daysLeft1 <= 2 {
		return nil
	}

	nextPrev, err := redis.Get(ctx, "nextPrev")
	if err != nil {
		return redis.SetEX(ctx, "nextPrev", string(raw), 14*24*time.Hour)
	}

	var nextSnap snapshot
	if json.Unmarshal([]byte(nextPrev), &nextSnap) != nil {
		return redis.SetEX(ctx, "nextPrev", string(raw), 14*24*time.Hour)
	}

	daysLeft2 := int((now - nextSnap.Date) / (24 * 3600000))
	if daysLeft2 > 2 {
		_ = redis.SetEX(ctx, "prev", nextPrev, 14*24*time.Hour)
		return redis.SetEX(ctx, "nextPrev", string(raw), 14*24*time.Hour)
	}
	return nil
}

func parseInt(s string) int {
	text := strings.ReplaceAll(strings.TrimSpace(s), ",", "")
	val, _ := strconv.Atoi(text)
	return val
}

func formatInt(n int) string {
	return fmt.Sprintf("%d", n)
}

func round2(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}
