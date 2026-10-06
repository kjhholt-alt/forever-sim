// Usage: go run ./tools/calibrate/cmd -log WoWCombatLog.txt [-player Name] [-min 5] > observed.json
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/wowsims/forever/tools/calibrate"
)

func main() {
	path := flag.String("log", "", "WoW: Forever combat log (advanced logging on)")
	player := flag.String("player", "", "only report this player (name without realm)")
	minLanded := flag.Int("min", 5, "skip spells with fewer landed hits")
	flag.Parse()
	if *path == "" {
		flag.Usage()
		os.Exit(2)
	}
	f, err := os.Open(*path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer f.Close()
	l, err := calibrate.Parse(f)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	spells, pulls := l.Spells(*minLanded), l.EncounterDPS()
	if *player != "" {
		spells = filter(spells, func(s calibrate.SpellStats) bool { return s.Player == *player })
		pulls = filter(pulls, func(p calibrate.EncounterDPS) bool { return p.Player == *player })
	}
	out := map[string]any{
		"log_version":   l.Version,
		"advanced":      l.Advanced,
		"spells":        spells,
		"encounter_dps": pulls,
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func filter[T any](v []T, keep func(T) bool) []T {
	var out []T
	for _, x := range v {
		if keep(x) {
			out = append(out, x)
		}
	}
	return out
}
