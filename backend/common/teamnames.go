package common

import (
	"fmt"
	"math/rand"
)

// Invented place-like names per seeded country (see SeededCountries) - deliberately
// not real cities or club names, just plausible-sounding ones in that country's style, combined
// with a locally-flavored naming pattern in RandomTeamName.
var placeNamesByCountry = map[string][]string{
	"es": {
		"Villanueva", "Montoro", "Peñarroya", "Arandilla", "Bellavista", "Torreblanca",
		"Puigverd", "Sanlázaro", "Almendral", "Riofrío", "Casasola", "Montseny",
		"Valdemora", "Fuenteagria", "Sierraverde", "Costamar",
	},
	"en": {
		"Ashford", "Millbrook", "Kingsley", "Thornbury", "Whitfield", "Eastwood",
		"Broadfield", "Rushmoor", "Hartwell", "Denholm", "Overbridge", "Fenwick",
		"Radcombe", "Marston", "Aldergate", "Ravensworth",
	},
	"de": {
		"Nordhausen", "Falkenberg", "Rheinbach", "Bergstadt", "Waldheim", "Steinfurt",
		"Lindenau", "Grunwald", "Eisenbach", "Hohenfeld", "Rotenburg", "Dornstadt",
		"Kaltenbrunn", "Sonnenberg", "Talheim", "Bruckstadt",
	},
	"fr": {
		"Montval", "Beaulieu", "Clairefontaine", "Vernoux", "Roussel", "Charmont",
		"Belrive", "Fontenay", "Lauris", "Valcourt", "Aubignac", "Montrose",
		"Rochebrune", "Grandchamp", "Servignac", "Aiglemont",
	},
	"it": {
		"Montefiore", "Vallescura", "Torrenova", "Campolungo", "Riobianco", "Castelmare",
		"Pratolungo", "Collegrande", "Fontanalta", "Serrapiana", "Piandoro", "Montesanto",
		"Casalvento", "Bellaripa",
	},
	"pt": {
		"Vilanova", "Montesinho", "Ribalonga", "Casaltorto", "Fontelas", "Serracosta",
		"Valmonte", "Pedraverde", "Riodouro", "Montebranco", "Campolide", "Alvorada",
		"Bragamonte", "Ponteverde",
	},
	"nl": {
		"Nieuwdorp", "Bergenhoek", "Vlietburg", "Zuidwoud", "Molenveld", "Hoogland",
		"Waterburg", "Duinzicht", "Rijnstede", "Veenoord", "Bosrand", "Kerkhoven",
	},
	"se": {
		"Norrbacka", "Sjövik", "Björkdal", "Lindholm", "Åsstrand", "Fjällby",
		"Skogshem", "Strandvik", "Bergfors", "Kvarnbo", "Solberga", "Furulund",
	},
	"pl": {
		"Nowogrod", "Zielonka", "Bialobrzeg", "Wolomin", "Krasnodwor", "Podlesie",
		"Grabowiec", "Lipowa", "Sosnowiec", "Debowa", "Rzeczno", "Kalinow",
	},
	"gr": {
		"Kastroupoli", "Petralona", "Argostoli", "Xylokastro", "Vryssoula", "Ammoudia",
		"Filotheo", "Kalamaki", "Drymona", "Elatochori", "Ampeli", "Kryoneri",
	},
	"be": {
		"Veldburcht", "Molenhoven", "Boshoek", "Lindendal", "Hoogveld", "Groenbrug",
		"Steenhaven", "Dalbeek", "Rozendal", "Kastelein", "Vlierhof", "Berkenveld",
	},
	"tr": {
		"Karatas", "Guneykoy", "Yesilova", "Akdere", "Bozdag", "Sarikale",
		"Beyazkent", "Golbasi", "Kizilcay", "Mavikoy", "Topraktepe", "Sutasi",
	},
	"at": {
		"Bergheim", "Talmoos", "Sonnental", "Wiesenberg", "Felsbach", "Lindhof",
		"Gruenwald", "Seebrunn", "Alpenfeld", "Rosenbach", "Falkenstein", "Tannenbach",
	},
	"sco": {
		"Glenmor", "Kirkbrae", "Lochend", "Strathburn", "Dunross", "Craigmuir",
		"Ardglen", "Benloch", "Clachan", "Tayburn", "Thornglen", "Mossburn",
		"Glenavon", "Braemuir",
	},
	"wal": {
		"Bryneira", "Glanmor", "Cwmrhos", "Pentraeth", "Llanwern", "Aberwen",
		"Dolgarth", "Trefelin", "Maesglas", "Felinwen", "Craigfan", "Rhosgoch",
		"Brynhafod", "Cefnbryn",
	},
	"ie": {
		"Ballyclough", "Kilross", "Glenmara", "Dunbrook", "Rathmoyle", "Drumlea",
		"Castleard", "Clonmara", "Kilbrennan", "Lissadell", "Ardglass", "Moycullen",
		"Rossbeg", "Tullyard",
	},
	"ru": {
		"Novograd", "Belozersk", "Starodub", "Krasnoluch", "Severnoye", "Zarechye",
		"Svetlogorsk", "Bolshevo", "Yasnogorsk", "Volkovo", "Lesnoye", "Chernavino",
		"Zolotov", "Beryozka",
	},
	"rs": {
		"Vranograd", "Belopolje", "Zlatkovo", "Jablanik", "Moravci", "Krajinovo",
		"Timocka", "Sumadici", "Banatsko", "Dunavec", "Moravgrad", "Kolubarsko",
		"Sremac", "Kosmaj",
	},
	"hr": {
		"Krapinsko", "Zelenbrod", "Kamengrad", "Suncanik", "Vinodolje", "Posavgrad",
		"Istragrad", "Dravica", "Slavonac", "Kvarnerac", "Zagorac", "Medvedgrad",
		"Velebitska", "Rijecanka",
	},
	"ba": {
		"Vrelograd", "Kraljevac", "Vrbasko", "Drinovo", "Zelenpolje", "Kamenbrod",
		"Sanovica", "Neretvica", "Miljacko", "Bosnagrad", "Trebinjac", "Hercegac",
		"Vlasenik", "Rudopolje",
	},
	"si": {
		"Kamnograd", "Dravodol", "Savagrad", "Zelenjak", "Bistrovec", "Vrhpolje",
		"Soteskina", "Planinica", "Loskovo", "Travnica", "Gorje", "Mirnovec",
		"Studenci", "Vipavac",
	},
	"mk": {
		"Vardarsko", "Pelagonka", "Ohridsko", "Skopsko", "Bitolsko", "Prilepec",
		"Kumanovec", "Strumicka", "Tetovec", "Gostivarka", "Kavadarec", "Debarsko",
		"Resensko", "Veleska",
	},
	"me": {
		"Crnagora", "Boksko", "Durmitorac", "Lovcenac", "Skadarsko", "Zetagrad",
		"Moracko", "Bjelopavlic", "Komarnica", "Kotorac", "Budvanka", "Cetinjac",
		"Niksicko", "Herceggrad",
	},
	"xk": {
		"Dukagjini", "Sharrplan", "Drenasi", "Pejani", "Bjeshkova", "Istogu",
		"Lumbardhi", "Rugova", "Decani", "Podujeva", "Zymberi", "Lugina",
	},
	"br": {
		"Novaterra", "Serrafundo", "Riopreto", "Campolindo", "Vistalegre", "Montenovo",
		"Vargedo", "Pedraverde", "Fontenova", "Riograna", "Serramar", "Valfundo",
		"Bomjardim", "Portoclaro",
	},
	"ar": {
		"Nuevaflores", "Rioseco", "Villamora", "Puertoclaro", "Cerroverde", "Lomasnegras",
		"Aguasclaras", "Sanmartin", "Vistahermosa", "Campolargo", "Villaester", "Fortinsol",
	},
	"mx": {
		"Vallehondo", "Cerroalto", "Riobravo", "Villaflor", "Montenegro", "Pueblorico",
		"Sancarlos", "Casagrande", "Tierrasanta", "Losllanos", "Aguadulce", "Puertosol",
	},
	"co": {
		"Villaverde", "Cascadales", "Altoprado", "Riohondo", "Valledorado", "Cerroblanco",
		"Puertosol", "Llanoverde", "Nuevaesperanza", "Bosquealto", "Montebello", "Sabanaflor",
	},
	"cl": {
		"Cerroazul", "Vallefrio", "Puertoluna", "Lagosur", "Montenevado", "Riobello",
		"Costaverde", "Altoandino", "Pinomar", "Sollago", "Vinamora", "Arenacosta",
	},
	"pe": {
		"Riomonte", "Vallesagrado", "Cerrodorado", "Puertoinca", "Tierrablanca", "Nevadoalto",
		"Sierradorada", "Costanorte", "Aguasfrias", "Montañarica", "Solandino", "Vallenieve",
	},
	"uy": {
		"Costaeste", "Praderaverde", "Riolindo", "Cerropampa", "Villamarina", "Solmontano",
		"Campodulce", "Bahiaclara", "Puntafuerte", "Lagosereno", "Villaeste", "Costadorada",
	},
	"us": {
		"Brookhaven", "Lakeside", "Riverbend", "Cedarfield", "Hillcrest", "Meadowpark",
		"Stonebridge", "Westfield", "Clearwater", "Pinehurst", "Fairview", "Oakridge",
	},
	"ca": {
		"Mapleridge", "Northgate", "Frostvale", "Cedarlake", "Birchmont", "Silverpine",
		"Lakeharbour", "Winterfield", "Snowridge", "Goldenvale", "Bayshore", "Timberhill",
	},
	"ec": {
		"Riosol", "Vallenorte", "Cerroluz", "Puertoazul", "Montesagrado", "Costaflor",
		"Andesalto", "Selvaverde", "Bahiadelsol", "Tierrasol", "Valleoro", "Riomanabi",
	},
	"ve": {
		"Montellano", "Riomiel", "Vallealto", "Cerromonte", "Playaverde", "Sabanagrande",
		"Lagoverde", "Puertoluz", "Floresta", "Costadorada", "Altollano", "Valleclaro",
	},
	"py": {
		"Villasol", "Yerbamar", "Cerroguazu", "Riobello", "Monteclaro", "Campoverde",
		"Costaazul", "Taraguira", "Lomabella", "Guaviral", "Fortinluz", "Esperanzal",
	},
	"bo": {
		"Altoplano", "Cerroverde", "Vallenuevo", "Montealto", "Riofrio", "Lagoazul",
		"Sierranieve", "Puebloalto", "Collasuyo", "Pampaverde", "Tierrafria", "Yungasol",
	},
	"jp": {
		"Kazehara", "Sorayama", "Tsukikawa", "Hoshizaki", "Yumemori", "Kagetaki",
		"Minamizora", "Akaneyama", "Fuyuzaki", "Harukawa", "Midorisaka", "Yozora",
	},
	"kr": {
		"Bukhan", "Namsan", "Donggang", "Seorak", "Baegok", "Hwaseong",
		"Cheongsan", "Haeundae", "Jindo", "Sanchon", "Gilju", "Yeonhwa",
	},
	"cn": {
		"Qinghe", "Baiyun", "Jinshan", "Xihu", "Longtan", "Fengming",
		"Yunlong", "Shuangqing", "Meilin", "Zhaoyang", "Hongqiao", "Tianyu",
	},
	"in": {
		"Nabagram", "Ramnagar", "Shantipur", "Vijaynagar", "Suryapur", "Chandpur",
		"Meenakshipuram", "Gopalganj", "Anandpur", "Kiranpur", "Ratnagiri", "Dharampur",
	},
	"th": {
		"Banphai", "Suandok", "Kohsai", "Nongsai", "Maenam", "Thungthong",
		"Wiangchai", "Bangrak", "Chomthong", "Srimuang", "Klongluang", "Donmuang",
	},
	"vn": {
		"Xuanha", "Thanhbinh", "Haiha", "Longan", "Phuoclong", "Trungson",
		"Bacha", "Vinhloc", "Tanphu", "Dongxuan", "Namdinh", "Thaibinh",
	},
	"id": {
		"Sukamaju", "Cintapuri", "Bukitindah", "Sumbertani", "Mekarjaya", "Tanjungsari",
		"Karangasri", "Wonorejo", "Batuputih", "Pantaimas", "Cahayabaru", "Sumberjaya",
	},
	"ph": {
		"Bagongbayan", "Malinao", "Kalayaan", "Marikit", "Bantayan", "Ligaya",
		"Malaya", "Dalisay", "Tagumpay", "Mabini", "Bayanihan", "Silangan",
	},
	"sa": {
		"Wadinoor", "Rasalkhair", "Nakhilan", "Baharia", "Sahran", "Qasrnur",
		"Rimaal", "Zahran", "Hijran", "Najdiya", "Yanburi", "Dahnaa",
	},
	"ir": {
		"Golshahr", "Shirvan", "Baharan", "Ferdows", "Sepidan", "Zaringol",
		"Nasrabad", "Kohsar", "Deylaman", "Sabzevaran", "Mehrshahr", "Delfan",
	},
	"qa": {
		"Rawdatal", "Zubarahi", "Dukhanil", "Khawri", "Faridan", "Simaisma",
		"Luqtawi", "Wakrahiya", "Mesaieedi", "Umsalal", "Dafnawi", "Ruwaysi",
	},
	"ae": {
		"Wahatnoor", "Ramlahadi", "Khawrsalam", "Faydasahra", "Najdaliya", "Suhailiya",
		"Marwaniya", "Dibbashams", "Hatasun", "Burjaliya", "Sahlaniya", "Qasriya",
	},
	"uz": {
		"Gulbogh", "Toshkoy", "Chinorabad", "Sardoba", "Oltinsoy", "Bogibolo",
		"Siryop", "Farhodtepa", "Gulrang", "Mehnatobod", "Bakhtiyor", "Chamanzor",
	},
	"ng": {
		"Ajumobi", "Odusanya", "Egbedun", "Okwudili", "Adukwu", "Bakori",
		"Danlaso", "Yarimatu", "Ogbunike", "Nnewichi", "Sabongari", "Igboefon",
	},
	"eg": {
		"Wadisalam", "Nasriya", "Karnakiya", "Zahraa", "Fayoumi", "Aswaniya",
		"Dahabiya", "Ramliya", "Sahrawiya", "Nilhira", "Basratine", "Marsakhal",
	},
	"gh": {
		"Adom", "Nkrofu", "Asantekrom", "Obuasu", "Akwadum", "Sekonda",
		"Tamalu", "Elmira", "Kumbo", "Bawuku", "Nsawan", "Tafowa",
	},
	"ci": {
		"Bouaka", "Yamousso", "Daloasu", "Abinou", "Korhogou", "Gagnoua",
		"Manu", "Divokro", "Soubrekro", "Bonouakro", "Adzope", "Sassandrou",
	},
	"cm": {
		"Ebolowu", "Ngaounda", "Kribiya", "Bafousso", "Dschanga", "Garouma",
		"Mamfea", "Edeya", "Bertouma", "Foumbana", "Nkongsu", "Bafiaka",
	},
	"sn": {
		"Thiesa", "Kaolacka", "Ziguina", "Diourba", "Toubama", "Rufisca",
		"Lougama", "Sailouis", "Koldama", "Tambama", "Kaffrina", "Podorou",
	},
	"za": {
		"Vryburgh", "Springfontein", "Rooidal", "Bloemhoek", "Sonop", "Kaaprivier",
		"Duinveld", "Groenvlei", "Uitspan", "Windrand", "Klipdrift", "Vaalhoek",
	},
	"ma": {
		"Ouedzam", "Ifrania", "Chefchana", "Tazama", "Berkana", "Nadora",
		"Ksarnour", "Taroudia", "Settana", "Benimar", "Zagorata", "Tifelt",
	},
	"dz": {
		"Tlemcia", "Bejaya", "Setifa", "Blidama", "Annabia", "Ghardana",
		"Skikama", "Mostagana", "Ouargama", "Batnama", "Bordjina", "Tissemna",
	},
	"ke": {
		"Naivama", "Kerichu", "Nyeria", "Machakama", "Kitalea", "Garissama",
		"Malinda", "Eldorana", "Voia", "Nakurama", "Embua", "Kajiada",
	},
	"tn": {
		"Ouedsafran", "Jebelnoor", "Hammamzeit", "Menzelqods", "Borjlila", "Ksarnejma",
		"Sidihana", "Rasalnaj", "Dartlazur", "Aynzeitoun", "Beniaman", "Zaytounia",
	},
	"tz": {
		"Mlimasafi", "Ziwaupepo", "Mashamba", "Mwambaoni", "Kijiweni", "Msituupya",
		"Pwaniazuri", "Nyikani", "Baharini", "Vijijini", "Milimani", "Dalalini",
	},
	"ml": {
		"Sigirato", "Bancouma", "Faradjan", "Kolokoro", "Dougoufie", "Samanfara",
		"Banikara", "Segoubani", "Tomboura", "Kayesso", "Sikasana", "Niorokoro",
	},
}

// %s is replaced with a place name from placeNamesByCountry.
var teamNamePatternsByCountry = map[string][]string{
	"es":  {"Real %s", "Atlético %s", "%s CF", "%s FC", "Deportivo %s", "Unión %s", "%s Balompié"},
	"en":  {"%s United", "%s City", "%s Town", "%s Rovers", "%s Wanderers", "%s Athletic", "%s Rangers"},
	"de":  {"%s FC", "SV %s", "TSV %s", "FC %s", "%s Sportfreunde", "Borussia %s"},
	"fr":  {"%s FC", "AS %s", "Olympique %s", "Racing %s", "Stade %s", "%s United"},
	"it":  {"%s FC", "AC %s", "AS %s", "US %s", "%s Calcio", "Virtus %s", "Polisportiva %s"},
	"pt":  {"%s FC", "Sporting %s", "%s SC", "Uniao %s", "Atletico %s", "%s Futebol Clube", "Vitoria %s"},
	"nl":  {"%s FC", "SC %s", "%s United", "VV %s", "%s Boys"},
	"se":  {"%s IF", "%s FF", "IFK %s", "%s BK"},
	"pl":  {"%s FC", "KS %s", "%s United", "Polonia %s"},
	"gr":  {"%s FC", "AO %s", "PAE %s", "%s AE"},
	"be":  {"%s FC", "KV %s", "RSC %s", "Royal %s", "Union %s", "%s United"},
	"tr":  {"%s FK", "%s Spor", "Genclik %s", "%s Genclerbirligi", "Buyuksehir %s"},
	"at":  {"SK %s", "%s FC", "SV %s", "Austria %s", "Rapid %s"},
	"sco": {"%s FC", "%s United", "%s Athletic", "%s Thistle", "%s Rovers", "Heart of %s"},
	"wal": {"%s Town", "%s FC", "%s AFC", "Clwb %s", "%s Rovers"},
	"ie":  {"%s FC", "%s United", "%s Athletic", "%s Rovers", "%s City"},
	"ru":  {"FC %s", "Dinamo %s", "Spartak %s", "Lokomotiv %s", "Zenit %s", "CSKA %s"},
	"rs":  {"FK %s", "%s FC", "Crvena %s", "Partizan %s", "%s United"},
	"hr":  {"NK %s", "%s FC", "HNK %s", "Dinamo %s", "%s United"},
	"ba":  {"FK %s", "NK %s", "%s FC", "%s United"},
	"si":  {"NK %s", "%s FC", "%s United"},
	"mk":  {"FK %s", "%s FC", "%s United"},
	"me":  {"FK %s", "%s FC", "%s United"},
	"xk":  {"FC %s", "KF %s", "%s United"},
	"br":  {"%s FC", "Esporte Clube %s", "Atletico %s", "%s AC", "Gremio %s", "%s Futebol Clube", "Sociedade %s"},
	"ar":  {"%s FC", "Club Atletico %s", "Deportivo %s", "%s Juniors", "Atletico %s"},
	"mx":  {"%s FC", "Deportivo %s", "Atletico %s", "Club %s", "%s Atletico"},
	"co":  {"%s FC", "Deportivo %s", "Atletico %s", "Independiente %s"},
	"cl":  {"%s FC", "Deportivo %s", "Union %s", "Club %s"},
	"pe":  {"%s FC", "Deportivo %s", "Atletico %s", "Union %s"},
	"uy":  {"%s FC", "Club %s", "Deportivo %s", "Atletico %s"},
	"us":  {"%s FC", "%s United", "%s SC", "Real %s"},
	"ca":  {"%s FC", "%s United", "%s SC"},
	"ec":  {"%s FC", "Deportivo %s", "Independiente %s", "Club %s"},
	"ve":  {"%s FC", "Deportivo %s", "Atletico %s", "Union %s", "Club %s"},
	"py":  {"%s FC", "Club %s", "Deportivo %s", "Sportivo %s", "General %s"},
	"bo":  {"%s FC", "Club %s", "Deportivo %s", "Real %s", "Atletico %s"},
	"jp":  {"%s FC", "%s United", "%s SC", "Vento %s", "%s Vulcan", "%s Fiore"},
	"kr":  {"%s FC", "%s United", "%s City"},
	"cn":  {"%s FC", "%s United"},
	"in":  {"%s FC", "%s United", "%s SC"},
	"th":  {"%s FC", "%s United"},
	"vn":  {"%s FC", "%s United"},
	"id":  {"Persatuan %s", "%s FC", "PS %s"},
	"ph":  {"%s FC", "%s United"},
	"sa":  {"Al %s", "%s FC", "Nadi %s"},
	"ir":  {"%s FC", "%s SC"},
	"qa":  {"Al %s", "%s SC", "Nadi %s", "%s Sports Club"},
	"ae":  {"Al %s", "%s FC", "Nadi %s", "%s Club"},
	"uz":  {"%s FK", "Dinamo %s", "%s United", "FK %s", "Lokomotiv %s"},
	"ng":  {"%s FC", "%s United", "%s Warriors", "Royal %s", "%s Stars", "%s City"},
	"eg":  {"Nadi %s", "%s SC", "Al %s"},
	"gh":  {"%s FC", "%s United", "%s Stars"},
	"ci":  {"%s FC", "%s United", "%s AC"},
	"cm":  {"%s FC", "Union %s", "%s AC"},
	"sn":  {"%s FC", "Casa %s", "%s AC"},
	"za":  {"%s FC", "%s United", "%s City"},
	"ma":  {"%s AC", "%s FC", "Chabab %s"},
	"dz":  {"%s FC", "USM %s", "Chabab %s"},
	"ke":  {"%s FC", "%s United", "%s Stars"},
	"tn":  {"Club %s", "Etoile %s", "Union %s", "Esperance %s", "%s SC"},
	"tz":  {"%s FC", "Young %s", "%s United", "%s Stars"},
	"ml":  {"%s AC", "Stade %s", "Union %s", "%s FC"},
}

// RandomTeamName generates a realistic-sounding club name for country - an invented place name
// combined with a naming pattern typical of that country's football culture (e.g. "Real
// Villanueva", "Ashford United", "SV Nordhausen"), replacing the old placeholder scheme of just
// appending "Team" to a random person's first name. Falls back to the "en" pools for an
// unrecognized country rather than producing an empty name.
func RandomTeamName(country string) string {
	places, ok := placeNamesByCountry[country]
	if !ok {
		places = placeNamesByCountry["en"]
	}
	patterns, ok := teamNamePatternsByCountry[country]
	if !ok {
		patterns = teamNamePatternsByCountry["en"]
	}
	place := places[rand.Intn(len(places))]
	pattern := patterns[rand.Intn(len(patterns))]
	return fmt.Sprintf(pattern, place)
}
