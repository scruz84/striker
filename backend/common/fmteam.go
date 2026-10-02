package common

func GetLateralDefenseValue(player Player) int {
	return (player.Lateral + player.Defense) / 2
}

func GetLateralMidfieldValue(player Player) int {
	return (player.Lateral + GetMidfieldValue(player)) / 2
}

func GetLateralStrikerValue(player Player) int {
	return (player.Lateral + GetStrikerValue(player)) / 2
}

func GetStrikerValue(player Player) int {
	// Shot counts twice as much as Dribbling or Pass - it's the finishing stat that actually
	// defines a frontman, while the other two are supporting skills. A plain three-way average
	// let a mediocre passer's otherwise-strong Shot/Dribbling drag down to - or below - an
	// unrelated position's single raw stat (see BestCorePosition), mislabeling clear finishers as
	// goalkeepers or defenders just because their Pass was weak.
	return (player.Shot*2 + player.Dribbling + player.Pass + 2) / 4
}

func GetMidfieldValue(player Player) int {
	return (player.Midfield + player.Pass) / 2
}
