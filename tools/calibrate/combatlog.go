// Package calibrate turns a WoW: Forever combat log (WoWCombatLog.txt, advanced logging on) into
// observed per-spell numbers the sim can be checked against: hit sizes, crit rates, avoidance and
// per-encounter DPS. Nothing here touches the game; it only reads the file the client writes.
package calibrate

import (
	"bufio"
	"encoding/csv"
	"io"
	"regexp"
	"strconv"
	"strings"
)

// Event is one damage or miss line from a player (or a player's pet, credited to its owner).
type Event struct {
	T      float64 // seconds since the first line
	Source string  // player GUID the damage is credited to
	Name   string  // player name without the realm
	Spell  string  // "Melee" for white swings
	Amount int
	Crit   bool
	Miss   string // MISS, DODGE, PARRY, ... for _MISSED events
	Target string // target GUID
}

type Encounter struct {
	Name       string
	Start, End float64
	Success    bool
}

type Log struct {
	Version    int
	Advanced   bool
	Events     []Event
	Casts      map[string]map[string]int // player GUID -> spell -> SPELL_CAST_SUCCESS count
	Names      map[string]string         // player GUID -> name
	Encounters []Encounter
}

// "10/4/2026 19:17:35.137-5  EVENT,..." (the -5 is the UTC offset Forever appends).
var lineRE = regexp.MustCompile(`^(\d+)/(\d+)/\d+ (\d+):(\d+):(\d+)\.(\d+)(?:[-+]\d+)?\s\s(.*)$`)
var tagRE = regexp.MustCompile(`^[A-Z]+$`)

func Parse(r io.Reader) (*Log, error) {
	l := &Log{Version: 22, Casts: map[string]map[string]int{}, Names: map[string]string{}}
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	base := -1.0
	for sc.Scan() {
		m := lineRE.FindStringSubmatch(sc.Text())
		if m == nil {
			continue
		}
		mon, _ := strconv.Atoi(m[1])
		day, _ := strconv.Atoi(m[2])
		h, _ := strconv.Atoi(m[3])
		mi, _ := strconv.Atoi(m[4])
		s, _ := strconv.Atoi(m[5])
		ms, _ := strconv.Atoi(m[6])
		t := float64((mon*31+day)*86400+h*3600+mi*60+s) + float64(ms)/1000
		if base < 0 {
			base = t
		}
		t -= base

		cr := csv.NewReader(strings.NewReader(m[7]))
		cr.LazyQuotes = true
		f, err := cr.Read()
		if err != nil || len(f) == 0 {
			continue
		}
		ev := f[0]
		switch ev {
		case "COMBAT_LOG_VERSION":
			l.Version, _ = strconv.Atoi(at(f, 1))
			for i := range f {
				if f[i] == "ADVANCED_LOG_ENABLED" && at(f, i+1) == "1" {
					l.Advanced = true
				}
			}
			continue
		case "ENCOUNTER_START":
			l.Encounters = append(l.Encounters, Encounter{Name: at(f, 2), Start: t, End: -1})
			continue
		case "ENCOUNTER_END":
			for i := len(l.Encounters) - 1; i >= 0; i-- {
				if e := &l.Encounters[i]; e.Name == at(f, 2) && e.End < 0 {
					e.End, e.Success = t, at(f, 5) == "1"
					break
				}
			}
			continue
		}
		if len(f) < 9 {
			continue
		}
		src := f[1]
		if strings.HasPrefix(src, "Player-") {
			l.Names[src] = strings.SplitN(f[2], "-", 2)[0]
		}
		spell := "Melee"
		if !strings.HasPrefix(ev, "SWING") {
			if !strings.HasPrefix(ev, "SPELL") && !strings.HasPrefix(ev, "RANGE") {
				continue
			}
			spell = at(f, 10)
		}
		if ev == "SPELL_CAST_SUCCESS" && strings.HasPrefix(src, "Player-") {
			if l.Casts[src] == nil {
				l.Casts[src] = map[string]int{}
			}
			l.Casts[src][spell]++
			continue
		}

		isDmg := strings.HasSuffix(ev, "_DAMAGE") && !strings.Contains(ev, "DURABILITY") && !strings.Contains(ev, "ENVIRONMENTAL")
		isMiss := strings.HasSuffix(ev, "_MISSED")
		if !isDmg && !isMiss {
			continue
		}
		// Pet and totem damage goes to its owner: with advanced logging the unit block names it.
		if !strings.HasPrefix(src, "Player-") {
			owner := at(f, 13)
			if strings.HasPrefix(ev, "SWING") {
				owner = at(f, 10)
			}
			if !l.Advanced || !strings.HasPrefix(owner, "Player-") {
				continue
			}
			src = owner
		}
		e := Event{T: t, Source: src, Spell: spell, Target: f[5]}
		if isDmg {
			// Damage suffix (log version 22): amount, baseAmount, overkill, school, resisted, blocked,
			// absorbed, critical, glancing, crushing, then an optional target tag such as "ST".
			n := len(f)
			if tagRE.MatchString(f[n-1]) {
				n--
			}
			if n < 10 {
				continue
			}
			e.Amount, _ = strconv.Atoi(f[n-10])
			e.Crit = f[n-3] == "1"
		} else {
			if strings.HasPrefix(ev, "SWING") {
				e.Miss = at(f, 9)
			} else {
				e.Miss = at(f, 12)
			}
		}
		l.Events = append(l.Events, e)
	}
	for i := range l.Events {
		l.Events[i].Name = l.Names[l.Events[i].Source]
	}
	return l, sc.Err()
}

func at(f []string, i int) string {
	if i < len(f) {
		return f[i]
	}
	return ""
}
