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
	// UEFA
	"al": {
		"Bregdeti", "Lumëra", "Shkodrina", "Vlorëza", "Korçava", "Beratsi", "Gjirova", "Durrësi",
		"Pogradeci", "Tomorra", "Kavajëz", "Lezhova", "Fierina", "Sarandor", "Mirdita", "Elbasani",
	},
	"ad": {
		"Canillor", "Ordinova", "Encampa", "Massanova", "Pasdelacosta", "Aixovall", "Sant Julià",
		"Escaldor", "Vallirà", "Arinsalo", "Llorts", "Soldeu", "Pal Sant", "Ransol", "Incles",
		"Engolasters",
	},
	"am": {
		"Araratik", "Gyumrin", "Vanadzor", "Echmiadzor", "Sevanik", "Kapanik", "Dilijanik", "Goris",
		"Artashen", "Masisik", "Hrazdanik", "Ashtarak", "Yeghegnik", "Abovyanik", "Gavar", "Sisian",
	},
	"az": {
		"Karabakhan", "Shirvanik", "Ganjabad", "Sumqayit", "Lankaran", "Shekiabad", "Mingachev",
		"Naftalan", "Qubadli", "Zaqatala", "Salyanik", "Khachmaz", "Yevlakh", "Agdamlar", "Gobustan",
		"Shamkir",
	},
	"by": {
		"Novopolsk", "Zaslavl", "Mirsk", "Brestovo", "Polotsk", "Grodnovo", "Vitebsko", "Nesvizh",
		"Lidovo", "Molodechno", "Borisovka", "Slutsk", "Mozyrsk", "Orshansk", "Pinsk", "Kobrinsk",
	},
	"bg": {
		"Rilska", "Stara Planina", "Chernomor", "Plovdivsko", "Varnenets", "Burgasko", "Ruse Dunav",
		"Pirinets", "Trakiya", "Balkanets", "Rodopa", "Sliven", "Haskovo", "Vidinska", "Pernishka",
		"Shumen",
	},
	"cy": {
		"Troodos", "Kyrenia Bay", "Larnakos", "Paphiakos", "Limassolia", "Ayios Tychon",
		"Xylotymbou", "Dherynia", "Polis Bay", "Kokkinos", "Akamasia", "Lefkaritis", "Pyrgos Cape",
		"Agros", "Lysos", "Salamina",
	},
	"cz": {
		"Novohrad", "Zelenov", "Vltavice", "Brnovice", "Sedlcany", "Hradec Dol", "Kralovice",
		"Podhradi", "Lipnice", "Zlatovice", "Jizerov", "Berounka", "Tisnov", "Vysocina", "Kamenice",
		"Moravice",
	},
	"dk": {
		"Nordhavn", "Sønderby", "Vestkær", "Ålestrup", "Havnsø", "Bjerrebo", "Skovlund", "Kærgaard",
		"Lundtofte", "Strandby", "Fjordsholm", "Marsbro", "Egeskov", "Rønnebæk", "Højager",
		"Tvedeby",
	},
	"ee": {
		"Pärnuvere", "Kuressaar", "Rakvera", "Narvamäe", "Tartumaa", "Haapsalu", "Võrumaa",
		"Viljandi", "Paldiski", "Sillamäe", "Otepää", "Valgamaa", "Rapla", "Kärdla", "Jõgeva",
		"Põlva",
	},
	"fo": {
		"Nólsoyri", "Tórshavnar", "Klaksvíkar", "Vágsbotn", "Hvalvík", "Kollafjørð", "Sandavágur",
		"Eiðisvík", "Fuglafjarðar", "Trøllanes", "Tjørnuvík", "Viðareiði", "Hósvík", "Syðradalur",
		"Gjógv", "Mykines",
	},
	"fi": {
		"Järvikoski", "Koivula", "Kuusamo", "Metsälä", "Rantaniemi", "Pohjola", "Mäntylä",
		"Lapinkylä", "Hirvijärvi", "Kalliola", "Salonmäki", "Joutsenlahti", "Aurinkola", "Savolinna",
		"Tuulikylä", "Harjula",
	},
	"ge": {
		"Kolkhidi", "Mtskhetis", "Batumeli", "Kutaisoba", "Telavuri", "Gorielo", "Zugdidoba",
		"Akhaltsikhe", "Rustavo", "Svanetia", "Kakhetini", "Poti Bay", "Sighnagi", "Borjomi",
		"Tsalka", "Gudauri",
	},
	"gi": {
		"Rockside", "Europa Point", "Catalan Bay", "Westside", "Southport", "Eastern Bay",
		"Gibel Alto", "Main Quay", "Casemates", "Upper Town", "Bayside", "Alameda", "Kingsway",
		"Sandpits", "Moorish Hill", "Harbour Reach",
	},
	"hu": {
		"Aranyhalom", "Dunafalva", "Tiszavár", "Balatonfok", "Mezőhegy", "Szilvásd", "Nagykert",
		"Hegyalja", "Kisbaráth", "Bükkszeg", "Pusztavár", "Rétköz", "Zalaszög", "Őrségi", "Vértesfő",
		"Gyöngyfa",
	},
	"is": {
		"Hafnarvík", "Akurholt", "Selfjall", "Borgarfell", "Vestmannaby", "Hveragerdi", "Ísafold",
		"Egilsvik", "Stykkisfjörd", "Keflaby", "Hólmavik", "Skagafell", "Seydisvik", "Akranes Bay",
		"Blönduvik", "Grindahaf",
	},
	"il": {
		"Carmelit", "Negevia", "Galilon", "Sharonit", "Emekim", "Hadera Mar", "Ashkelonit",
		"Kinneret", "Shfelam", "Yarkona", "Tavorim", "Beersheva Dor", "Modiron", "Raananit",
		"Herzlia Yam", "Netanyaron",
	},
	"kz": {
		"Altynsay", "Qaragandy", "Aqtobe Dala", "Shymkala", "Semeybek", "Pavlodar Su", "Atyrau Mor",
		"Kostanay Zhol", "Turkistan Ak", "Oral Zhaiyk", "Taraz Bulak", "Ekibas", "Qyzylorda",
		"Balqash Kol", "Petropavl Ak", "Zhezkazgan",
	},
	"lv": {
		"Daugavpils Zil", "Jūrmalnieki", "Cēsuciems", "Liepājnieki", "Ventspils Jūra", "Rēzeknieki",
		"Valmieras", "Jelgavnieki", "Kuldīgnieki", "Salacgrīva", "Ogrenieki", "Tukumnieki",
		"Smiltene", "Saldus Lauki", "Bauskas", "Madona Kalni",
	},
	"li": {
		"Rheinau", "Triesenberg", "Schellenfeld", "Balzenau", "Gamprinfeld", "Eschenhof",
		"Mauerberg", "Planken", "Ruggell Au", "Nendelfeld", "Schaanwald", "Gaflei", "Malbunberg",
		"Steg Alpe", "Bendern Ried", "Vaduzer Au",
	},
	"lt": {
		"Nemunas", "Kauniškė", "Klaipėdiečiai", "Šiauliai Žemė", "Panevėžys Lanka", "Alytaus",
		"Marijampolė", "Utenos", "Mažeikiai", "Telšiai", "Kėdainių", "Jonavos", "Palangos",
		"Druskininkų", "Trakų Ežeras", "Plungė",
	},
	"lu": {
		"Moselfeld", "Ardennerhof", "Bourgvillage", "Esch Rouge", "Differbach", "Wiltzenau",
		"Clervaux Sud", "Mersch Val", "Redange Plain", "Grevenberg", "Echternach Pre",
		"Vianden Mont", "Diekirchois", "Steinfort Ridge", "Remich Cote", "Capellen Haut",
	},
	"mt": {
		"Marsaxlokk Bay", "Ghajnsielem", "Birgu Heights", "Mellieha Ridge", "Zebbug Vale",
		"Mosta Valley", "Rabat Hill", "Qormi Fields", "Gharb Cliff", "Mdina Walls", "Xaghra",
		"Siggiewi Pass", "Sliema Point", "Zurrieq", "Hamrun Park", "Birkirkara",
	},
	"md": {
		"Codrului", "Nistreni", "Prutenii", "Orheiul Nou", "Soroca Veche", "Cahulean", "Bălțeni",
		"Ungheni Mari", "Hâncești Deal", "Căușeni", "Edineț Nord", "Cimișlia", "Strășeni",
		"Drochia Sat", "Rezina Piatră", "Leova",
	},
	"nir": {
		"Lisbrae", "Ballymore", "Dunseverick", "Carrowglen", "Drumbeg", "Killyfane", "Knockbrae",
		"Moneyglass", "Tullybeg", "Glenshane", "Ardmoyle", "Cloughmills", "Loughbeg", "Slieveglen",
		"Bannfoot", "Rathmore",
	},
	"no": {
		"Fjellheim", "Nordvik", "Storvåg", "Lysefoss", "Bergtun", "Skogheim", "Vikøy", "Grønnes",
		"Haugland", "Elvestad", "Solvang", "Tindefjell", "Breidvik", "Nesodden", "Rognan",
		"Vestfjord",
	},
	"ro": {
		"Carpatina", "Valea Rosie", "Dunareni", "Muntenia Noua", "Banatul", "Oltenesti", "Ardealul",
		"Mureseni", "Prahovan", "Dobrogea", "Bucovina", "Moldovita", "Tarnavei", "Somesan",
		"Cluj Dealuri", "Brasovenii",
	},
	"sm": {
		"Monte Titano", "Borgo Alto", "Serravalle Alta", "Fiorentino", "Acquaviva", "Domagnano",
		"Faetano", "Chiesanuova", "Montegiardino", "Dogana Nuova", "Cailungo", "Ventoso",
		"Torraccia", "Falciano", "Montecerreto", "Rovereta",
	},
	"sk": {
		"Tatrany", "Podhorie", "Zvolenovo", "Nitrovce", "Vahovce", "Orava Dol", "Liptovec",
		"Spis Vrch", "Kosicovce", "Zilinany", "Trnavsko", "Presovec", "Banovce", "Hronsky Dol",
		"Turiec", "Gemerany",
	},
	"ch": {
		"Seebach", "Alpenau", "Rheinfall", "Bergmatt", "Tannenfeld", "Glarnerhof", "Aarburgau",
		"Sonnenrain", "Zugerberg", "Engelmatt", "Lindenberg", "Waldstatt", "Pilatusblick",
		"Bergdorf Nord", "Thunerau", "Matterhof",
	},
	"ua": {
		"Dniprovsk", "Zaporizhka", "Karpatska", "Polissya", "Slobozhansk", "Bukovynets", "Podolsk",
		"Halych Grad", "Chornomorsk", "Volynsk", "Cherkaska", "Sumshchyna", "Poltavka", "Kharkivets",
		"Vinnychyna", "Lvivets",
	},
	// CONCACAF
	"ai": {
		"Shoal Bay", "Sandy Ground", "Road Bay", "Blowing Point", "Meads Bay", "Island Harbour",
		"Crocus Hill", "Rendezvous", "Cap Juluca", "Stoney Ground", "Little Bay", "Forest Gate",
		"Long Path", "Mount Fortune", "Cove Reach", "Katouche",
	},
	"ag": {
		"Falmouth Reach", "Parham Hill", "Willikies", "Bolans Bay", "Cedar Grove", "Barbuda Point",
		"Codrington Bay", "Jolly Hill", "Bethesda", "Fig Tree Rise", "Hawksbill", "Dickenson Cove",
		"Freemans Bay", "Lignum Vitae", "Potworks", "Mamora Bay",
	},
	"aw": {
		"Oranjebaai", "Paradera", "Savaneta", "Santa Cruz", "Noord Beach", "Tanki Leendert",
		"Pos Chiquito", "Sero Colorado", "Bubali", "Piedra Plat", "Balashi", "Madiki", "Hooiberg",
		"Eagle Beach", "Malmok", "Bushiri",
	},
	"bs": {
		"Cable Bay", "Gambier", "Adelaide", "Fox Hill", "Carmichael", "Lyford Cay", "Harbour Rock",
		"Staniel Cove", "Governors Reach", "Rock Sound", "Kemps Bay", "Sweetings", "Nicolls Town",
		"Hope Town Bay", "Spanish Wells", "Arthurs Town",
	},
	"bb": {
		"Holetown", "Speightsbay", "Bathsheba Rise", "Oistins Cove", "Six Cross", "Worthing Hill",
		"Crane Ridge", "Hastings Point", "Welchman Hall", "Silver Sands", "Greenland",
		"Sam Lord Bay", "Bridgeview", "Cave Hill", "Pie Corner", "Sunset Crest",
	},
	"bz": {
		"Placencia Bay", "Hopkins Rise", "Corozal Way", "Orange Ridge", "Dangriga Cove",
		"Punta Reef", "Benque Hill", "Caye Point", "San Ignacio Rise", "Gales Point", "Sarteneja",
		"Maya Vale", "Monkey River", "Burrell Boom", "Crooked Tree", "Hattieville",
	},
	"bm": {
		"Somerset Reach", "Hamilton Sound", "Flatts Inlet", "Dockyard", "Warwick Bay", "Paget Point",
		"Smiths Hill", "Devonshire Rise", "Pembroke Cove", "Southampton Ridge", "Tuckers Point",
		"Spanish Rock", "Harrington Sound", "Crawl Bay", "Elbow Beach", "St Davids Head",
	},
	"vg": {
		"Road Harbour", "Cane Garden", "Sage Mountain", "Anegada Reef", "East End", "Brewers Bay",
		"Carrot Rock", "Fat Hogs Bay", "Long Bay", "Little Apple", "Spanish Town Bay", "Jost Cove",
		"Virgin Gorda Rise", "Norman Point", "Soper Hill", "Great Camanoe",
	},
	"ky": {
		"Bodden Reach", "Seven Mile", "West Bay Cove", "Rum Point", "Savannah Rise", "Spotts Bay",
		"Little Cayman Point", "Northward", "Frank Sound", "Cayman Brac Bluff", "Prospect",
		"Gun Bay", "Bodden Town Rise", "Red Bay", "Newlands", "Colliers",
	},
	"cr": {
		"Turrialba Alta", "San Ramonal", "Guanacaste Sur", "Puntarenal", "Limonera", "Heredial",
		"Cartago Viejo", "Alajuelita Norte", "Monteverde Alto", "Sarapiquí", "Quepos Mar", "Orotina",
		"Liberia Nueva", "Nicoya Sol", "Cañas Verde", "Pérez Zeledón",
	},
	"cu": {
		"Pinar Verde", "Matanzas Sur", "Camagüey Real", "Holguinera", "Cienfuegal", "Trinidad Alto",
		"Santiaguera", "Baracoa Mar", "Sancti Pinar", "Bayamera", "Guantanamera", "Cárdenas Sol",
		"Morón Verde", "Sagua Norte", "Remedios Sur", "Manzanilla",
	},
	"cw": {
		"Otrobanda", "Santa Rosa", "Scharloo", "Bandabou", "Jan Thiel", "Willemstad Zuid",
		"Marie Pompoen", "Salinja", "Zeelandia", "Rooi Catootje", "Piscadera", "Koraal Specht",
		"Souax", "Tera Cora", "Banda Abou", "Fuik",
	},
	"dm": {
		"Roseau Vale", "Portsmouth Bay", "Soufriere Rise", "Canefield", "Mahaut", "Salisbury Hill",
		"Marigot Cove", "Pointe Michel", "Grand Bay", "Castle Bruce", "Wesley Rise", "La Plaine",
		"Delices", "Layou", "Bellevue Chopin", "Trafalgar",
	},
	"do": {
		"La Romana Sur", "Puerto Verde", "San Pedro Mar", "Santiago Alto", "Higüey Real",
		"Bonao Norte", "Baní Costa", "Samaná Bay", "Monte Cristi", "Barahona Sol",
		"San Cristobalito", "Moca Verde", "Jarabacoa", "Azua Sur", "La Vega Nueva", "Neiba",
	},
	"sv": {
		"Cuscatlán Alto", "Santa Anita", "San Vicentino", "Usulután Mar", "Sonsonate Sol",
		"Chalatenango", "Ahuachapán", "Izalco Verde", "Cojutepeque", "La Unión Sur", "Zacatecoluca",
		"Metapán Alto", "Apopa Norte", "Ilopango Bay", "Nejapa", "Morazán Alto",
	},
	"gd": {
		"Saint Georges Bay", "Grand Anse", "Gouyave", "Sauteurs Hill", "Grenville Vale",
		"Victoria Point", "Hillsborough", "Carriacou Cove", "Petite Martinique", "Belmont",
		"Telescope", "Perdmontemps", "La Sagesse", "Woburn", "Calivigny", "Mount Moritz",
	},
	"gt": {
		"Antigua Real", "Quetzal Alto", "Escuintla Sur", "Cobán Verde", "Petén Norte", "Retalhuleu",
		"Chimaltenango", "Zacapa Seco", "Huehuetenango", "Mixco Alto", "Villa Nueva", "Amatitlán",
		"Jalapa Verde", "Sololá Lago", "Chiquimula", "Totonicapán",
	},
	"gy": {
		"Demerara Reach", "Berbice Point", "Essequibo", "Linden Hill", "Bartica Bay", "Kaieteur",
		"Lethem Rise", "Mahaica", "Anna Regina", "Skeldon", "New Amsterdam Rise", "Parika",
		"Charity", "Mabaruma", "Rosignol", "Vreed en Hoop",
	},
	"ht": {
		"Jacmelle", "Cap Soleil", "Gonaïves Mar", "Les Cayelle", "Petit Goâve", "Jérémieu",
		"Léogâne Sud", "Saint-Marc Haut", "Hinche Vert", "Fort Liberté", "Port-de-Paix", "Miragoâne",
		"Belladère", "Ouanaminthe", "Cabaret Nord", "Kenscoff",
	},
	"hn": {
		"Comayagua Alto", "Choluteca Sol", "Danlí Verde", "Tela Mar", "La Ceiba Sur", "Olancho",
		"Copán Real", "Roatán Costa", "Siguatepeque", "Yoro Alto", "Juticalpa", "Catacamas",
		"Santa Rosita", "El Progreso", "Intibucá", "Lempira Sur",
	},
	"jm": {
		"Portmore Hill", "Ocho Rios Bay", "Mandeville Rise", "Negril Point", "Spanish Vale",
		"Montego Reach", "Port Antonio", "Black River Bend", "Savanna Cove", "Falmouth Rise",
		"May Pen Green", "Kingston Hills", "Treasure Vale", "Whitehouse", "Linstead Bend",
		"Blue Mountain",
	},
	"ms": {
		"Plymouth Ridge", "Salem Hill", "Little Bay", "Brades", "Woodlands", "Cudjoe Head",
		"Old Towne", "Lees Rise", "St Peters", "Davy Hill", "Baker Hill", "Gerald Bay", "Look Out",
		"Isles Bay", "Cork Hill", "Richmond Hill",
	},
	"ni": {
		"Granada Real", "León Viejo", "Masaya Alto", "Estelí Norte", "Chinandega", "Matagalpa",
		"Rivas Sol", "Jinotega", "Juigalpa", "Ocotal", "Bluefields Sur", "Somoto Alto", "Diriamba",
		"Boaco Verde", "Nandaime", "Tipitapa",
	},
	"pa": {
		"Colón Mar", "Chitré Sol", "David Alto", "Santiago Verde", "Penonomé", "Las Tablas",
		"Aguadulce", "Bocas Norte", "Darién Real", "Veraguas", "Chepo Alto", "Boquete Sur",
		"La Chorrera", "Arraiján", "Portobelo", "Volcán Alto",
	},
	"pr": {
		"Ponce Viejo", "Arecibo Mar", "Mayagüez Sol", "Fajardo Alto", "Humacao Sur", "Cayey Verde",
		"Caguas Norte", "Utuado", "Aguadilla Bay", "Vieques Real", "Guayama", "Yauco Alto",
		"Manatí Costa", "Isabela Sol", "Naguabo", "Coamo",
	},
	"kn": {
		"Basseterre Bay", "Charlestown Rise", "Sandy Point", "Cayon Hill", "Frigate Bay",
		"Newton Ground", "Old Road", "Dieppe Bay", "Tabernacle", "Monkey Hill", "Brimstone",
		"Cotton Ground", "Gingerland", "Fig Tree", "Saddlers", "Molineux",
	},
	"lc": {
		"Castries Reach", "Gros Islet Bay", "Vieux Fort Rise", "Soufrière Hill", "Micoud",
		"Dennery Cove", "Laborie", "Choiseul", "Anse La Raye", "Marigot Point", "Babonneau",
		"Rodney Bay", "Canaries", "Pigeon Point", "Cap Estate", "Piton Vale",
	},
	"vc": {
		"Kingstown Rise", "Bequia Bay", "Mustique Point", "Georgetown", "Barrouallie", "Layou Hill",
		"Chateaubelair", "Canouan", "Calliaqua", "Arnos Vale", "Union Island", "Questelles",
		"Biabou", "Sandy Bay", "Mayreau", "Spring Village",
	},
	"sr": {
		"Paramaribo Noord", "Nickerie", "Wanica", "Brokopondo", "Moengo", "Albina", "Lelydorp",
		"Coronie", "Saramacca", "Commewijne", "Marowijne", "Para Oost", "Sipaliwini", "Groningen",
		"Totness", "Domburg",
	},
	"tt": {
		"Arima Valley", "Chaguanas Hill", "Point Fortin", "Scarborough Bay", "Couva Rise",
		"Sangre Grande", "Siparia", "Tunapuna", "Rio Claro", "Princes Cove", "Moruga",
		"Penal Heights", "Mayaro Bay", "Toco Point", "Debe", "Blanchisseuse",
	},
	"tc": {
		"Cockburn Town", "Providenciales Bay", "Blue Hills", "Five Cays", "Grace Bay", "Kew Town",
		"Bottle Creek", "Whitby", "North Caicos", "Salt Cay", "Sapodilla", "Long Bay Hills",
		"Leeward", "Chalk Sound", "Wheeland", "South Creek",
	},
	"vi": {
		"Charlotte Reach", "Frederiksted", "Christiansted", "Cruz Bay", "Coral Bay", "Estate Thomas",
		"Red Hook", "Magens", "Water Island", "Savan Hill", "Smith Bay", "Anna's Retreat",
		"Lindbergh Bay", "Sion Farm", "Bovoni", "Cinnamon Cove",
	},
	// AFC
	"af": {
		"Kabulan", "Heratak", "Mazaran", "Kandahar Ghar", "Jalalak", "Bamiyan Kot", "Ghazni Dara",
		"Panjshir Kol", "Kunduz Zor", "Nangarhar", "Helmandi", "Farah Kot", "Badakhshan",
		"Lashkar Dara", "Khost Ghar", "Parwan Kot",
	},
	"au": {
		"Kingsbrook", "Redgum", "Wattleby", "Bluewater", "Coolabah", "Thornbury Plains", "Saltbush",
		"Ironbark", "Mallee Ridge", "Banksia Heath", "Stringybark", "Outback Rise", "Wombat Creek",
		"Gumtree Flat", "Billabong", "Kookaburra",
	},
	"bh": {
		"Manaman", "Muharraqi", "Riffa Sands", "Sitra Bay", "Hamad Town", "Isa Heights", "Budaiya",
		"Jidhafs", "Sanabis", "Zallaq", "Galali", "Hidd", "Tubli", "Arad", "Jasra", "Dumistan",
	},
	"bd": {
		"Dhanmondi", "Sylhet Hill", "Rajshahir", "Padma Char", "Comilla Bari", "Meghna", "Khulnapur",
		"Barisal Ghat", "Rangpurer", "Jamuna Char", "Chattogram Bay", "Mymensing", "Bogra Gram",
		"Narayan Ghat", "Cox Sands", "Tangail Mor",
	},
	"bt": {
		"Thimphuri", "Paro Dzong", "Punakha Valley", "Wangdue", "Haa Valley", "Trongsa", "Bumthang",
		"Mongar Ridge", "Samtse", "Gelephu", "Phuntsholing", "Lhuntse", "Zhemgang", "Tashigang",
		"Pemagatshel", "Dagana",
	},
	"bn": {
		"Bandar Baru", "Kuala Belait", "Tutong Bay", "Muara Point", "Seria Hills", "Jerudong",
		"Gadong", "Temburong", "Kampong Ayer", "Lumut Bay", "Panaga", "Sengkurong", "Lamunin",
		"Kiulap", "Mentiri", "Rimba",
	},
	"kh": {
		"Angkor Vale", "Siem Rise", "Battambang Rice", "Kampot Pepper", "Mekong Reach", "Tonle Bay",
		"Kratie Falls", "Sihanouk Bay", "Koh Kong", "Preah Hill", "Kandal Plain", "Svay Reach",
		"Takeo Lake", "Pursat Hills", "Ratanak", "Mondul",
	},
	"tw": {
		"Taoyuan Hill", "Hualien Bay", "Tainan Port", "Keelung Harbour", "Yilan Plain",
		"Chiayi Ridge", "Kaohsiung Sun", "Pingtung", "Nantou Peak", "Miaoli", "Taitung Coast",
		"Hsinchu Wind", "Changhua", "Yunlin", "Penghu", "Kinmen",
	},
	"gu": {
		"Hagatna Bay", "Tamuning", "Dededo Hill", "Yigo", "Barrigada", "Mangilao", "Piti",
		"Santa Rita", "Agat Bay", "Talofofo", "Umatac", "Inarajan", "Merizo", "Asan", "Chalan Pago",
		"Sinajana",
	},
	"hk": {
		"Victoria Peak", "Kowloon Bay", "Aberdeen Harbour", "Lantau", "Tai Po Hill", "Sha Tin",
		"Tuen Mun", "Yuen Long", "Stanley Bay", "Cheung Chau", "Mong Kok Rise", "Sai Kung",
		"Repulse Bay", "Lamma", "Fanling", "Kwun Tong",
	},
	"iq": {
		"Basrawi", "Mosulan", "Karbalai", "Najafi Sands", "Kirkuk Oil", "Erbil Citadel",
		"Tigris Bend", "Euphrates Gate", "Ramadi Plain", "Nasiriyah", "Sulaymani", "Dohuk Hills",
		"Samarra", "Hilla Palm", "Amara", "Kut Bridge",
	},
	"jo": {
		"Ammanite", "Petra Rose", "Aqaba Bay", "Irbid Hills", "Zarqa Valley", "Madaba",
		"Jerash Stones", "Karak Castle", "Ajloun", "Salt Heights", "Wadi Rum", "Mafraq", "Tafilah",
		"Ma'an Dunes", "Dead Sea Shore", "Ruwaished",
	},
	"kw": {
		"Salmiyan", "Hawalli Bay", "Jahra Gate", "Farwaniya", "Ahmadi Oil", "Mubarak Shore",
		"Fahaheel", "Sabah Harbour", "Khaitan", "Abdali", "Sulaibikhat", "Mangaf", "Fintas",
		"Jabriya", "Rumaithiya", "Kaifan",
	},
	"kg": {
		"Issyk Bulak", "Naryn Tash", "Osh Bazar", "Talas Sai", "Karakol Zhol", "Jalalabad Suu",
		"Batken Tau", "Tokmok", "Kant Ordo", "Cholpon", "Kochkor", "Toktogul", "Uzgen", "Kara Balta",
		"Ala Too", "Song Kol",
	},
	"la": {
		"Vientiane Bend", "Luang Hills", "Pakse Falls", "Savannakhet", "Xieng Rise", "Mekong Isle",
		"Phonsavan", "Thakhek", "Vang Vieng", "Sam Neua", "Attapeu", "Muang Khoun", "Salavan",
		"Phongsaly", "Oudomxay", "Champasak",
	},
	"lb": {
		"Beirutan", "Tripoli Port", "Sidon Sea", "Tyrian Hills", "Baalbek Stones", "Cedars Ridge",
		"Zahle Valley", "Byblos Bay", "Jounieh", "Batroun", "Nabatieh", "Aley Hills", "Jbeil Shore",
		"Chouf", "Bekaa", "Anjar",
	},
	"mo": {
		"Taipa Isle", "Coloane Bay", "Cotai Strip", "Barra Hill", "Ruinas", "Praia Grande", "Senado",
		"Lótus", "Patane", "Mong Ha", "Areia Preta", "Ka Ho", "Hac Sa", "Lapa", "Macau Peninsula",
		"Guia Hill",
	},
	"my": {
		"Kuala Selatan", "Penang Hill", "Johor Strait", "Melaka River", "Perak Valley", "Sabah Peak",
		"Sarawak Bay", "Kelantan Rice", "Terengganu", "Pahang Forest", "Langkawi", "Ipoh Karst",
		"Kuching Reach", "Putra Ridge", "Cameron Hills", "Seremban",
	},
	"mv": {
		"Male Atoll", "Addu Reef", "Hulhumale", "Fuvahmulah", "Thinadhoo", "Kulhudhuffushi",
		"Naifaru", "Hithadhoo", "Mahibadhoo", "Funadhoo", "Dhidhdhoo", "Eydhafushi", "Veymandoo",
		"Felidhoo", "Ukulhas", "Maafushi",
	},
	"mn": {
		"Ulaan Khot", "Khovd Gol", "Erdenet Uul", "Darkhan Tal", "Choibalsan", "Dornod Tal",
		"Altai Uul", "Gobi Khot", "Orkhon Gol", "Tuv Aimag", "Bayan Uul", "Selenge Gol",
		"Khangai Uul", "Sukhbaatar", "Uvs Nuur", "Zavkhan",
	},
	"mm": {
		"Yangon Bay", "Mandalay Hill", "Bagan Plain", "Irrawaddy Bend", "Shan Heights", "Mawlamyine",
		"Pathein Delta", "Sittwe Port", "Taunggyi", "Naypyi", "Bago Gate", "Myitkyina", "Magway",
		"Hpa-An", "Dawei", "Pyay",
	},
	"np": {
		"Kathmandu Pass", "Pokhara Lake", "Lumbini Garden", "Biratnagar", "Bhaktapur Hill",
		"Janakpur", "Dharan Ridge", "Hetauda", "Butwal", "Nepalgunj", "Dhangadhi", "Itahari",
		"Birgunj", "Namche", "Patan Square", "Chitwan Jungle",
	},
	"kp": {
		"Chongjin", "Hamhung Bay", "Wonsan Shore", "Sinuiju", "Kaesong Hills", "Nampo Port", "Haeju",
		"Hyesan", "Kanggye", "Sariwon", "Songnim", "Tanchon", "Pochon", "Rason", "Anju", "Kimchaek",
	},
	"om": {
		"Muscati", "Salalah Sands", "Nizwa Fort", "Sohar Port", "Sur Dhow", "Ibri Oasis",
		"Barka Coast", "Rustaq", "Khasab Cliffs", "Bahla Wall", "Seeb Shore", "Ibra Valley", "Duqm",
		"Samail", "Masirah", "Sinaw",
	},
	"pk": {
		"Karachi Bay", "Lahori Bagh", "Multan Gate", "Peshawar Pass", "Quetta Hill", "Faisal Mandi",
		"Sialkot Road", "Gujrat Kot", "Hyderabad Sind", "Bahawal Pur", "Sukkur Bridge", "Abbottabad",
		"Murree Ridge", "Gwadar Port", "Swat Valley", "Hunza",
	},
	"ps": {
		"Nablusi", "Hebron Hill", "Ramallah Ridge", "Jericho Palm", "Gaza Shore", "Bethlehem Star",
		"Jenin Valley", "Tulkarm", "Qalqilya", "Khan Younis", "Rafah Gate", "Salfit", "Tubas",
		"Beit Jala", "Deir Balah", "Olive Grove",
	},
	"sg": {
		"Tampines Bay", "Jurong Lake", "Bishan Park", "Woodlands", "Clementi Ridge", "Punggol",
		"Sengkang", "Bedok Reservoir", "Queenstown", "Toa Payoh", "Pasir Ris", "Yishun",
		"Marina Reach", "Sentosa Isle", "Changi Point", "Hougang",
	},
	"lk": {
		"Colombo Fort", "Kandy Lake", "Galle Fort", "Jaffna Peninsula", "Negombo Beach",
		"Matara Coast", "Trinco Bay", "Anuradha", "Batticaloa", "Kurunegala", "Ratnapura",
		"Nuwara Hills", "Badulla", "Polonnaru", "Hambantota", "Dambulla",
	},
	"sy": {
		"Damascene", "Aleppo Citadel", "Homs Plain", "Latakia Coast", "Hama Wheels", "Tartus Port",
		"Palmyra Sands", "Daraa Hills", "Idlib Grove", "Raqqa Bend", "Deir Valley", "Suwayda",
		"Qamishli", "Hasakah", "Quneitra", "Bosra",
	},
	"tj": {
		"Dushanbe Bog", "Khujand Su", "Kulob Tepa", "Pamir Qala", "Panjakent", "Istaravshan",
		"Tursunzoda", "Vahdat", "Hisor Kot", "Rasht Dara", "Fann Tau", "Murghab", "Khorog", "Norak",
		"Danghara", "Gharm",
	},
	"tl": {
		"Dili Bay", "Baucau Hill", "Lospalos", "Maliana", "Suai Coast", "Same Ridge", "Viqueque",
		"Aileu Hills", "Liquica", "Ermera Coffee", "Manatuto", "Oecusse", "Atauro Isle", "Ainaro",
		"Balibo", "Maubisse",
	},
	"tm": {
		"Ashgabat Dag", "Mary Gala", "Turkmenabat", "Dashoguz", "Balkan Dag", "Kopetdag",
		"Karakum Kum", "Serdar", "Tejen", "Bayramaly", "Atamyrat", "Gokdepe", "Kaka Tepe", "Annau",
		"Gyzylarbat", "Gumdag",
	},
	"ye": {
		"Sanaani", "Aden Harbour", "Taiz Heights", "Hodeida Coast", "Mukalla Bay", "Ibb Green",
		"Marib Dam", "Zinjibar", "Seiyun Palm", "Dhamar", "Saada Hills", "Sayun", "Socotra",
		"Hajjah", "Amran", "Bayda",
	},
	// CAF
	"ao": {
		"Luanda Mar", "Benguela Sul", "Huambo Alto", "Lobito Baía", "Cabinda Norte", "Malanje Verde",
		"Namibe Sol", "Uíge Real", "Lubango Serra", "Kuito Centro", "Saurimo Leste", "Ondjiva",
		"Menongue", "Soyo Costa", "Dundo", "Cuanza",
	},
	"bj": {
		"Cotonou Mer", "Ouidah Plage", "Abomeyen", "Parakou Nord", "Djougou", "Natitingou",
		"Lokossa", "Kandi", "Bohicon", "Savalou", "Dassa", "Allada", "Pobè", "Malanville",
		"Tanguiéta", "Come",
	},
	"bw": {
		"Gaborone Dam", "Francistown Hill", "Maun Delta", "Kasane Falls", "Serowe Kgotla",
		"Molepolole", "Palapye Rail", "Lobatse Hill", "Selebi Mine", "Mahalapye", "Kanye Ridge",
		"Tsabong", "Orapa", "Jwaneng", "Ghanzi Plain", "Thamaga",
	},
	"bf": {
		"Ouagadou", "Bobo Sud", "Koudougou", "Banfora Falls", "Ouahigouya", "Kaya Nord",
		"Fada Gourma", "Dédougou", "Tenkodogo", "Dori Sahel", "Gaoua", "Ziniaré", "Léo", "Manga",
		"Kombissiri", "Réo",
	},
	"bi": {
		"Bujumbura Lac", "Gitega Haut", "Ngozi Nord", "Muyinga", "Kayanza", "Rumonge", "Makamba",
		"Cibitoke", "Bururi", "Rutana", "Karuzi", "Kirundo", "Cankuzo", "Mwaro", "Bubanza", "Ruyigi",
	},
	"cv": {
		"Praia Alta", "Mindelo Baía", "Sal Rei", "Tarrafal", "Assomada", "Santa Maria",
		"Pedra Badejo", "Fogo Cume", "Brava Mar", "Sao Filipe", "Calheta", "Ribeira Grande",
		"Maio Sol", "Porto Novo", "Espargos", "Cidade Velha",
	},
	"cf": {
		"Bangui Rive", "Bimbo Sud", "Berberati", "Bambari", "Bouar", "Carnot", "Bria Nord", "Sibut",
		"Kaga", "Mbaïki", "Bossangoa", "Nola", "Paoua", "Alindao", "Obo", "Zemio",
	},
	"td": {
		"Ndjamena Sud", "Moundou", "Sarh Rive", "Abéché", "Kelo", "Koumra", "Pala", "Am Timan",
		"Bongor", "Mongo", "Doba", "Ati", "Laï", "Massakory", "Bol Lac", "Fada",
	},
	"km": {
		"Moroni Mer", "Mutsamudu", "Fomboni", "Domoni", "Mitsamiouli", "Ouani", "Sima", "Foumbouni",
		"Mbeni", "Itsandra", "Mremani", "Chomoni", "Ntsaoueni", "Nioumakele", "Bambao", "Mohéli",
	},
	"cg": {
		"Brazza Rive", "Pointe Noire", "Dolisie", "Ouesso", "Owando", "Impfondo", "Madingou",
		"Sibiti", "Kinkala", "Djambala", "Gamboma", "Mossendjo", "Loandjili", "Makélékélé",
		"Talangaï", "Mfilou",
	},
	"cd": {
		"Kinshasa Fleuve", "Lubumbashi Cuivre", "Mbuji Mayi", "Kisangani", "Goma Lac", "Bukavu Rive",
		"Kolwezi", "Likasi", "Tshikapa", "Matadi Port", "Kananga", "Kikwit", "Uvira", "Butembo",
		"Mbandaka", "Boma",
	},
	"dj": {
		"Djibouti Port", "Ali Sabieh", "Tadjoura", "Obock", "Dikhil", "Arta", "Holhol", "Balbala",
		"Gabode", "Loyada", "Damerjog", "Yoboki", "Randa", "Goubet", "Sagallo", "Lac Assal",
	},
	"gq": {
		"Malabo Bahía", "Bata Mar", "Ebebiyín", "Mongomo", "Luba Sur", "Evinayong", "Mbini",
		"Aconibe", "Oyala", "Añisoc", "Cogo", "Nsok", "Riaba", "Baney", "Rebola", "Annobón",
	},
	"er": {
		"Asmara Alta", "Keren Valle", "Massawa Mar", "Mendefera", "Assab Sur", "Dekemhare",
		"Barentu", "Adi Keyh", "Teseney", "Nakfa", "Ghinda", "Afabet", "Senafe", "Tessenei",
		"Dahlak", "Agordat",
	},
	"sz": {
		"Mbabane Hills", "Manzini Plains", "Lobamba Royal", "Siteki Ridge", "Piggs Peak",
		"Nhlangano", "Big Bend", "Hlathikhulu", "Mhlume", "Simunye", "Malkerns Valley", "Ezulwini",
		"Matsapha", "Lavumisa", "Tshaneni", "Bhunya",
	},
	"et": {
		"Addis Alto", "Gondar Castle", "Lalibela Rock", "Hawassa Lake", "Adama Sun", "Bahir Dar",
		"Dire Dawa", "Jimma Coffee", "Mekelle Hill", "Harar Gate", "Axum Stelae", "Dessie Ridge",
		"Debre Zeit", "Arba Minch", "Nekemte", "Sodo",
	},
	"ga": {
		"Libreville Mer", "Port-Gentil", "Franceville", "Oyem Nord", "Lambaréné", "Moanda", "Mouila",
		"Tchibanga", "Koulamoutou", "Makokou", "Bitam", "Gamba", "Ntoum", "Mayumba", "Lastourville",
		"Booué",
	},
	"gm": {
		"Banjul Bay", "Serekunda", "Brikama Hill", "Bakau Cape", "Farafenni", "Lamin", "Basse Bend",
		"Gunjur", "Soma", "Kerewan", "Janjanbureh", "Sukuta", "Brufut", "Bwiam", "Kartong", "Essau",
	},
	"gn": {
		"Conakry Mer", "Kankan Rive", "Kindia Hill", "Labé Fouta", "Nzérékoré", "Mamou", "Boké Port",
		"Faranah", "Siguiri Or", "Kamsar", "Dabola", "Télimélé", "Macenta", "Guéckédou", "Fria",
		"Dubréka",
	},
	"gw": {
		"Bissau Velho", "Bafatá Rio", "Gabu Norte", "Cacheu", "Bolama", "Bubaque", "Farim", "Mansôa",
		"Quinhámel", "Catió", "Buba", "Canchungo", "Fulacunda", "Bijagós", "Bambadinca",
		"Cantchungo",
	},
	"ls": {
		"Maseru Hills", "Teyateyaneng", "Mafeteng Plain", "Hlotse", "Mohale Dam", "Quthing",
		"Butha Buthe", "Mokhotlong Peak", "Leribe Ridge", "Roma Valley", "Thaba Tseka", "Qacha Nek",
		"Morija", "Peka", "Semonkong", "Maputsoe",
	},
	"lr": {
		"Monrovia Bay", "Buchanan Port", "Gbarnga Hill", "Kakata Road", "Harper Cape", "Voinjama",
		"Zwedru", "Robertsport", "Sanniquellie", "Greenville Coast", "Bensonville", "Ganta",
		"Tubmanburg", "Yekepa", "Foya", "Paynesville",
	},
	"ly": {
		"Tripolis", "Benghazian", "Misrata Port", "Sabha Oasis", "Zawiya Shore", "Tobruk Bay",
		"Sirte Gulf", "Derna Valley", "Zliten", "Ghadames", "Bayda Hills", "Khoms Ruins", "Ajdabiya",
		"Kufra Palm", "Murzuq", "Sabratha",
	},
	"mg": {
		"Tana Haut", "Toamasina Mer", "Antsirabe Sud", "Mahajanga Baie", "Fianarantsoa",
		"Toliara Sud", "Antsiranana", "Ambatondrazaka", "Morondava", "Nosy Be", "Manakara",
		"Sambava", "Farafangana", "Ambositra", "Maroantsetra", "Tolagnaro",
	},
	"mw": {
		"Lilongwe Hill", "Blantyre Ridge", "Mzuzu Plateau", "Zomba Mount", "Kasungu", "Salima Lake",
		"Mangochi Bay", "Karonga North", "Nkhata Bay", "Dedza Ridge", "Mulanje Peak", "Liwonde",
		"Balaka", "Ntcheu", "Chitipa", "Mchinji",
	},
	"mr": {
		"Nouakchott Dune", "Nouadhibou Cap", "Rosso Fleuve", "Kaédi", "Zouérat", "Atar Oasis",
		"Kiffa", "Aleg", "Tidjikja", "Sélibaby", "Akjoujt", "Néma", "Boutilimit", "Ouadane",
		"Chinguetti", "Aïoun",
	},
	"mu": {
		"Port Louisien", "Curepipe Haut", "Vacoas Hill", "Quatre Bornes", "Rose Hill Sud",
		"Beau Bassin", "Mahébourg Bay", "Flic en Flac", "Grand Baie", "Triolet", "Goodlands",
		"Rodrigues", "Souillac Cape", "Moka Vale", "Pamplemousses", "Bel Ombre",
	},
	"mz": {
		"Maputo Baía", "Beira Porto", "Nampula Norte", "Quelimane Rio", "Tete Serra", "Pemba Costa",
		"Chimoio Alto", "Lichinga", "Xai-Xai Mar", "Inhambane", "Nacala Baía", "Gurué", "Mocuba",
		"Matola Sul", "Songo", "Angoche",
	},
	"na": {
		"Windhoek Hills", "Walvis Bay", "Swakopmund Dune", "Oshakati", "Rundu Rapids",
		"Katima Mulilo", "Keetmanshoop", "Otjiwarongo", "Lüderitz Bay", "Rehoboth", "Tsumeb Mine",
		"Gobabis Plain", "Mariental", "Okahandja", "Etosha Pan", "Henties Bay",
	},
	"ne": {
		"Niamey Fleuve", "Zinder Banc", "Maradi Sud", "Agadez Désert", "Tahoua Nord", "Dosso Rive",
		"Diffa Lac", "Tillabéri", "Arlit Mine", "Birni", "Gaya Pont", "Tessaoua", "Dogondoutchi",
		"Madaoua", "Magaria", "Aïr",
	},
	"rw": {
		"Kigali Hills", "Butare Sud", "Gisenyi Lac", "Ruhengeri", "Byumba", "Cyangugu", "Kibuye Lac",
		"Rwamagana", "Nyanza Royal", "Huye Plateau", "Musanze Volcans", "Rubavu", "Muhanga",
		"Nyagatare", "Kayonza", "Bugesera",
	},
	"st": {
		"São Tomé Baía", "Santo António", "Trindade", "Neves", "Santana", "Guadalupe",
		"Porto Alegre", "Ribeira Afonso", "Príncipe Cume", "Pantufo", "Madalena", "Bombom",
		"Roça Velha", "Água Grande", "Cantagalo", "Lembá",
	},
	"sc": {
		"Victoria Port", "Beau Vallon", "Anse Royale", "Praslin Bay", "La Digue", "Takamaka",
		"Anse Boileau", "Mont Fleuri", "Cascade", "Baie Lazare", "Glacis", "Bel Ombre", "Port Glaud",
		"Grand Anse", "Anse Etoile", "Au Cap",
	},
	"sl": {
		"Freetown Bay", "Bo Town", "Kenema Hill", "Makeni Road", "Koidu Mine", "Lunsar", "Port Loko",
		"Waterloo", "Kabala", "Magburaka", "Segbwema", "Moyamba", "Bonthe", "Lungi", "Kailahun",
		"Pujehun",
	},
	"so": {
		"Mogadishu Mar", "Hargeisa Alta", "Kismayo Puerto", "Baidoa", "Berbera Port", "Bosaso",
		"Garowe", "Galkayo", "Merca", "Jowhar", "Burao", "Beledweyne", "Afgooye", "Borama",
		"Las Anod", "Erigavo",
	},
	"ss": {
		"Juba Nile", "Wau Plain", "Malakal Bend", "Yei Valley", "Bor Marsh", "Torit Hills", "Bentiu",
		"Rumbek", "Aweil Sun", "Yambio", "Kapoeta", "Nimule", "Maridi", "Renk", "Kuajok", "Mundri",
	},
	"sd": {
		"Khartoumi", "Omdurman Nile", "Port Sudan", "Kassala Hill", "Nyala Dune", "El Obeid",
		"Gedaref Plain", "Wad Medani", "Atbara", "Dongola Bend", "Kosti", "Sennar", "Kadugli",
		"Geneina", "Dilling", "Berber",
	},
	"tg": {
		"Lomé Mer", "Kara Nord", "Sokodé", "Kpalimé Mont", "Atakpamé", "Dapaong", "Tsévié", "Aného",
		"Bassar", "Tchamba", "Badou", "Notsé", "Vogan", "Kanté", "Mango", "Cinkassé",
	},
	"ug": {
		"Kampala Hill", "Jinja Nile", "Entebbe Bay", "Gulu North", "Mbarara Ridge", "Mbale Mount",
		"Fort Portal", "Masaka Lake", "Lira Plain", "Arua Gate", "Kabale Hills", "Soroti Rock",
		"Hoima Oil", "Tororo Rock", "Kasese", "Busia",
	},
	"zm": {
		"Lusaka Plain", "Ndola Copper", "Kitwe Mine", "Livingstone Falls", "Kabwe Rail", "Chingola",
		"Mufulira", "Kasama Hill", "Chipata East", "Solwezi", "Mongu Plain", "Mazabuka",
		"Choma Ridge", "Kafue Flats", "Mansa Lake", "Siavonga",
	},
	"zw": {
		"Harare Kopje", "Bulawayo Stone", "Mutare Hills", "Gweru Rail", "Masvingo Ruins",
		"Kwekwe Gold", "Chitungwiza", "Victoria Falls", "Hwange Coal", "Bindura", "Kariba Lake",
		"Marondera", "Chinhoyi Caves", "Zvishavane", "Beitbridge", "Gokwe",
	},
	// OFC (folded into the Asian region, see RegionForCountry)
	"as": {
		"Pago Bay", "Tafuna Hill", "Fagatogo", "Leone Valley", "Nuuuli", "Utulei", "Aua Point",
		"Vatia Cove", "Aoloau", "Fatumafuti", "Malaeloa", "Faleniu", "Ofu Isle", "Taʻu Peak",
		"Alega", "Amouli",
	},
	"ck": {
		"Avarua Bay", "Muri Lagoon", "Arorangi", "Titikaveka", "Aitutaki Reef", "Atiu Cave",
		"Mangaia Cliff", "Ngatangiia", "Matavera", "Tupapa", "Takuvaine", "Avatiu Harbour",
		"Mauke Isle", "Pukapuka", "Rakahanga", "Manihiki",
	},
	"fj": {
		"Suva Bay", "Nadi Reef", "Lautoka Mill", "Labasa Plains", "Levuka Harbour", "Sigatoka Dune",
		"Savusavu Cove", "Nausori", "Rakiraki", "Taveuni Rise", "Korovou", "Navua Delta",
		"Ba Valley", "Tavua Mine", "Vatukoula", "Pacific Harbour",
	},
	"nc": {
		"Nouméa Baie", "Dumbéa", "Païta Plaine", "Koné Nord", "Bourail Plage", "Lifou Récif",
		"Maré Rocher", "Thio Mine", "Poindimié", "Houaïlou", "Canala Baie", "Yaté Lac", "Pouembout",
		"Voh Ouest", "Kaala", "Ouvéa Lagon",
	},
	"nz": {
		"Kauriwood", "Tasman Bay", "Fernhill", "Rotoruan", "Waikato Flat", "Taranaki Ridge",
		"Canterbury Plain", "Otago Harbour", "Marlborough", "Hawkes Reach", "Whangarei Vale",
		"Napier Cove", "Nelson Sound", "Queenstown Rise", "Wairarapa", "Southland",
	},
	"pg": {
		"Port Hills", "Lae Harbour", "Madang Reef", "Mount Valley", "Goroka Ridge", "Rabaul Bay",
		"Kokopo Cove", "Wewak Coast", "Kimbe Bay", "Popondetta", "Daru Isle", "Alotau Bay",
		"Mendi Highlands", "Kundiawa", "Wabag", "Vanimo",
	},
	"ws": {
		"Apia Harbour", "Faleata", "Vaitele", "Salelologa", "Lefaga Beach", "Falealili", "Safotu",
		"Asau Bay", "Vailima", "Leulumoega", "Satupaitea", "Aleipata", "Manono Isle", "Fagaloa",
		"Savaii Rise", "Lalomanu",
	},
	"sb": {
		"Honiara Ridge", "Gizo Reef", "Auki Bay", "Munda Point", "Kirakira", "Buala", "Lata Isle",
		"Tulagi Harbour", "Noro", "Taro Isle", "Malaita Hills", "Choiseul Bay", "Henderson",
		"Ghatere", "Santa Cruz", "Rennell",
	},
	"pf": {
		"Papeete Baie", "Faaa Plage", "Punaauia", "Pirae", "Moorea Lagon", "Bora Lagon", "Arue",
		"Mahina Point", "Teahupoo", "Taravao", "Paea", "Huahine Récif", "Raiatea", "Papara",
		"Mataiea", "Tiarei",
	},
	"to": {
		"Nukualofa", "Vavau Harbour", "Haapai Reef", "Neiafu", "Pangai", "Mua Ruins", "Houma",
		"Kolonga", "Hihifo", "Fua'amotu", "Eua Cliffs", "Ngele'ia", "Niuafoou", "Lapaha", "Vaini",
		"Nukunuku",
	},
	"vu": {
		"Port Vila Bay", "Luganville", "Tanna Volcano", "Malekula", "Espiritu Reef", "Pentecost",
		"Erromango", "Ambrym Fire", "Santo Coast", "Mele Bay", "Epi Isle", "Futuna Isle", "Aneityum",
		"Lenakel", "Norsup", "Isangel",
	},
}

// Shared club-naming patterns for the FIFA-membership countries (see KnownCountries), grouped by
// language or football culture - a country with its own distinct style keeps its own slice.
var (
	afenTeamPatterns = []string{"%s FC", "%s United", "%s Stars", "%s City"}
	afrTeamPatterns  = []string{"%s FC", "AS %s", "%s AC", "Union %s", "Etoile %s"}
	araTeamPatterns  = []string{"Al %s", "%s SC", "Nadi %s", "%s FC"}
	balTeamPatterns  = []string{"FK %s", "FC %s", "%s United", "JK %s"}
	dutTeamPatterns  = []string{"%s FC", "SV %s", "%s United", "Deportivo %s"}
	engTeamPatterns  = []string{"%s FC", "%s United", "%s City", "%s Rovers", "%s Athletic"}
	espTeamPatterns  = []string{"%s FC", "Deportivo %s", "Atletico %s", "Club %s", "Real %s"}
	fraTeamPatterns  = []string{"%s FC", "AS %s", "Olympique %s", "Stade %s", "%s AC"}
	gerTeamPatterns  = []string{"FC %s", "SC %s", "FC %s United", "%s FC"}
	grTeamPatterns   = []string{"%s FC", "AO %s", "Omonia %s", "%s AE"}
	indTeamPatterns  = []string{"%s FC", "%s United", "%s SC", "%s Athletic"}
	itTeamPatterns   = []string{"%s FC", "AC %s", "AS %s", "US %s", "%s Calcio"}
	norTeamPatterns  = []string{"%s IF", "%s BK", "IFK %s", "%s FK"}
	pacTeamPatterns  = []string{"%s FC", "%s United", "%s Warriors", "%s Rangers"}
	porTeamPatterns  = []string{"%s FC", "Sporting %s", "Uniao %s", "Atletico %s", "%s Clube"}
	seaTeamPatterns  = []string{"%s FC", "%s United", "%s City", "%s SC"}
	slavTeamPatterns = []string{"FK %s", "%s FC", "Dinamo %s", "%s United"}
	turkTeamPatterns = []string{"%s FK", "Dinamo %s", "%s United", "Lokomotiv %s"}
)

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
	// UEFA
	"al":  slavTeamPatterns,
	"ad":  espTeamPatterns,
	"am":  slavTeamPatterns,
	"az":  turkTeamPatterns,
	"by":  slavTeamPatterns,
	"bg":  slavTeamPatterns,
	"cy":  grTeamPatterns,
	"cz":  slavTeamPatterns,
	"dk":  norTeamPatterns,
	"ee":  balTeamPatterns,
	"fo":  norTeamPatterns,
	"fi":  balTeamPatterns,
	"ge":  slavTeamPatterns,
	"gi":  engTeamPatterns,
	"hu":  slavTeamPatterns,
	"is":  norTeamPatterns,
	"il":  araTeamPatterns,
	"kz":  turkTeamPatterns,
	"lv":  balTeamPatterns,
	"li":  gerTeamPatterns,
	"lt":  balTeamPatterns,
	"lu":  fraTeamPatterns,
	"mt":  engTeamPatterns,
	"md":  slavTeamPatterns,
	"nir": engTeamPatterns,
	"no":  norTeamPatterns,
	"ro":  slavTeamPatterns,
	"sm":  itTeamPatterns,
	"sk":  slavTeamPatterns,
	"ch":  gerTeamPatterns,
	"ua":  slavTeamPatterns,
	// CONCACAF
	"ai": engTeamPatterns,
	"ag": engTeamPatterns,
	"aw": dutTeamPatterns,
	"bs": engTeamPatterns,
	"bb": engTeamPatterns,
	"bz": engTeamPatterns,
	"bm": engTeamPatterns,
	"vg": engTeamPatterns,
	"ky": engTeamPatterns,
	"cr": espTeamPatterns,
	"cu": espTeamPatterns,
	"cw": dutTeamPatterns,
	"dm": engTeamPatterns,
	"do": espTeamPatterns,
	"sv": espTeamPatterns,
	"gd": engTeamPatterns,
	"gt": espTeamPatterns,
	"gy": engTeamPatterns,
	"ht": fraTeamPatterns,
	"hn": espTeamPatterns,
	"jm": engTeamPatterns,
	"ms": engTeamPatterns,
	"ni": espTeamPatterns,
	"pa": espTeamPatterns,
	"pr": espTeamPatterns,
	"kn": engTeamPatterns,
	"lc": engTeamPatterns,
	"vc": engTeamPatterns,
	"sr": dutTeamPatterns,
	"tt": engTeamPatterns,
	"tc": engTeamPatterns,
	"vi": engTeamPatterns,
	// AFC
	"af": turkTeamPatterns,
	"au": engTeamPatterns,
	"bh": araTeamPatterns,
	"bd": indTeamPatterns,
	"bt": indTeamPatterns,
	"bn": seaTeamPatterns,
	"kh": seaTeamPatterns,
	"tw": seaTeamPatterns,
	"gu": pacTeamPatterns,
	"hk": engTeamPatterns,
	"iq": araTeamPatterns,
	"jo": araTeamPatterns,
	"kw": araTeamPatterns,
	"kg": turkTeamPatterns,
	"la": seaTeamPatterns,
	"lb": araTeamPatterns,
	"mo": porTeamPatterns,
	"my": seaTeamPatterns,
	"mv": indTeamPatterns,
	"mn": turkTeamPatterns,
	"mm": seaTeamPatterns,
	"np": indTeamPatterns,
	"kp": seaTeamPatterns,
	"om": araTeamPatterns,
	"pk": indTeamPatterns,
	"ps": araTeamPatterns,
	"sg": seaTeamPatterns,
	"lk": indTeamPatterns,
	"sy": araTeamPatterns,
	"tj": turkTeamPatterns,
	"tl": porTeamPatterns,
	"tm": turkTeamPatterns,
	"ye": araTeamPatterns,
	// CAF
	"ao": porTeamPatterns,
	"bj": fraTeamPatterns,
	"bw": afenTeamPatterns,
	"bf": afrTeamPatterns,
	"bi": afrTeamPatterns,
	"cv": porTeamPatterns,
	"cf": fraTeamPatterns,
	"td": fraTeamPatterns,
	"km": fraTeamPatterns,
	"cg": fraTeamPatterns,
	"cd": fraTeamPatterns,
	"dj": fraTeamPatterns,
	"gq": espTeamPatterns,
	"er": araTeamPatterns,
	"sz": afenTeamPatterns,
	"et": afenTeamPatterns,
	"ga": fraTeamPatterns,
	"gm": afenTeamPatterns,
	"gn": fraTeamPatterns,
	"gw": porTeamPatterns,
	"ls": afenTeamPatterns,
	"lr": afenTeamPatterns,
	"ly": araTeamPatterns,
	"mg": fraTeamPatterns,
	"mw": afenTeamPatterns,
	"mr": araTeamPatterns,
	"mu": fraTeamPatterns,
	"mz": porTeamPatterns,
	"na": afenTeamPatterns,
	"ne": fraTeamPatterns,
	"rw": fraTeamPatterns,
	"st": porTeamPatterns,
	"sc": fraTeamPatterns,
	"sl": afenTeamPatterns,
	"so": araTeamPatterns,
	"ss": afenTeamPatterns,
	"sd": araTeamPatterns,
	"tg": fraTeamPatterns,
	"ug": afenTeamPatterns,
	"zm": afenTeamPatterns,
	"zw": afenTeamPatterns,
	// OFC (folded into the Asian region, see RegionForCountry)
	"as": pacTeamPatterns,
	"ck": pacTeamPatterns,
	"fj": pacTeamPatterns,
	"nc": fraTeamPatterns,
	"nz": engTeamPatterns,
	"pg": pacTeamPatterns,
	"ws": pacTeamPatterns,
	"sb": pacTeamPatterns,
	"pf": pacTeamPatterns,
	"to": pacTeamPatterns,
	"vu": pacTeamPatterns,
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
