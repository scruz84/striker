package common

import (
	"fmt"
	"math/rand"
)

// Shared stadium-naming patterns for the FIFA-membership countries, grouped like the club patterns in
// teamnames.go.
var (
	araStadiumPatterns  = []string{"Malaab %s", "%s Stadium"}
	balStadiumPatterns  = []string{"%s Staadion", "%s Arena"}
	dutStadiumPatterns  = []string{"%s Stadion", "Parke %s", "Estadio %s"}
	engStadiumPatterns  = []string{"%s Stadium", "%s Park", "%s Ground"}
	espStadiumPatterns  = []string{"Estadio %s", "Campo %s", "Cancha %s"}
	fraStadiumPatterns  = []string{"Stade %s", "Parc %s", "Arena %s"}
	gerStadiumPatterns  = []string{"%s Stadion", "%s Arena", "Sportpark %s"}
	grStadiumPatterns   = []string{"Stadio %s", "%s Arena"}
	indStadiumPatterns  = []string{"%s Stadium", "%s Ground", "%s Sports Complex"}
	itStadiumPatterns   = []string{"Stadio %s", "Arena %s", "Campo %s"}
	norStadiumPatterns  = []string{"%s Stadion", "%s Arena", "%s Idrettspark"}
	porStadiumPatterns  = []string{"Estadio %s", "Arena %s", "Campo %s"}
	seaStadiumPatterns  = []string{"%s Stadium", "%s Arena"}
	slavStadiumPatterns = []string{"Stadion %s", "%s Arena"}
)

// %s is replaced with a place name from placeNamesByCountry (see teamnames.go) - a stadium takes
// its host town's name the same way a club does, just combined with a stadium-style suffix
// instead of a club-style one.
var stadiumNamePatternsByCountry = map[string][]string{
	"es":  {"Estadio %s", "Campo de %s", "Ciudad Deportiva %s"},
	"en":  {"%s Stadium", "%s Park", "%s Arena", "%s Ground"},
	"de":  {"%s Stadion", "%s Arena", "%s Sportpark"},
	"fr":  {"Stade %s", "Parc %s", "Arena %s"},
	"it":  {"Stadio %s", "Arena %s", "Campo %s"},
	"pt":  {"Estadio %s", "Parque %s", "Arena %s"},
	"nl":  {"%s Stadion", "%s Arena", "Sportpark %s"},
	"se":  {"%s Arena", "%s IP", "%s Stadion"},
	"pl":  {"Stadion %s", "%s Arena"},
	"gr":  {"Stadio %s", "%s Arena"},
	"be":  {"%s Stadion", "%s Arena", "Sportpaleis %s"},
	"tr":  {"%s Stadyumu", "%s Arena"},
	"at":  {"%s Stadion", "%s Arena"},
	"sco": {"%s Park", "%s Stadium", "%s Ground"},
	"wal": {"Parc %s", "%s Park", "%s Stadium"},
	"ie":  {"%s Park", "%s Stadium"},
	"ru":  {"Stadion %s", "%s Arena"},
	"rs":  {"Stadion %s", "%s Arena"},
	"hr":  {"Stadion %s", "%s Arena"},
	"ba":  {"Stadion %s", "%s Arena"},
	"si":  {"Stadion %s", "%s Arena"},
	"mk":  {"Stadion %s", "%s Arena"},
	"me":  {"Stadion %s", "%s Arena"},
	"xk":  {"Stadiumi %s", "%s Arena"},
	"br":  {"Estadio %s", "Arena %s", "Parque %s"},
	"ar":  {"Estadio %s", "Cancha %s", "Parque %s"},
	"mx":  {"Estadio %s", "Cancha %s", "Parque %s"},
	"co":  {"Estadio %s", "Cancha %s", "Parque %s"},
	"cl":  {"Estadio %s", "Cancha %s", "Parque %s"},
	"pe":  {"Estadio %s", "Cancha %s", "Parque %s"},
	"uy":  {"Estadio %s", "Cancha %s", "Parque %s"},
	"us":  {"%s Stadium", "%s Field", "%s Arena"},
	"ca":  {"%s Stadium", "%s Field", "%s Arena"},
	"ec":  {"Estadio %s", "Cancha %s", "Parque %s"},
	"ve":  {"Estadio %s", "Polideportivo %s"},
	"py":  {"Estadio %s", "Cancha %s"},
	"bo":  {"Estadio %s", "Coliseo %s"},
	"jp":  {"%s Stadium", "%s Dome", "%s Arena"},
	"kr":  {"%s Stadium", "%s Arena"},
	"cn":  {"%s Stadium", "%s Arena"},
	"in":  {"%s Stadium", "%s Ground"},
	"th":  {"%s Stadium", "%s Arena"},
	"vn":  {"%s Stadium", "%s Arena"},
	"id":  {"Stadion %s", "%s Arena"},
	"ph":  {"%s Stadium", "%s Arena"},
	"sa":  {"Malaab %s", "%s Stadium"},
	"ir":  {"Varzeshgah %s", "%s Stadium"},
	"qa":  {"Malaab %s", "%s Stadium"},
	"ae":  {"Malaab %s", "%s Stadium"},
	"uz":  {"Stadion %s", "%s Arena"},
	"ng":  {"%s Stadium", "%s Sports Complex", "%s Arena"},
	"eg":  {"Malaab %s", "%s Stadium"},
	"gh":  {"%s Park", "%s Stadium"},
	"ci":  {"%s Stadium", "%s Arena"},
	"cm":  {"%s Stadium", "%s Arena"},
	"sn":  {"Stade %s", "%s Stade"},
	"za":  {"%s Stadium", "%s Park"},
	"ma":  {"Stade %s", "%s Stade"},
	"dz":  {"Stade %s", "%s Stade"},
	"ke":  {"%s Stadium", "%s Park"},
	"tn":  {"Stade %s", "%s Stade"},
	"tz":  {"%s Stadium", "%s Grounds"},
	"ml":  {"Stade %s", "%s Stade"},
	// UEFA
	"al":  slavStadiumPatterns,
	"ad":  espStadiumPatterns,
	"am":  slavStadiumPatterns,
	"az":  slavStadiumPatterns,
	"by":  slavStadiumPatterns,
	"bg":  slavStadiumPatterns,
	"cy":  grStadiumPatterns,
	"cz":  slavStadiumPatterns,
	"dk":  norStadiumPatterns,
	"ee":  balStadiumPatterns,
	"fo":  norStadiumPatterns,
	"fi":  balStadiumPatterns,
	"ge":  slavStadiumPatterns,
	"gi":  engStadiumPatterns,
	"hu":  slavStadiumPatterns,
	"is":  norStadiumPatterns,
	"il":  araStadiumPatterns,
	"kz":  slavStadiumPatterns,
	"lv":  balStadiumPatterns,
	"li":  gerStadiumPatterns,
	"lt":  balStadiumPatterns,
	"lu":  gerStadiumPatterns,
	"mt":  engStadiumPatterns,
	"md":  slavStadiumPatterns,
	"nir": engStadiumPatterns,
	"no":  norStadiumPatterns,
	"ro":  slavStadiumPatterns,
	"sm":  itStadiumPatterns,
	"sk":  slavStadiumPatterns,
	"ch":  gerStadiumPatterns,
	"ua":  slavStadiumPatterns,
	// CONCACAF
	"ai": engStadiumPatterns,
	"ag": engStadiumPatterns,
	"aw": dutStadiumPatterns,
	"bs": engStadiumPatterns,
	"bb": engStadiumPatterns,
	"bz": engStadiumPatterns,
	"bm": engStadiumPatterns,
	"vg": engStadiumPatterns,
	"ky": engStadiumPatterns,
	"cr": espStadiumPatterns,
	"cu": espStadiumPatterns,
	"cw": dutStadiumPatterns,
	"dm": engStadiumPatterns,
	"do": espStadiumPatterns,
	"sv": espStadiumPatterns,
	"gd": engStadiumPatterns,
	"gt": espStadiumPatterns,
	"gy": engStadiumPatterns,
	"ht": fraStadiumPatterns,
	"hn": espStadiumPatterns,
	"jm": engStadiumPatterns,
	"ms": engStadiumPatterns,
	"ni": espStadiumPatterns,
	"pa": espStadiumPatterns,
	"pr": espStadiumPatterns,
	"kn": engStadiumPatterns,
	"lc": engStadiumPatterns,
	"vc": engStadiumPatterns,
	"sr": dutStadiumPatterns,
	"tt": engStadiumPatterns,
	"tc": engStadiumPatterns,
	"vi": engStadiumPatterns,
	// AFC
	"af": slavStadiumPatterns,
	"au": engStadiumPatterns,
	"bh": araStadiumPatterns,
	"bd": indStadiumPatterns,
	"bt": indStadiumPatterns,
	"bn": seaStadiumPatterns,
	"kh": seaStadiumPatterns,
	"tw": seaStadiumPatterns,
	"gu": engStadiumPatterns,
	"hk": engStadiumPatterns,
	"iq": araStadiumPatterns,
	"jo": araStadiumPatterns,
	"kw": araStadiumPatterns,
	"kg": slavStadiumPatterns,
	"la": seaStadiumPatterns,
	"lb": araStadiumPatterns,
	"mo": porStadiumPatterns,
	"my": seaStadiumPatterns,
	"mv": indStadiumPatterns,
	"mn": slavStadiumPatterns,
	"mm": seaStadiumPatterns,
	"np": indStadiumPatterns,
	"kp": seaStadiumPatterns,
	"om": araStadiumPatterns,
	"pk": indStadiumPatterns,
	"ps": araStadiumPatterns,
	"sg": seaStadiumPatterns,
	"lk": indStadiumPatterns,
	"sy": araStadiumPatterns,
	"tj": slavStadiumPatterns,
	"tl": porStadiumPatterns,
	"tm": slavStadiumPatterns,
	"ye": araStadiumPatterns,
	// CAF
	"ao": porStadiumPatterns,
	"bj": fraStadiumPatterns,
	"bw": engStadiumPatterns,
	"bf": fraStadiumPatterns,
	"bi": fraStadiumPatterns,
	"cv": porStadiumPatterns,
	"cf": fraStadiumPatterns,
	"td": fraStadiumPatterns,
	"km": fraStadiumPatterns,
	"cg": fraStadiumPatterns,
	"cd": fraStadiumPatterns,
	"dj": fraStadiumPatterns,
	"gq": espStadiumPatterns,
	"er": araStadiumPatterns,
	"sz": engStadiumPatterns,
	"et": engStadiumPatterns,
	"ga": fraStadiumPatterns,
	"gm": engStadiumPatterns,
	"gn": fraStadiumPatterns,
	"gw": porStadiumPatterns,
	"ls": engStadiumPatterns,
	"lr": engStadiumPatterns,
	"ly": araStadiumPatterns,
	"mg": fraStadiumPatterns,
	"mw": engStadiumPatterns,
	"mr": araStadiumPatterns,
	"mu": engStadiumPatterns,
	"mz": porStadiumPatterns,
	"na": engStadiumPatterns,
	"ne": fraStadiumPatterns,
	"rw": fraStadiumPatterns,
	"st": porStadiumPatterns,
	"sc": engStadiumPatterns,
	"sl": engStadiumPatterns,
	"so": araStadiumPatterns,
	"ss": engStadiumPatterns,
	"sd": araStadiumPatterns,
	"tg": fraStadiumPatterns,
	"ug": engStadiumPatterns,
	"zm": engStadiumPatterns,
	"zw": engStadiumPatterns,
	// OFC (folded into the Asian region, see RegionForCountry)
	"as": engStadiumPatterns,
	"ck": engStadiumPatterns,
	"fj": engStadiumPatterns,
	"nc": fraStadiumPatterns,
	"nz": engStadiumPatterns,
	"pg": engStadiumPatterns,
	"ws": engStadiumPatterns,
	"sb": engStadiumPatterns,
	"pf": fraStadiumPatterns,
	"to": engStadiumPatterns,
	"vu": fraStadiumPatterns,
}

// RandomStadiumName generates a stadium name for country in the same spirit as RandomTeamName -
// an invented place name (drawn from the same pool RandomTeamName uses, so a club and its ground
// plausibly share a hometown) combined with a stadium-style naming pattern typical of that
// country (e.g. "Estadio Villanueva", "Ashford Park", "Nordhausen Stadion"). Falls back to the
// "en" pools for an unrecognized country, same as RandomTeamName.
func RandomStadiumName(country string) string {
	places, ok := placeNamesByCountry[country]
	if !ok {
		places = placeNamesByCountry["en"]
	}
	patterns, ok := stadiumNamePatternsByCountry[country]
	if !ok {
		patterns = stadiumNamePatternsByCountry["en"]
	}
	place := places[rand.Intn(len(places))]
	pattern := patterns[rand.Intn(len(patterns))]
	return fmt.Sprintf(pattern, place)
}
