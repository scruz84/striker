package common

import "math/rand"

var eyeTypes = [...]string{"Default", "EyeRoll", "Happy", "Side", "Squint", "Surprised", "Wink", "WinkWacky"}

func randomAvatar() PlayerAvatar {
	return PlayerAvatar{AvatarStyle: "Transparent", AccessoriesType: "Blank", ClotheType: "ShirtCrewNeck",
		EyeType: randomEyeType()}
}

func randomEyeType() string {
	return eyeTypes[rand.Intn(len(eyeTypes))]
}
