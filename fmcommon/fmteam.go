package fmcommon

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
	return (player.Shot + player.Pass + player.Dribbling) / 3
}

func GetMidfieldValue(player Player) int {
	return (player.Midfield + player.Pass) / 2
}
