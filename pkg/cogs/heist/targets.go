package heist

import "encoding/json"

var (
	defaultTargets = map[string]*Target{
		"Bouncy Castle": {
			ID:       "Bouncy Castle",
			CrewSize: 17,
			Success:  4.25,
			Vault:    167000,
			VaultMax: 167000,
		},
		"Fault Towers": {
			ID:       "Faulty Towers",
			CrewSize: 39,
			Success:  1.86,
			Vault:    448000,
			VaultMax: 448000,
		},
		"Fort Knobs": {
			ID:       "Fort Knobs",
			CrewSize: 14,
			Success:  5.2,
			Vault:    133000,
			VaultMax: 133000,
		},
		"Gobbo Campus": {
			ID:       "Gobbo Campus",
			CrewSize: 21,
			Success:  3.5,
			Vault:    213000,
			VaultMax: 213000,
		},
		"Gobboton": {
			ID:       "Gobboton",
			CrewSize: 11,
			Success:  6.75,
			Vault:    101000,
			VaultMax: 101000,
		},
		"Goblin Forest": {
			ID:       "Goblin Forest",
			CrewSize: 2,
			Success:  29.3,
			Vault:    16000,
			VaultMax: 16000,
		},
		"Goblin Gauntlet": {
			ID:       "Goblin Gauntlet",
			CrewSize: 8,
			Success:  9.5,
			Vault:    71000,
			VaultMax: 71000,
		},
		"Goblin Outpost": {
			ID:       "Goblin Outpost",
			CrewSize: 3,
			Success:  20.65,
			Vault:    24000,
			VaultMax: 24000,
		},
		"Megamansion": {
			ID:       "Megamansion",
			CrewSize: 44,
			Success:  1.64,
			Vault:    512000,
			VaultMax: 512000,
		},
		"Obsidian Tower": {
			ID:       "Obsidian Tower",
			CrewSize: 29,
			Success:  2.49,
			Vault:    314000,
			VaultMax: 314000,
		},
		"P.e.k.k.a's Playhouse": {
			ID:       "P.e.k.k.a's Playhouse",
			CrewSize: 49,
			Success:  1.46,
			Vault:    598000,
			VaultMax: 598000,
		},
		"Queen's Gambit": {
			ID:       "Queen's Gambit",
			CrewSize: 34,
			Success:  2.15,
			Vault:    379000,
			VaultMax: 379000,
		},
		"Rocky Fort": {
			ID:       "Rocky Fort",
			CrewSize: 5,
			Success:  14.5,
			Vault:    42000,
			VaultMax: 42000,
		},
		"Sherbet Towers": {
			ID:       "Sherbet Towers",
			CrewSize: 55,
			Success:  1.31,
			Vault:    688000,
			VaultMax: 688000,
		},
		"Walls Of Steel": {
			ID:       "Walls Of Steel",
			CrewSize: 25,
			Success:  2.91,
			Vault:    263000,
			VaultMax: 263000,
		},
	}
)

// Target is a target of a heist.
type Target struct {
	ID       string  `json:"_id" bson:"_id"`
	CrewSize int64   `json:"crew" bson:"crew"`
	Success  float64 `json:"success" bson:"success"`
	Vault    int64   `json:"vault" bson:"vault"`
	VaultMax int64   `json:"vault_max" bson:"vault_max"`
}

// NewTarget creates a new target for a heist
func NewTarget(id string, maxCrewSize int64, success float64, vaultCurrent int64, maxVault int64) *Target {
	target := Target{
		ID:       id,
		CrewSize: maxCrewSize,
		Success:  success,
		Vault:    vaultCurrent,
		VaultMax: maxVault,
	}
	return &target

}

// String returns a string representation of the target.
func (t *Target) String() string {
	out, _ := json.Marshal(t)
	return string(out)
}
