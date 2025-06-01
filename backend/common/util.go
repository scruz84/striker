package common

import (
	"encoding/json"
	"math/rand"
)

// https://getavataaars.com/
var eyeTypes = [...]string{"Default", "EyeRoll", "Happy", "Side", "Squint", "Surprised", "Wink", "WinkWacky"}
var eyeBrow = [...]string{"Angry", "AngryNatural", "Default", "DefaultNatural", "FlatNatural", "RaisedExcited", "RaisedExcitedNatural",
	"SadConcerned", "SadConcernedNatural"}
var facialHair = [...]string{"Blank", "BeardMedium", "BeardLight", "BeardMajestic", "MoustacheFancy", "MoustacheMagnum"}
var facialHairColor = [...]string{"Auburn", "Black", "Blonde", "BlondeGolden", "Brown", "BrownDark",
	"Blue", "Red"}
var hairColor = [...]string{"Auburn", "Black", "Blonde", "BlondeGolden", "Brown", "BrownDark", "PastelPink", "Blue",
	"Platinum", "Red", "SilverGray"}
var mouth = [...]string{"Concerned", "Default", "Disbelief", "Eating", "Grimace", "Sad", "Serious"}
var skin = [...]string{"Tanned", "Black", "Yellow", "Pale", "Light", "Brown", "DarkBrown"}

func RandomAvatar() PlayerAvatar {
	return PlayerAvatar{AvatarStyle: "Transparent", AccessoriesType: "Blank", ClotheType: "ShirtCrewNeck",
		EyeType: randomEyeType(), HairColor: randomHairColor(), FacialHairType: randomFacialHair(),
		FacialHairColor: randomFacialHairColor(), SkinColor: randomSkin(), EyebrowType: randomEyeBrowType()}
}

func RandomAvatarAsString() string {
	avatar := RandomAvatar()
	b, _ := json.Marshal(avatar)
	return string(b)
}

func GetAvatarFromString(avatar string) PlayerAvatar {
	var playerAvatar PlayerAvatar
	json.Unmarshal([]byte(avatar), &playerAvatar)
	return playerAvatar
}

func randomEyeType() string {
	return eyeTypes[rand.Intn(len(eyeTypes))]
}

func randomEyeBrowType() string {
	return eyeBrow[rand.Intn(len(eyeBrow))]
}

func randomHairColor() string {
	return hairColor[rand.Intn(len(hairColor))]
}

func randomFacialHair() string {
	return facialHair[rand.Intn(len(facialHair))]
}

func randomFacialHairColor() string {
	return facialHairColor[rand.Intn(len(facialHairColor))]
}

func randomMouth() string {
	return mouth[rand.Intn(len(mouth))]
}


func randomSkin() string {
	return skin[rand.Intn(len(skin))]
}
