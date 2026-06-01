package content

import "github.com/uinjad/AzureNights2/internal/domain/faction"

type factionDTO struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Beats string `json:"beats"`
}

type factionsFileDTO struct {
	AdvantageMult    float64      `json:"advantage_mult"`
	DisadvantageMult float64      `json:"disadvantage_mult"`
	Factions         []factionDTO `json:"factions"`
}

func loadFactions() (*faction.Table, error) {
	file, err := readJSON[factionsFileDTO]("data/factions.json")
	if err != nil {
		return nil, err
	}
	names := make(map[faction.ID]string, len(file.Factions))
	beats := make(map[faction.ID]faction.ID, len(file.Factions))
	for _, f := range file.Factions {
		names[faction.ID(f.ID)] = f.Name
		if f.Beats != "" {
			beats[faction.ID(f.ID)] = faction.ID(f.Beats)
		}
	}
	return faction.NewTable(file.AdvantageMult, file.DisadvantageMult, names, beats)
}
