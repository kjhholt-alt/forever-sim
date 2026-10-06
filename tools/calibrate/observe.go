package calibrate

import (
	"sort"
)

// SpellStats is what a log says about one player's spell: the numbers the sim's spell data and
// attack table have to reproduce for that player's gear.
type SpellStats struct {
	Player      string         `json:"player"`
	Spell       string         `json:"spell"`
	Landed      int            `json:"landed"`
	CritPct     float64        `json:"crit_pct"`
	NonCritMin  int            `json:"noncrit_min"`
	NonCritMed  float64        `json:"noncrit_median"`
	NonCritMax  int            `json:"noncrit_max"`
	CritMed     float64        `json:"crit_median"`
	Avoided     map[string]int `json:"avoided,omitempty"`
	AvoidedPct  float64        `json:"avoided_pct"`
	TotalDamage int            `json:"total_damage"`
}

// EncounterDPS is the per-player damage over a boss pull: the number a sim run of the same pull
// is graded against.
type EncounterDPS struct {
	Encounter string  `json:"encounter"`
	Success   bool    `json:"success"`
	Seconds   float64 `json:"seconds"`
	Player    string  `json:"player"`
	Damage    int     `json:"damage"`
	DPS       float64 `json:"dps"`
}

// Spells summarises every player spell with at least minLanded landed hits.
func (l *Log) Spells(minLanded int) []SpellStats {
	type acc struct {
		nc, cr []int
		miss   map[string]int
		total  int
	}
	by := map[[2]string]*acc{}
	for _, e := range l.Events {
		k := [2]string{e.Source, e.Spell}
		a := by[k]
		if a == nil {
			a = &acc{miss: map[string]int{}}
			by[k] = a
		}
		switch {
		case e.Miss != "":
			a.miss[e.Miss]++
		case e.Crit:
			a.cr = append(a.cr, e.Amount)
		default:
			a.nc = append(a.nc, e.Amount)
		}
		a.total += e.Amount
	}
	var out []SpellStats
	for k, a := range by {
		landed := len(a.nc) + len(a.cr)
		if landed < minLanded {
			continue
		}
		s := SpellStats{Player: l.Names[k[0]], Spell: k[1], Landed: landed, TotalDamage: a.total,
			CritPct: pct(len(a.cr), landed), NonCritMed: median(a.nc), CritMed: median(a.cr)}
		if len(a.nc) > 0 {
			s.NonCritMin, s.NonCritMax = minMax(a.nc)
		}
		missed := 0
		for _, n := range a.miss {
			missed += n
		}
		if missed > 0 {
			s.Avoided = a.miss
			s.AvoidedPct = pct(missed, landed+missed)
		}
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Player != out[j].Player {
			return out[i].Player < out[j].Player
		}
		return out[i].TotalDamage > out[j].TotalDamage
	})
	return out
}

// EncounterDPS returns each player's damage and DPS for every finished boss pull.
func (l *Log) EncounterDPS() []EncounterDPS {
	var out []EncounterDPS
	for _, enc := range l.Encounters {
		if enc.End <= enc.Start {
			continue
		}
		dmg := map[string]int{}
		for _, e := range l.Events {
			if e.T >= enc.Start && e.T <= enc.End {
				dmg[e.Source] += e.Amount
			}
		}
		secs := enc.End - enc.Start
		var pull []EncounterDPS
		for src, d := range dmg {
			if d == 0 {
				continue
			}
			pull = append(pull, EncounterDPS{Encounter: enc.Name, Success: enc.Success, Seconds: round1(secs),
				Player: l.Names[src], Damage: d, DPS: round1(float64(d) / secs)})
		}
		sort.Slice(pull, func(i, j int) bool { return pull[i].DPS > pull[j].DPS })
		out = append(out, pull...)
	}
	return out
}

func pct(n, d int) float64 {
	if d == 0 {
		return 0
	}
	return round1(100 * float64(n) / float64(d))
}

func median(v []int) float64 {
	if len(v) == 0 {
		return 0
	}
	s := append([]int(nil), v...)
	sort.Ints(s)
	if len(s)%2 == 1 {
		return float64(s[len(s)/2])
	}
	return float64(s[len(s)/2-1]+s[len(s)/2]) / 2
}

func minMax(v []int) (int, int) {
	lo, hi := v[0], v[0]
	for _, x := range v {
		lo, hi = min(lo, x), max(hi, x)
	}
	return lo, hi
}

func round1(x float64) float64 { return float64(int(x*10+0.5)) / 10 }
