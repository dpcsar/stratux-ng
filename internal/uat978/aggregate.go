package uat978

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

type AggregatorConfig struct {
	MaxTowers int
	MaxText   int
	MaxRows   int
}

type Aggregator struct {
	mu       sync.Mutex
	towers   map[string]*towerStats
	products map[uint32]*productStats
	textRing []TextReport
	textNext int
	textSize int
	cfg      AggregatorConfig
}

type bucket struct {
	sec   int64
	count uint32
	sumDb float64
	ssN   uint32
}

type towerStats struct {
	key string
	lat float64
	lon float64

	signalNowDb float64
	signalMaxDb float64
	lastSeen    time.Time
	total       uint64
	buckets     [60]bucket
}

type productStats struct {
	id       uint32
	lastSeen time.Time
	total    uint64
	buckets  [60]bucket
}

// TextReport is a single decoded weather line.
type TextReport struct {
	ReceivedUTC string  `json:"received_utc"`
	TowerLatDeg float64 `json:"tower_lat_deg"`
	TowerLonDeg float64 `json:"tower_lon_deg"`
	Text        string  `json:"text"`
}

type TowerSnapshot struct {
	Key               string  `json:"key"`
	LatDeg            float64 `json:"lat_deg"`
	LonDeg            float64 `json:"lon_deg"`
	SignalNowDb       float64 `json:"signal_now_db"`
	SignalAvg1MinDb   float64 `json:"signal_avg_1min_db"`
	SignalMaxDb       float64 `json:"signal_max_db"`
	MessagesLastMin   uint64  `json:"messages_last_min"`
	MessagesTotal     uint64  `json:"messages_total"`
	LastSeenUTC       string  `json:"last_seen_utc"`
	HasSignalStrength bool    `json:"has_signal_strength"`
}

type ProductSnapshot struct {
	ProductID        uint32 `json:"product_id"`
	MessagesLastMin  uint64 `json:"messages_last_min"`
	MessagesTotal    uint64 `json:"messages_total"`
	LastSeenUTC      string `json:"last_seen_utc"`
	ProductName      string `json:"product_name"`
	IsText           bool   `json:"is_text"`
	IsNexradRegional bool   `json:"is_nexrad_regional"`
	IsNexradNational bool   `json:"is_nexrad_national"`
}

type WeatherSnapshot struct {
	Products []ProductSnapshot `json:"products"`
	Text     []TextReport      `json:"text"`
}

func NewAggregator(cfg AggregatorConfig) *Aggregator {
	if cfg.MaxTowers <= 0 {
		cfg.MaxTowers = 64
	}
	if cfg.MaxText <= 0 {
		cfg.MaxText = 40
	}
	if cfg.MaxRows <= 0 {
		cfg.MaxRows = 50
	}
	return &Aggregator{
		towers:   make(map[string]*towerStats),
		products: make(map[uint32]*productStats),
		textRing: make([]TextReport, cfg.MaxText),
		cfg:      cfg,
	}
}

func (a *Aggregator) Add(nowUTC time.Time, decoded DecodedUplink, signalDb float64, hasSignal bool) {
	if a == nil {
		return
	}
	if nowUTC.IsZero() {
		nowUTC = time.Now().UTC()
	}

	sec := nowUTC.Unix()
	key := fmt.Sprintf("(%.6f,%.6f)", decoded.TowerLatDeg, decoded.TowerLonDeg)

	a.mu.Lock()
	defer a.mu.Unlock()

	// Tower stats.
	tw := a.towers[key]
	if tw == nil {
		if len(a.towers) >= a.cfg.MaxTowers {
			// Simple eviction: drop an arbitrary entry.
			for k := range a.towers {
				delete(a.towers, k)
				break
			}
		}
		tw = &towerStats{key: key, lat: decoded.TowerLatDeg, lon: decoded.TowerLonDeg, signalMaxDb: -999}
		a.towers[key] = tw
	}
	tw.lastSeen = nowUTC
	tw.total++
	if hasSignal {
		tw.signalNowDb = signalDb
		if signalDb > tw.signalMaxDb {
			tw.signalMaxDb = signalDb
		}
	}
	bidx := int(sec % 60)
	b := &tw.buckets[bidx]
	if b.sec != sec {
		*b = bucket{sec: sec}
	}
	b.count++
	if hasSignal {
		b.sumDb += signalDb
		b.ssN++
	}

	// Product stats.
	for _, pid := range decoded.ProductIDs {
		ps := a.products[pid]
		if ps == nil {
			ps = &productStats{id: pid}
			a.products[pid] = ps
		}
		ps.lastSeen = nowUTC
		ps.total++
		pb := &ps.buckets[bidx]
		if pb.sec != sec {
			*pb = bucket{sec: sec}
		}
		pb.count++
	}

	// Text reports.
	for _, line := range decoded.TextReports {
		if line == "" {
			continue
		}
		a.textRing[a.textNext] = TextReport{
			ReceivedUTC: nowUTC.Format(time.RFC3339Nano),
			TowerLatDeg: decoded.TowerLatDeg,
			TowerLonDeg: decoded.TowerLonDeg,
			Text:        line,
		}
		a.textNext = (a.textNext + 1) % len(a.textRing)
		if a.textSize < len(a.textRing) {
			a.textSize++
		}
	}
}

func (a *Aggregator) Snapshot(nowUTC time.Time) (towers []TowerSnapshot, weather WeatherSnapshot) {
	if a == nil {
		return nil, WeatherSnapshot{}
	}
	if nowUTC.IsZero() {
		nowUTC = time.Now().UTC()
	}
	sec := nowUTC.Unix()
	minSec := sec - 59

	a.mu.Lock()
	defer a.mu.Unlock()

	towers = make([]TowerSnapshot, 0, len(a.towers))
	for _, tw := range a.towers {
		var cnt uint64
		var sumDb float64
		var ssN uint64
		for i := range tw.buckets {
			b := tw.buckets[i]
			if b.sec < minSec {
				continue
			}
			cnt += uint64(b.count)
			sumDb += b.sumDb
			ssN += uint64(b.ssN)
		}
		if cnt == 0 {
			continue
		}
		avg := 0.0
		hasSignal := ssN > 0
		if hasSignal {
			avg = sumDb / float64(ssN)
		}
		lastSeen := ""
		if !tw.lastSeen.IsZero() {
			lastSeen = tw.lastSeen.UTC().Format(time.RFC3339Nano)
		}
		towers = append(towers, TowerSnapshot{
			Key:               tw.key,
			LatDeg:            tw.lat,
			LonDeg:            tw.lon,
			SignalNowDb:       tw.signalNowDb,
			SignalAvg1MinDb:   avg,
			SignalMaxDb:       tw.signalMaxDb,
			MessagesLastMin:   cnt,
			MessagesTotal:     tw.total,
			LastSeenUTC:       lastSeen,
			HasSignalStrength: hasSignal,
		})
	}
	if len(towers) > 0 {
		sort.Slice(towers, func(i, j int) bool {
			if towers[i].MessagesLastMin == towers[j].MessagesLastMin {
				return towers[i].Key < towers[j].Key
			}
			return towers[i].MessagesLastMin > towers[j].MessagesLastMin
		})
		if len(towers) > a.cfg.MaxRows {
			towers = towers[:a.cfg.MaxRows]
		}
	}

	products := make([]ProductSnapshot, 0, len(a.products))
	for _, ps := range a.products {
		var cnt uint64
		for i := range ps.buckets {
			b := ps.buckets[i]
			if b.sec < minSec {
				continue
			}
			cnt += uint64(b.count)
		}
		if cnt == 0 {
			continue
		}
		lastSeen := ""
		if !ps.lastSeen.IsZero() {
			lastSeen = ps.lastSeen.UTC().Format(time.RFC3339Nano)
		}
		p := ProductSnapshot{
			ProductID:        ps.id,
			MessagesLastMin:  cnt,
			MessagesTotal:    ps.total,
			LastSeenUTC:      lastSeen,
			ProductName:      productName(ps.id),
			IsText:           ps.id == 413,
			IsNexradRegional: ps.id == 63,
			IsNexradNational: ps.id == 64,
		}
		products = append(products, p)
	}
	sort.Slice(products, func(i, j int) bool {
		if products[i].MessagesLastMin == products[j].MessagesLastMin {
			return products[i].ProductID < products[j].ProductID
		}
		return products[i].MessagesLastMin > products[j].MessagesLastMin
	})
	if len(products) > a.cfg.MaxRows {
		products = products[:a.cfg.MaxRows]
	}

	// Text ring snapshot newest-first.
	text := make([]TextReport, 0, a.textSize)
	for i := 0; i < a.textSize; i++ {
		idx := a.textNext - 1 - i
		for idx < 0 {
			idx += len(a.textRing)
		}
		tr := a.textRing[idx]
		if tr.Text == "" {
			continue
		}
		text = append(text, tr)
	}
	if len(text) > a.cfg.MaxRows {
		text = text[:a.cfg.MaxRows]
	}

	weather = WeatherSnapshot{Products: products, Text: text}
	return towers, weather
}

func productName(id uint32) string {
	switch id {
	case 413:
		return "Text (DLAC)"
	case 63:
		return "NEXRAD (Regional)"
	case 64:
		return "NEXRAD (National)"
	default:
		return ""
	}
}
