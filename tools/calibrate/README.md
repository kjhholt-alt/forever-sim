# calibrate: grade the sim against real WoW: Forever combat logs

Reads a `WoWCombatLog.txt` written with Advanced Combat Logging on (log version 22). It outputs what the log observed for each player:

- **Per spell:** landed hits, crit %, non-crit min/median/max, crit median, avoidance (miss/dodge/parry/...), and total damage. Pet and totem damage is credited to the owner.
- **Per boss pull:** each player's damage and DPS over the `ENCOUNTER_START`…`ENCOUNTER_END` window.

```sh
go run ./tools/calibrate/cmd -log WoWCombatLog.txt -player Icey > observed.json
go test ./tools/calibrate/...
```

It only reads the file the client writes. It sends nothing to the game.

## Where this is going
1. **Observed numbers** (this tool): what really happened, per spell and per pull.
2. **Same pull in the sim:** feed the player's gear, talents, buffs and the pull's length to the sim, and record predicted per-spell ranges, crit % and DPS.
3. **Grade:** flag every spell whose sim range or crit rate falls outside what the log observed, and publish the median DPS error per spec. That error number is the accuracy claim.

The first real log (Gnomeregan, beta build 1.60.1) already shows Forever's DoT crits in practice: Rip crit on 13.6% of 110 ticks.
