package calibrate

import (
	"os"
	"testing"
)

func load(t *testing.T) *Log {
	t.Helper()
	f, err := os.Open("testdata/sample.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	l, err := Parse(f)
	if err != nil {
		t.Fatal(err)
	}
	return l
}

func find(s []SpellStats, player, spell string) *SpellStats {
	for i := range s {
		if s[i].Player == player && s[i].Spell == spell {
			return &s[i]
		}
	}
	return nil
}

func TestParseHeader(t *testing.T) {
	l := load(t)
	if l.Version != 22 || !l.Advanced {
		t.Fatalf("version %d advanced %v, want 22 true", l.Version, l.Advanced)
	}
	if l.Casts["Player-1-AAAA"]["Claw"] != 1 {
		t.Fatalf("Claw casts = %d, want 1", l.Casts["Player-1-AAAA"]["Claw"])
	}
}

func TestSpellStats(t *testing.T) {
	s := load(t).Spells(1)
	claw := find(s, "Cat", "Claw")
	if claw == nil {
		t.Fatal("no Claw stats")
	}
	if claw.Landed != 2 || claw.CritPct != 50 || claw.NonCritMed != 94 || claw.CritMed != 188 {
		t.Fatalf("Claw = %+v", *claw)
	}
	if claw.Avoided["PARRY"] != 1 || claw.AvoidedPct != 33.3 {
		t.Fatalf("Claw avoidance = %v %.1f", claw.Avoided, claw.AvoidedPct)
	}
	if rip := find(s, "Cat", "Rip"); rip == nil || rip.NonCritMed != 34 {
		t.Fatalf("Rip = %+v", rip)
	}
	melee := find(s, "Cat", "Melee")
	if melee == nil || melee.NonCritMed != 182 || melee.Avoided["DODGE"] != 1 {
		t.Fatalf("Melee = %+v", melee)
	}
	// The imp's swing is credited to its owner.
	if pet := find(s, "Lock", "Melee"); pet == nil || pet.TotalDamage != 14 {
		t.Fatalf("pet melee = %+v", pet)
	}
}

func TestEncounterDPS(t *testing.T) {
	p := load(t).EncounterDPS()
	if len(p) != 2 || p[0].Player != "Cat" || p[0].Encounter != "Grubbis" || !p[0].Success {
		t.Fatalf("pulls = %+v", p)
	}
	// 94 + 188 + 34 + 182 over 10 seconds.
	if p[0].Damage != 498 || p[0].DPS != 49.8 {
		t.Fatalf("Cat = %+v", p[0])
	}
}
