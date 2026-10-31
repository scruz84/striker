#!/usr/bin/env python3
"""Assemble utils/namepools/<code>/{first,last}.txt for every country except Spain.

How the lists were built (re-run only to regenerate them from scratch; the text files are the source
of truth and are edited by hand afterwards):
  * names already in the repository's migrations are kept (init68/init69 pools, init.sql);
  * hand-written seeds for cultures the dataset does not cover (name_pool_seeds.py);
  * the `names-dataset` PyPI package (MIT; first/last names per country, 105 countries) gives
    the bulk for covered countries. Its names are aggregated frequency lists, and the package's
    own README says they were extracted from the 2021 Facebook data leak - no personal records are
    stored here, only common names. Each name is kept for a country only if that country holds a
    meaningful share of the name's occurrences, and the lists of migrant-heavy countries (Gulf
    states) are replaced by Arab-world donors;
  * uncovered countries blend the lists of culturally close covered countries (DONORS below), each
    country taking a different hash-selected subset so neighbours do not end up identical;
  * Vietnam, Thailand and the Korean surname list are composed from syllables/clan seats, because
    the dataset has too few real entries to reach the minimum pool sizes.

Run:  pip install names-dataset pycountry && python3 utils/build_name_pool_sources.py
then: python3 utils/gen_name_pools.py
"""
import hashlib, json, re, sys, unicodedata
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent))
from name_pool_seeds import S as SEEDS

REPO = Path(__file__).resolve().parent.parent
OUT = REPO / "utils" / "namepools"


def load_dataset():
    from names_dataset import NameDataset

    nd = NameDataset()
    data = {}
    for cc in sorted(nd.get_country_codes(alpha_2=True)):
        data[cc] = {
            "first": nd.get_top_names(n=4000, use_first_names=True, country_alpha2=cc, gender="Male")[cc]["M"],
            "last": nd.get_top_names(n=6000, use_first_names=False, country_alpha2=cc)[cc],
        }
    return data


ND = load_dataset()

N_FIRST, N_LAST = 380, 650
ALLOWED = set(" '-.")
BLACKLIST = {w.lower() for w in """
mr mrs ms miss dr sir haji hajji hj sheikh sheik shaikh prince princess king queen lord lady ratu mohd md
official page fan fans club facebook real boss star love baby angel lover lovely sweet sexy cute cool
king queens crazy happy little big mama papa daddy brothers brother sister family team group
tonga fiji jamaica georgia china india japan korea flower syrian kurdish iraqi arab arabic muslim islam sport spb dream kt ch bk
""".split()}

# --- existing pools, from the migrations -------------------------------------------------------
def existing_pools():
    pools = {}
    sql_dir = REPO / "backend" / "resources" / "sql"
    pat = re.compile(r"INSERT INTO (names|lastnames)_(\w+)\b(.*?);", re.S | re.I)
    tup = re.compile(r"\(\s*'((?:[^']|'')*)'\s*,\s*[\d.]+\s*\)")
    for f in sorted(sql_dir.glob("*.sql")):
        if f.name.startswith("names_es") or f.name.startswith("lastnames_es"):
            continue
        for kind, code, body in pat.findall(f.read_text(encoding="utf-8")):
            names = [n.replace("''", "'") for n in tup.findall(body)]
            key = (code, "first" if kind == "names" else "last")
            pools.setdefault(key, [])
            for n in names:
                if n not in pools[key]:
                    pools[key].append(n)
    return pools

def is_latin(name):
    for ch in name:
        if ch in ALLOWED:
            continue
        if not ch.isalpha() or not unicodedata.name(ch, "").startswith("LATIN"):
            return False
    return True

def clean(name, lo=2, hi=24):
    name = " ".join(name.split())
    if name.isupper() and len(name) > 3:
        name = name.title()
    if not (lo <= len(name) <= hi) or not is_latin(name) or not name[0].isalpha() or not name[0].isupper():
        return None
    low = name.lower()
    if len(low) >= 5 and not re.search(r"[aeiouy\u00e0-\u00ff]", low):
        return None
    if low in BLACKLIST or any(w in BLACKLIST for w in low.split()):
        return None
    if re.search(r"(.)\1\1", low):  # aaa, junk
        return None
    return name

import pycountry
from names_dataset import NameDataset
_nd = NameDataset()
_share = {}
NATIVE_DIACRITICS = set("AL AT AZ BE BG CH CY CZ DE DK EE ES FI FR GB GE GR HR HU IE IS IT LT LU MD MT NL NO PL PT RS RU SE SI TR TM KZ AR BO BR CA CL CO CR EC GT HN MX PA PE PR SV US UY HT".split())
FEM = {
 "RU": ("ova", "eva", "ina", "yna", "aya", "ovna", "evna"), "KZ": ("ova", "eva", "ina", "aya"), "TM": ("ova", "eva", "aya"),
 "BG": ("ova", "eva", "ina"), "AZ": ("ova", "eva"), "CZ": ("ova", "\u00e1"), "PL": ("ska", "cka", "dzka", "zka"),
 "LT": ("iene", "aite", "yte", "ute", "\u0117", "\u00e9"), "IS": ("d\u00f3ttir", "dottir"), "EE": ("ova", "eva"),
 "MD": ("ova", "eva"), "GE": (), "SI": ("ova",), "HR": (), "RS": (),
}
def _fold(s):
    return "".join(c for c in unicodedata.normalize("NFD", s) if not unicodedata.combining(c))

ARAB = ["Saudi Arabia", "Egypt", "Iraq", "Jordan", "Lebanon", "Syrian Arab Republic", "Palestine, State of", "Yemen", "Morocco", "Algeria", "Tunisia", "Libya", "Sudan", "Kuwait", "Qatar", "United Arab Emirates", "Bahrain", "Oman"]
GULF = {"QA", "AE", "KW", "BH", "OM", "SA"}
NONGULF_ARAB = ["Egypt", "Iraq", "Jordan", "Lebanon", "Syrian Arab Republic", "Palestine, State of", "Yemen", "Morocco", "Algeria", "Tunisia", "Libya", "Sudan"]

def share_ok(cc, name, kind):
    key = ("first_name" if kind == "first" else "last_name", name)
    if key not in _share:
        r = _nd.search(name)[key[0]]
        _share[key] = r["country"] if r else {}
    dist = _share[key]
    if not dist:
        return False
    cname = pycountry.countries.get_alpha_2 if False else pycountry.countries.get(alpha_2=cc).name
    share = dist.get(cname, 0)
    if share < 0.02 or share < 0.25 * max(dist.values()):
        return False
    if cc in GULF:
        # migrant-heavy countries: keep only names of Arab origin
        return sum(dist.get(c, 0) for c in NONGULF_ARAB) >= 0.3 * sum(dist.values())
    return True

FIRST_AS_LAST = {"RU", "KZ", "BG", "MD", "CZ", "PL", "LT", "TM", "AZ", "GE", "EE", "SI", "HR", "RS", "HU"}
_firsts = {}
def firsts(cc):
    if cc not in _firsts:
        names = _nd.get_top_names(n=700, use_first_names=True, country_alpha2=cc)[cc]
        _firsts[cc] = {x.casefold() for g in names.values() for x in g}
    return _firsts[cc]

SHORT_OK = {"CN", "TW", "HK", "MO", "SG", "KR", "VN", "MY", "JP", "KH", "ID", "TH", "BN"}

def nd_list(cc, kind, limit=1600):
    out, seen = [], set()
    for n in ND[cc][kind]:
        n = clean(n)
        if not n or n.casefold() in seen:
            continue
        if cc not in NATIVE_DIACRITICS and any(ord(c) > 127 for c in n):
            continue
        if len(n) < 3 and cc not in SHORT_OK:
            continue
        low = _fold(n).lower()
        if kind == "last" and any(low.endswith(suf) for suf in FEM.get(cc, ())):
            continue
        if kind == "last" and cc in FIRST_AS_LAST and n.casefold() in firsts(cc):
            continue
        if not share_ok(cc, n, kind):
            continue
        seen.add(n.casefold())
        out.append(n)
        if len(out) >= limit:
            break
    return out

def dk(n):
    return n.replace("\u0130", "i").casefold()

def keep(code, name, pct):
    h = int(hashlib.md5(f"{code}:{name}".encode()).hexdigest(), 16) % 100
    return h < pct

# --- specs ---------------------------------------------------------------------------------------
DONORS = {
 # UEFA
 "ad": ({"ES": .5, "FR": .3, "PT": .2}, []),
 "am": ({"RU": .5, "GE": .3, "LB": .2}, ["armenian"]),
 "ba": ({"RS": .35, "HR": .35, "AL": .15, "TR": .15}, []),
 "by": ({"RU": .55, "PL": .25, "LT": .2}, []),
 "fo": ({"DK": .4, "NO": .3, "IS": .3}, ["faroese"]),
 "gi": ({"GB": .4, "ES": .35, "MT": .15, "PT": .1}, []),
 "li": ({"CH": .4, "DE": .3, "AT": .3}, []),
 "lv": ({"LT": .5, "EE": .3, "RU": .2}, ["latvian"]),
 "me": ({"RS": .55, "HR": .2, "AL": .15, "TR": .1}, []),
 "mk": ({"RS": .4, "BG": .3, "AL": .3}, []),
 "ro": ({"MD": .5, "HU": .2, "BG": .15, "TR": .15}, ["romanian"]),
 "sk": ({"CZ": .6, "PL": .2, "HU": .2}, []),
 "sm": ({"IT": .9, "ES": .1}, []),
 "ua": ({"RU": .6, "PL": .2, "MD": .2}, ["ukrainian"]),
 "xk": ({"AL": .7, "RS": .15, "TR": .15}, []),
 # CONCACAF
 "ai": ({"JM": .6, "US": .2, "GB": .2}, []),
 "ag": ({"JM": .6, "US": .2, "GB": .2}, []),
 "bs": ({"JM": .6, "US": .25, "GB": .15}, []),
 "bb": ({"JM": .6, "US": .15, "GB": .25}, []),
 "bz": ({"JM": .4, "GT": .3, "US": .15, "GB": .15}, []),
 "bm": ({"JM": .4, "US": .2, "GB": .3, "PT": .1}, []),
 "vg": ({"JM": .6, "US": .2, "GB": .2}, []),
 "ky": ({"JM": .6, "GB": .2, "US": .2}, []),
 "dm": ({"JM": .6, "US": .15, "GB": .15, "HT": .1}, []),
 "gd": ({"JM": .6, "US": .15, "GB": .25}, []),
 "kn": ({"JM": .6, "US": .2, "GB": .2}, []),
 "lc": ({"JM": .55, "US": .15, "GB": .15, "HT": .15}, []),
 "vc": ({"JM": .6, "US": .2, "GB": .2}, []),
 "tt": ({"JM": .5, "IN": .25, "US": .15, "GB": .1}, []),
 "tc": ({"JM": .5, "HT": .2, "US": .2, "GB": .1}, []),
 "ms": ({"JM": .6, "GB": .25, "US": .15}, []),
 "vi": ({"JM": .5, "US": .3, "PR": .2}, []),
 "gy": ({"JM": .35, "IN": .35, "GB": .15, "US": .15}, []),
 "aw": ({"NL": .4, "PR": .25, "CO": .2, "US": .15}, ["papiamento_sranan"]),
 "cw": ({"NL": .4, "DO": .0, "PR": .25, "CO": .2, "US": .15}, ["papiamento_sranan"]),
 "sr": ({"NL": .5, "IN": .3, "ID": .2}, ["papiamento_sranan"]),
 "cu": ({"ES": .3, "MX": .3, "PR": .2, "CO": .2}, []),
 "do": ({"PR": .4, "MX": .3, "ES": .15, "US": .15}, []),
 "ni": ({"HN": .3, "CR": .3, "MX": .2, "GT": .2}, []),
 # CONMEBOL
 "ve": ({"CO": .4, "ES": .2, "IT": .1, "PT": .1, "PE": .2}, []),
 "py": ({"AR": .35, "BR": .15, "ES": .15, "UY": .15, "DE": .1, "CL": .1}, []),
 # AFC
 "au": ({"GB": .5, "IT": .12, "GR": .1, "CN": .1, "IN": .1, "DE": .08}, []),
 "bt": ({"IN": .6, "BD": .4}, ["nepali_bhutanese"]),
 "gu": ({"PH": .5, "US": .3, "ES": .2}, []),
 "kg": ({"KZ": .6, "RU": .2, "TR": .2}, ["uzbek_turkic"]),
 "la": ({"KH": .5, "MY": .3, "ID": .2}, ["burmese_lao_tetum"]),
 "mm": ({"KH": .3, "ID": .3, "IN": .2, "BD": .2}, ["burmese_lao_tetum"]),
 "mn": ({"KZ": .4, "CN": .4, "RU": .2}, ["mongolian"]),
 "np": ({"IN": .8, "BD": .2}, ["nepali_bhutanese"]),
 "kp": ({"KR": .85, "CN": .15}, []),
 "pk": ({"IN": .35, "AF": .25, "BD": .15, "IR": .15, "SA": .1}, []),
 "lk": ({"IN": .7, "MV": .3}, ["sinhalese"]),
 "tj": ({"IR": .4, "AF": .3, "KZ": .3}, ["uzbek_turkic"]),
 "tl": ({"PT": .3, "ID": .3, "PH": .2, "BR": .2}, ["burmese_lao_tetum"]),
 "th": ({"CN": .4, "MY": .2, "KH": .2, "SG": .2}, ["thai"]),
 "vn": ({"CN": .3, "TW": .3, "KH": .2, "HK": .2}, ["vietnamese"]),
 "uz": ({"KZ": .5, "TM": .3, "RU": .2}, ["uzbek_turkic"]),
 # CAF
 "bj": ({"BF": .35, "CM": .25, "NG": .2, "FR": .2}, ["west_african_francophone"]),
 "cv": ({"PT": .4, "BR": .3, "AO": .3}, []),
 "cf": ({"CM": .5, "BF": .3, "FR": .2}, []),
 "td": ({"CM": .3, "SD": .3, "BF": .2, "FR": .2}, []),
 "km": ({"SA": .3, "YE": .2, "MA": .1, "FR": .2, "TZ": .0, "MG": .0, "DJ": .2}, []),
 "cg": ({"CM": .5, "FR": .3, "AO": .2}, []),
 "cd": ({"CM": .4, "AO": .3, "FR": .15, "BI": .15}, []),
 "gq": ({"ES": .5, "CM": .3, "AO": .2}, []),
 "er": ({"ET": .7, "SD": .3}, []),
 "sz": ({"ZA": .6, "NA": .2, "BW": .2}, ["southern_bantu"]),
 "ga": ({"CM": .5, "FR": .3, "AO": .2}, []),
 "gm": ({"GH": .35, "NG": .35, "SD": .15, "GB": .15}, ["west_african_francophone"]),
 "gn": ({"BF": .5, "CM": .25, "FR": .25}, ["west_african_francophone"]),
 "gw": ({"PT": .3, "AO": .3, "BF": .2, "BR": .2}, ["west_african_francophone"]),
 "ls": ({"ZA": .8, "NA": .1, "BW": .1}, ["southern_bantu"]),
 "lr": ({"GH": .4, "NG": .3, "US": .3}, []),
 "mg": ({"FR": .5, "MU": .3, "CM": .2}, ["malagasy"]),
 "mw": ({"ZA": .3, "BW": .2, "NA": .2, "GH": .3}, ["southern_bantu"]),
 "mr": ({"MA": .3, "DZ": .3, "SD": .2, "SA": .2}, ["west_african_francophone"]),
 "mz": ({"PT": .4, "AO": .3, "BR": .15, "ZA": .15}, []),
 "ne": ({"BF": .4, "NG": .3, "DJ": .1, "FR": .2}, ["west_african_francophone"]),
 "rw": ({"BI": .6, "FR": .2, "ET": .2}, ["east_african"]),
 "st": ({"PT": .5, "AO": .3, "BR": .2}, []),
 "sc": ({"FR": .4, "MU": .4, "GB": .2}, []),
 "sl": ({"NG": .4, "GH": .4, "GB": .2}, []),
 "so": ({"DJ": .3, "SA": .2, "ET": .2, "YE": .15, "EG": .15}, ["somali"]),
 "ss": ({"SD": .35, "ET": .25, "BI": .15, "GB": .15, "NG": .1}, ["east_african"]),
 "tg": ({"BF": .3, "GH": .3, "CM": .2, "FR": .2}, ["west_african_francophone"]),
 "ug": ({"GB": .3, "BI": .3, "ET": .15, "GH": .15, "NG": .1}, ["east_african"]),
 "zm": ({"ZA": .35, "NA": .2, "BW": .2, "GH": .25}, ["southern_bantu"]),
 "zw": ({"ZA": .5, "BW": .2, "GB": .15, "NA": .15}, ["southern_bantu"]),
 "ci": ({"BF": .4, "CM": .3, "GH": .1, "FR": .2}, ["west_african_francophone"]),
 "sn": ({"CM": .2, "BF": .3, "MA": .15, "FR": .2, "GH": .15}, ["west_african_francophone"]),
 "ke": ({"GB": .3, "BI": .25, "ET": .15, "GH": .1, "IN": .1, "ZA": .1}, ["east_african"]),
 "tz": ({"GB": .25, "BI": .25, "OM": .15, "IN": .15, "ET": .1, "ZA": .1}, ["east_african"]),
 "ml": ({"BF": .5, "DZ": .1, "MA": .15, "FR": .1, "CM": .15}, ["west_african_francophone"]),
 # Gulf: the dataset lists for these are mostly migrant workers' names, so they draw on the Arab world
 "qa": ({"SA": .45, "IQ": .15, "JO": .1, "EG": .1, "YE": .1, "SY": .1}, ["gulf_arab"]),
 "ae": ({"SA": .4, "IQ": .15, "JO": .1, "EG": .15, "YE": .1, "SY": .1}, ["gulf_arab"]),
 "kw": ({"SA": .4, "IQ": .2, "JO": .1, "EG": .1, "SY": .1, "LB": .1}, ["gulf_arab"]),
 "bh": ({"SA": .35, "IQ": .25, "IR": .1, "JO": .1, "EG": .1, "YE": .1}, ["gulf_arab"]),
 "om": ({"YE": .35, "SA": .3, "IQ": .1, "EG": .1, "SD": .15}, ["gulf_arab"]),
 "sa": ({"IQ": .25, "JO": .15, "EG": .2, "YE": .2, "SY": .1, "LB": .1}, ["gulf_arab"]),
 # OFC
 "as": ({"US": .5, "FJ": .3, "PH": .2}, ["polynesian"]),
 "ck": ({"FJ": .4, "GB": .3, "US": .3}, ["polynesian"]),
 "nc": ({"FR": .6, "FJ": .25, "MU": .15}, ["melanesian"]),
 "nz": ({"GB": .55, "FJ": .15, "IN": .1, "CN": .1, "ZA": .1}, ["polynesian"]),
 "pg": ({"ID": .3, "PH": .3, "FJ": .4}, ["melanesian"]),
 "ws": ({"FJ": .4, "US": .3, "GB": .3}, ["polynesian"]),
 "sb": ({"FJ": .5, "GB": .3, "PH": .2}, ["melanesian"]),
 "pf": ({"FR": .6, "FJ": .2, "CN": .2}, ["polynesian"]),
 "to": ({"FJ": .4, "GB": .3, "US": .3}, ["polynesian"]),
 "vu": ({"FR": .4, "FJ": .4, "GB": .2}, ["melanesian"]),
}
# Direct countries whose own dataset list is short: topped up from these.
SUPPLEMENT = {"SV": {"GT": .4, "HN": .4, "MX": .2}, "FJ": {"IN": .5, "GB": .3, "AU_": 0, "US": .2}, "KH": {"TH_": 0, "MY": .5, "SG": .5},
              "HN": {"GT": .5, "MX": .3, "CR": .2}, "BF": {"CM": .6, "FR": .4},
              "YE": {"SA": .5, "OM": .5}, "KZ": {"RU": .6, "AZ": .2, "TM": .2}}
HOME = {"en": "GB", "sco": "GB", "wal": "GB", "nir": "GB"}
HOME_SEEDS = {"sco": ["scottish"], "wal": ["welsh"], "nir": ["northern_irish"], "en": []}
HOME_SHARE = {"sco": 0, "wal": 1, "nir": 2}


def weighted_merge(code, kind, donors, target, already, core=60, pct=72):
    lists = {cc: nd_list(cc, kind) for cc, w in donors.items() if w > 0 and cc in ND}
    weights = {cc: w for cc, w in donors.items() if cc in lists}
    taken = {cc: 0 for cc in lists}
    ptr = {cc: 0 for cc in lists}
    seen = {dk(n) for n in already}
    out = []
    while len(already) + len(out) < target:
        # donor furthest below its weight share
        cands = [cc for cc in lists if ptr[cc] < len(lists[cc])]
        if not cands:
            break
        cc = min(cands, key=lambda c: taken[c] / weights[c])
        name = lists[cc][ptr[cc]]
        rank = ptr[cc]
        ptr[cc] += 1
        if dk(name) in seen:
            continue
        if rank >= core and not keep(code, name, pct):
            continue
        seen.add(dk(name))
        out.append(name)
        taken[cc] += 1
    return out


import itertools, random

def compose(code, kind):
    rng = random.Random(f"compose-{code}-{kind}")
    out = []
    if code == "vn":
        mids = "Van Huu Duc Quang Minh Xuan Quoc Dinh Cong Ngoc Thanh Hoang Gia The Trong Tien Viet Anh Bao Phu Trung".split()
        givens = "Anh Bao Binh Cuong Dat Dung Duy Hai Hieu Hoang Hung Huy Khanh Khoa Kien Lam Long Minh Nam Nghia Nhan Phat Phong Phuc Quan Quang Son Tai Tam Thang Thanh Tien Trung Tuan Tung Vinh Viet Vu Hoa Giang Hanh Hiep Hoai Khai Luan Nguyen Sang Thien Thinh Toan Tri Truong Cuong Dang Dong Duong Hao Hien Kiet Lap Linh Loc Manh Nhat Ninh Phu Quyen Quyet Tan Thai Thuan Tu Tuyen Vy".split()
        fams = "Nguyen Tran Le Pham Hoang Huynh Phan Vu Vo Dang Bui Do Ho Ngo Duong Ly Truong Dinh Doan Lam Mai Trinh Ha Luong Tong Cao Lai Quach Ta Chu Kieu Thai Han Vuong Phung Diep Tang La Chau".split()
        if kind == "first":
            cand = [f"{m} {g}" for m, g in itertools.product(mids, givens)]
            rng.shuffle(cand)
            out = givens + cand
        else:
            cand = [f"{f} {m}" for f, m in itertools.product(fams, mids)]
            rng.shuffle(cand)
            out = fams + cand
    elif code == "kr" and kind == "last":
        fams = "Kim Lee Park Choi Jung Kang Cho Yoon Jang Lim Han Oh Seo Shin Kwon Hwang Ahn Song Yoo Hong".split()
        seats = "Gimhae Gyeongju Miryang Jeonju Andong Pyeongchang Cheongju Haeju Jinju Naju Gwangsan Suwon Dongnae Hamyang Uiryeong Sunchang Yeoju Pyeongyang Anseong Ulsan Gangneung Seongju Nampyeong Gaeseong Hwangju Cheongsong Jangheung Gimpo Yangcheon Bugwon Cheongpung Dalseong Gwangju Haman Hadong Jeonui Muan Yeonan Gongju".split()
        cand = [f"{f} {x}" for f, x in itertools.product(fams, seats)]
        rng.shuffle(cand)
        out = cand
    elif code == "th":
        pre = "Som Pra Thana Chai Wira Kitti Suthi Pong Anu Sak Nara Pichit Chalerm Boon Sura Wat Ek Teera Nat Pat Kan Sun Sathit Thawee Wichit Songk Kriang Sompo Prach Ruang Surat Thira Worap Yut Pit Pon Mon Ar Chat Chan Ni Nit Ka Si Wanch Tha Than".split()
        suf = "chai sak pong wat korn rat phon dej sit wit thep nan pat pol phan rit san sin sant sawat rung kiat chok phum yot dit".split()
        spre = "Sri Wong Chai Boon Rat Kaew Thong Suk Sang Pra Jit Mee Pan Pho Sa Than Wan Kit Chan Nak Duang Sawat Phet Ruang Som Sin Sam Thip Kham Charoen Intha Wat Yin Pak Lert".split()
        ssuf = "sai sawat rat kul chai suk nont wan pong thip kaew sri wong rit tham phan mee boon rung sin thai yot pat kasem lert pol phak dee".split()
        if kind == "first":
            cand = [a + b for a, b in itertools.product(pre, suf)]
        else:
            cand = [a + b for a, b in itertools.product(spre, ssuf)]
        cand = [c.capitalize() if not c[0].isupper() else c for c in cand]
        cand = [c for c in cand if c[0].isupper() and len(c) <= 14]
        rng.shuffle(cand)
        out = cand
    return out

COMPOSED = {"vn", "th"}

def build(code, existing):
    res = {}
    for kind, target in (("first", N_FIRST), ("last", N_LAST)):
        head = []
        def add(names):
            seen = {dk(h) for h in head}
            for n in names:
                n2 = clean(n, 2, 30)
                if n2 and dk(n2) not in seen:
                    head.append(n2); seen.add(dk(n2))
        add(existing.get((code, kind), []))
        idx = 0 if kind == "first" else 1
        seeds = []
        if code in DONORS:
            for s in DONORS[code][1]:
                seeds += SEEDS[s][idx]
        elif code in HOME_SEEDS:
            for s in HOME_SEEDS[code]:
                seeds += SEEDS[s][idx]
        add(seeds)
        if code in COMPOSED:
            add(compose(code, kind)[:target])
        elif code == "kr" and kind == "last":
            add(nd_list("KR", "last")[:100])
            add(compose("kr", "last")[:target])
        elif code in DONORS:
            add(weighted_merge(code, kind, DONORS[code][0], target, head))
        elif code in HOME:
            gb = nd_list("GB", kind)
            if code == "en":
                add(gb[:target])
            else:
                add(gb[:120 if kind == "first" else 150])
                part = HOME_SHARE[code]
                extra = [n for n in gb[120:2500] if int(hashlib.md5(n.encode()).hexdigest(), 16) % 3 == part]
                if code == "nir":
                    ie = nd_list("IE", kind)
                    extra = [x for pair in zip(extra, ie[60:]) for x in pair]
                add(extra[:target])
        else:
            cc = code.upper()
            add(nd_list(cc, kind)[:target])
            if len(head) < target and cc in SUPPLEMENT:
                add(weighted_merge(code, kind, {k: v for k, v in SUPPLEMENT[cc].items() if v > 0}, target, head, core=0, pct=100))
        res[kind] = head[:max(target, len(head) if len(head) < target else target)]
    return res


def main():
    existing = existing_pools()
    # the 63 original codes except es
    orig = re.findall(r'Code: "(\w+)"', (REPO / "backend" / "common" / "countries.go").read_text())
    allcodes = sorted(set(orig) - {"es"})
    short = []
    for code in allcodes:
        r = build(code, existing)
        d = OUT / code
        d.mkdir(parents=True, exist_ok=True)
        (d / "first.txt").write_text("\n".join(r["first"]) + "\n", encoding="utf-8")
        (d / "last.txt").write_text("\n".join(r["last"]) + "\n", encoding="utf-8")
        if len(r["first"]) < 300 or len(r["last"]) < 500:
            short.append((code, len(r["first"]), len(r["last"])))
    print(len(allcodes), "countries; short:", short)

main()
