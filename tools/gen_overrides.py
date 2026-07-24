"""Hand-maintained Go handler entries for cases gen_handlers.py cannot translate.

OVERRIDES is keyed by (field, python_pattern) -- the exact pattern string from
the vendored PTT source. When PTT changes one of these patterns upstream, the
key stops matching, the generator reports it unresolved, and the entry here
must be re-verified against the new upstream pattern. That is deliberate:
these entries encode RE2 workarounds for lookarounds Go cannot express.

CUSTOM maps a PTT custom-handler (or pattern-factory) name to a Go expression
defined in parser/handlers_custom.go.

Seeded from the pre-generator hand-ported Go table (MunifTanjim/go-ptt); the
ported PTT test corpus arbitrates semantic equivalence.
"""

CUSTOM: dict[str, str] = {
    "create_adult_pattern()": "\tcustom_adult,",
    "create_adult_pattern": "\tcustom_adult,",
    "handle_bit_depth": "\tcustom_handle_bit_depth,",
    "handle_space_in_codec": "\tcustom_handle_space_in_codec,",
    "handle_volumes": "\tcustom_handle_volumes,",
    "handle_episodes": "\tcustom_handle_episodes,",
    "handle_anime_eps": "\tcustom_handle_anime_eps,",
    "infer_language_based_on_naming": "\tcustom_infer_language_based_on_naming,",
    "handle_group": "\tcustom_handle_group,",
    "handle_group_exclusion": "\tcustom_handle_group_exclusion,",
}

OVERRIDES: dict[tuple[str, str], str] = {}

# --- handlers.py:49 field=scene why=pattern
# py pattern: '^(?=.*(\\b\\d{3,4}p\\b).*([_. ]WEB[_. ])(?!DL)\\b)|\\b(-CAKES|-GGEZ|-GGWP|-GLHF|-GOSSIP|-NAISU|-KOGI|-PECULATE|-SLOT|-EDITH|-ETHEL|-ELEANOR|-B2B|-SPAMnEGGS|-FTP|-DiRT|-SYNCOPY|-BAE|-SuccessfulCrab|-NHTFS|-SURCODE|-B0MBARDIERS)'
# transformer: boolean
# options: {'remove': False}
# NO GOOD MATCH (best ratio=0.00) — hand-translate
OVERRIDES[('scene', '^(?=.*(\\b\\d{3,4}p\\b).*([_. ]WEB[_. ])(?!DL)\\b)|\\b(-CAKES|-GGEZ|-GGWP|-GLHF|-GOSSIP|-NAISU|-KOGI|-PECULATE|-SLOT|-EDITH|-ETHEL|-ELEANOR|-B2B|-SPAMnEGGS|-FTP|-DiRT|-SYNCOPY|-BAE|-SuccessfulCrab|-NHTFS|-SURCODE|-B0MBARDIERS)')] = r'''
	custom_scene,
'''

# --- handlers.py:59 field=extras why=pattern
# py pattern: '(?:(?<=\\b(?:19\\d{2}|20\\d{2})\\b.*)\\b(?:Featurettes?)\\b|\\bFeaturettes?\\b(?!.*\\b(?:19\\d{2}|20\\d{2})\\b))'
# transformer: uniq_concat(value('Featurette'))
# options: {'skipFromTitle': True, 'remove': False}
# NO GOOD MATCH (best ratio=0.00) — hand-translate
OVERRIDES[('extras', '(?:(?<=\\b(?:19\\d{2}|20\\d{2})\\b.*)\\b(?:Featurettes?)\\b|\\bFeaturettes?\\b(?!.*\\b(?:19\\d{2}|20\\d{2})\\b))')] = r'''
	custom_extras_featurette,
'''

# --- handlers.py:60 field=extras why=pattern
# py pattern: '(?:(?<=\\b(?:19\\d{2}|20\\d{2})\\b.*)\\b(?:Sample)\\b|\\b(?:Sample)\\b(?!.*\\b(?:19\\d{2}|20\\d{2})\\b))'
# transformer: uniq_concat(value('Sample'))
# options: {'skipFromTitle': True, 'remove': False}
# NO GOOD MATCH (best ratio=0.00) — hand-translate
OVERRIDES[('extras', '(?:(?<=\\b(?:19\\d{2}|20\\d{2})\\b.*)\\b(?:Sample)\\b|\\b(?:Sample)\\b(?!.*\\b(?:19\\d{2}|20\\d{2})\\b))')] = r'''
	custom_extras_sample,
'''

# --- handlers.py:61 field=extras why=pattern
# py pattern: '(?:(?<=\\b(?:19\\d{2}|20\\d{2})\\b.*)\\b(?:Trailers?)\\b|\\bTrailers?\\b(?!.*\\b(?:19\\d{2}|20\\d{2}|.(Park|And))\\b))'
# transformer: uniq_concat(value('Trailer'))
# options: {'skipFromTitle': True, 'remove': False}
# NO GOOD MATCH (best ratio=0.00) — hand-translate
OVERRIDES[('extras', '(?:(?<=\\b(?:19\\d{2}|20\\d{2})\\b.*)\\b(?:Trailers?)\\b|\\bTrailers?\\b(?!.*\\b(?:19\\d{2}|20\\d{2}|.(Park|And))\\b))')] = r'''
	custom_extras_trailer,
'''

# --- handlers.py:99 field=trash why=pattern
# py pattern: '\\b(?:H[DQ][ .-]*)?CAM(?!.?(S|E|\\()\\d+)(?:H[DQ])?(?:[ .-]*Rip|Rp)?\\b'
# transformer: boolean
# options: {'remove': False}
# NO GOOD MATCH (best ratio=0.00) — hand-translate
OVERRIDES[('trash', '\\b(?:H[DQ][ .-]*)?CAM(?!.?(S|E|\\()\\d+)(?:H[DQ])?(?:[ .-]*Rip|Rp)?\\b')] = r'''
	custom_trash_cam,
'''

# --- handlers.py:113 field=date why=transformer:date('YYYY MM DD')
# py pattern: '(?:\\W|^)([[(]?(?:19[6-9]|20[012])[0-9]([. \\-/\\\\])(?:0[1-9]|1[012])\\2(?:0[1-9]|[12][0-9]|3[01])[])]?)(?:\\W|$)'
# transformer: date('YYYY MM DD')
# options: {'remove': True}
# MATCH ratio=0.98 old go pattern: '(?:\\W|^)([(\\[]?((?:19[6-9]|20[012])[0-9]([. \\-/\\\\])(?:0[1-9]|1[012])([. \\-/\\\\])(?:0[1-9]|[12][0-9]|3'
#   // parser.addHandler("date", /(?<=\W|^)([([]?(?:19[6-9]|20[012])[0-9]([. \-/\\])(?:0[1-9]|1[012])\2(?:0[1-9]|[12][0-9]|3[01])[)\]]?)(?=\W|$)/, date("YYYY MM DD"), { remove: true });
#   // parser.addHandler("date", /(?<=\W|^)([([]?(?:0[1-9]|[12][0-9]|3[01])([. \-/\\])(?:0[1-9]|1[012])\2(?:19[6-9]|20[012])[0-9][)\]]?)(?=\W|$)/, date("DD MM YYYY"), { remove: true });
#   // parser.addHandler("date", /(?<=\W)([([]?(?:0[1-9]|1[012])([. \-/\\])(?:0[1-9]|[12][0-9]|3[01])\2(?:19[6-9]|20[012])[0-9][)\]]?)(?=\W|$)/, date("MM DD YYYY"), { remove: true });
#   // parser.addHandler("date", /(?<=\W)([([]?(?:0[1-9]|1[012])([. \-/\\])(?:0[1-9]|[12][0-9]|3[01])\2(?:[0][1-9]|[0126789][0-9])[)\]]?)(?=\W|$)/, date("MM DD YY"), { remove: true });
#   // parser.addHandler("date", /(?<=\W)([([]?(?:0[1-9]|[12][0-9]|3[01])([. \-/\\])(?:0[1-9]|1[012])\2(?:[0][1-9]|[0126789][0-9])[)\]]?)(?=\W|$)/, date("DD MM YY"), { remove: true });
#   // parser.addHandler("date", /(?<=\W|^)([([]?(?:0?[1-9]|[12][0-9]|3[01])[. ]?(?:st|nd|rd|th)?([. \-/\\])(?:feb(?:ruary)?|jan(?:uary)?|mar(?:ch)?|apr(?:il)?|may|june?|july?|aug(?:ust)?|sept?(?:ember)?|oct(?:ober)?|nov(?:ember)?|dec(?:ember)?)\2(?:19[7-9]|20[012])[0-9][)\]]?)(?=\W|$)/i, date("DD MMM YYYY"), { remove: true });
#   // parser.addHandler("date", /(?<=\W|^)([([]?(?:0?[1-9]|[12][0-9]|3[01])[. ]?(?:st|nd|rd|th)?([. \-/\\])(?:feb(?:ruary)?|jan(?:uary)?|mar(?:ch)?|apr(?:il)?|may|june?|july?|aug(?:ust)?|sept?(?:ember)?|oct(?:ober)?|nov(?:ember)?|dec(?:ember)?)\2(?:0[1-9]|[0126789][0-9])[)\]]?)(?=\W|$)/i, date("DD MMM YY"), { remove: true });
#   // parser.addHandler("date", /(?<=\W|^)([([]?20[012][0-9](?:0[1-9]|1[012])(?:0[1-9]|[12][0-9]|3[01])[)\]]?)(?=\W|$)/, date("YYYYMMDD"), { remove: true });
OVERRIDES[('date', '(?:\\W|^)([[(]?(?:19[6-9]|20[012])[0-9]([. \\-/\\\\])(?:0[1-9]|1[012])\\2(?:0[1-9]|[12][0-9]|3[01])[])]?)(?:\\W|$)')] = r'''
	{
		Field:         "date",
		Pattern:       regexp.MustCompile(`(?:\W|^)([[(]?(?:19[6-9]|20[012])[0-9]([. \-/\\])(?:0[1-9]|1[012])([. \-/\\])(?:0[1-9]|[12][0-9]|3[01])[])]?)(?:\W|$)`),
		ValidateMatch: validate_matched_groups_are_same(2, 3),
		Transform:     to_ptt_date(`YYYY MM DD`),
		Remove:        true,
	},
'''

# --- handlers.py:114 field=date why=transformer:date('DD MM YYYY')
# py pattern: '(?:\\W|^)(\\[?\\]?(?:0[1-9]|[12][0-9]|3[01])([. \\-/\\\\])(?:0[1-9]|1[012])\\2(?:19[6-9]|20[01])[0-9][\\])]?)(?:\\W|$)'
# transformer: date('DD MM YYYY')
# options: {'remove': True}
# MATCH ratio=0.97 old go pattern: '(?:\\W|^)[(\\[]?((?:0[1-9]|[12][0-9]|3[01])([. \\-/\\\\])(?:0[1-9]|1[012])([. \\-/\\\\])(?:19[6-9]|20[012])['
OVERRIDES[('date', '(?:\\W|^)(\\[?\\]?(?:0[1-9]|[12][0-9]|3[01])([. \\-/\\\\])(?:0[1-9]|1[012])\\2(?:19[6-9]|20[01])[0-9][\\])]?)(?:\\W|$)')] = r'''
	{
		Field:         "date",
		Pattern:       regexp.MustCompile(`(?:\W|^)(\[?\]?(?:0[1-9]|[12][0-9]|3[01])([. \-/\\])(?:0[1-9]|1[012])([. \-/\\])(?:19[6-9]|20[01])[0-9][\])]?)(?:\W|$)`),
		ValidateMatch: validate_matched_groups_are_same(2, 3),
		Transform:     to_ptt_date(`DD MM YYYY`),
		Remove:        true,
	},
'''

# --- handlers.py:115 field=date why=transformer:date('MM DD YY')
# py pattern: '(?:\\W)(\\[?\\]?(?:0[1-9]|1[012])([. \\-/\\\\])(?:0[1-9]|[12][0-9]|3[01])\\2(?:[0][1-9]|[0126789][0-9])[\\])]?)(?:\\W|$)'
# transformer: date('MM DD YY')
# options: {'remove': True}
# MATCH ratio=0.98 old go pattern: '(?:\\W)[(\\[]?((?:0[1-9]|1[012])([. \\-/\\\\])(?:0[1-9]|[12][0-9]|3[01])([. \\-/\\\\])(?:[0][1-9]|[0126789]['
OVERRIDES[('date', '(?:\\W)(\\[?\\]?(?:0[1-9]|1[012])([. \\-/\\\\])(?:0[1-9]|[12][0-9]|3[01])\\2(?:[0][1-9]|[0126789][0-9])[\\])]?)(?:\\W|$)')] = r'''
	{
		Field:         "date",
		Pattern:       regexp.MustCompile(`(?:\W)(\[?\]?(?:0[1-9]|1[012])([. \-/\\])(?:0[1-9]|[12][0-9]|3[01])([. \-/\\])(?:[0][1-9]|[0126789][0-9])[\])]?)(?:\W|$)`),
		ValidateMatch: validate_matched_groups_are_same(2, 3),
		Transform:     to_ptt_date(`MM DD YY`),
		Remove:        true,
	},
'''

# --- handlers.py:116 field=date why=transformer:date('DD MM YY')
# py pattern: '(?:\\W)(\\[?\\]?(?:0[1-9]|[12][0-9]|3[01])([. \\-/\\\\])(?:0[1-9]|1[012])\\2(?:[0][1-9]|[0126789][0-9])[\\])]?)(?:\\W|$)'
# transformer: date('DD MM YY')
# options: {'remove': True}
# MATCH ratio=0.98 old go pattern: '(?:\\W)[(\\[]?((?:0[1-9]|[12][0-9]|3[01])([. \\-/\\\\])(?:0[1-9]|1[012])([. \\-/\\\\])(?:[0][1-9]|[0126789]['
OVERRIDES[('date', '(?:\\W)(\\[?\\]?(?:0[1-9]|[12][0-9]|3[01])([. \\-/\\\\])(?:0[1-9]|1[012])\\2(?:[0][1-9]|[0126789][0-9])[\\])]?)(?:\\W|$)')] = r'''
	{
		Field:         "date",
		Pattern:       regexp.MustCompile(`(?:\W)(\[?\]?(?:0[1-9]|[12][0-9]|3[01])([. \-/\\])(?:0[1-9]|1[012])([. \-/\\])(?:[0][1-9]|[0126789][0-9])[\])]?)(?:\W|$)`),
		ValidateMatch: validate_matched_groups_are_same(2, 3),
		Transform:     to_ptt_date(`DD MM YY`),
		Remove:        true,
	},
'''

# --- handlers.py:117 field=date why=transformer:date(['DD MMM YYYY', 'Do MMM YYYY', 'Do MMMM YYYY'])
# py pattern: '(?:\\W|^)([([]?(?:0?[1-9]|[12][0-9]|3[01])[. ]?(?:st|nd|rd|th)?([. \\-/\\\\])(?:feb(?:ruary)?|jan(?:uary)?|mar(?:ch)?|apr(?:il)?|may|june?|july?|aug(?:ust)?|sept?(?:ember)?|oct(?:ober)?|nov(?:ember)?|dec(?:ember)?)\\2(?:19[7-9]|20[012])[0-9][)\\]]?)(?=\\W|$)'
# transformer: date(['DD MMM YYYY', 'Do MMM YYYY', 'Do MMMM YYYY'])
# options: {'remove': True}
# MATCH ratio=1.00 old go pattern: '(?i)(?:\\W|^)[(\\[]?((?:0?[1-9]|[12][0-9]|3[01])[. ]?(?:st|nd|rd|th)?([. \\-/\\\\])(?:feb(?:ruary)?|jan(?'
OVERRIDES[('date', '(?:\\W|^)([([]?(?:0?[1-9]|[12][0-9]|3[01])[. ]?(?:st|nd|rd|th)?([. \\-/\\\\])(?:feb(?:ruary)?|jan(?:uary)?|mar(?:ch)?|apr(?:il)?|may|june?|july?|aug(?:ust)?|sept?(?:ember)?|oct(?:ober)?|nov(?:ember)?|dec(?:ember)?)\\2(?:19[7-9]|20[012])[0-9][)\\]]?)(?=\\W|$)')] = r'''
	{
		Field:         "date",
		Pattern:       regexp.MustCompile(`(?i)(?:\W|^)([([]?(?:0?[1-9]|[12][0-9]|3[01])[. ]?(?:st|nd|rd|th)?([. \-/\\])(?:feb(?:ruary)?|jan(?:uary)?|mar(?:ch)?|apr(?:il)?|may|june?|july?|aug(?:ust)?|sept?(?:ember)?|oct(?:ober)?|nov(?:ember)?|dec(?:ember)?)([. \-/\\])(?:19[7-9]|20[012])[0-9][)\]]?)`),
		ValidateMatch: validate_and(validate_matched_groups_are_same(2, 3), validate_lookahead(`\W|$`, ``, true)),
		Transform:     to_ptt_date(`DD MMM YYYY`, `Do MMM YYYY`, `Do MMMM YYYY`),
		Remove:        true,
	},
'''

# --- handlers.py:123 field=date why=transformer:date('DD MMM YY')
# py pattern: '(?:\\W|^)(\\[?\\]?(?:0?[1-9]|[12][0-9]|3[01])[. ]?(?:st|nd|rd|th)?([. \\-\\/\\\\])(?:feb(?:ruary)?|jan(?:uary)?|mar(?:ch)?|apr(?:il)?|may|june?|july?|aug(?:ust)?|sept?(?:ember)?|oct(?:ober)?|nov(?:ember)?|dec(?:ember)?)\\2(?:0[1-9]|[0126789][0-9])[\\])]?)(?:\\W|$)'
# transformer: date('DD MMM YY')
# options: {'remove': True}
# MATCH ratio=1.00 old go pattern: '(?i)(?:\\W|^)[(\\[]?((?:0?[1-9]|[12][0-9]|3[01])[. ]?(?:st|nd|rd|th)?([. \\-/\\\\])(?:feb(?:ruary)?|jan(?'
OVERRIDES[('date', '(?:\\W|^)(\\[?\\]?(?:0?[1-9]|[12][0-9]|3[01])[. ]?(?:st|nd|rd|th)?([. \\-\\/\\\\])(?:feb(?:ruary)?|jan(?:uary)?|mar(?:ch)?|apr(?:il)?|may|june?|july?|aug(?:ust)?|sept?(?:ember)?|oct(?:ober)?|nov(?:ember)?|dec(?:ember)?)\\2(?:0[1-9]|[0126789][0-9])[\\])]?)(?:\\W|$)')] = r'''
	{
		Field:         "date",
		Pattern:       regexp.MustCompile(`(?i)(?:\W|^)(\[?\]?(?:0?[1-9]|[12][0-9]|3[01])[. ]?(?:st|nd|rd|th)?([. \-/\\])(?:feb(?:ruary)?|jan(?:uary)?|mar(?:ch)?|apr(?:il)?|may|june?|july?|aug(?:ust)?|sept?(?:ember)?|oct(?:ober)?|nov(?:ember)?|dec(?:ember)?)([. \-/\\])(?:0[1-9]|[0126789][0-9])[\])]?)(?:\W|$)`),
		ValidateMatch: validate_matched_groups_are_same(2, 3),
		Transform:     to_ptt_date(`DD MMM YY`),
		Remove:        true,
	},
'''

# --- handlers.py:129 field=date why=transformer:date('YYYYMMDD')
# py pattern: '(?:\\W|^)(\\[?\\]?20[012][0-9](?:0[1-9]|1[012])(?:0[1-9]|[12][0-9]|3[01])[\\])]?)(?:\\W|$)'
# transformer: date('YYYYMMDD')
# options: {'remove': True}
# MATCH ratio=1.00 old go pattern: '(?:\\W|^)[(\\[]?(20[012][0-9](?:0[1-9]|1[012])(?:0[1-9]|[12][0-9]|3[01]))[)\\]]?(?:\\W|$)'
OVERRIDES[('date', '(?:\\W|^)(\\[?\\]?20[012][0-9](?:0[1-9]|1[012])(?:0[1-9]|[12][0-9]|3[01])[\\])]?)(?:\\W|$)')] = r'''
	{
		Field:     "date",
		Pattern:   regexp.MustCompile(`(?:\W|^)(\[?\]?20[012][0-9](?:0[1-9]|1[012])(?:0[1-9]|[12][0-9]|3[01])[\])]?)(?:\W|$)`),
		Transform: to_ptt_date(`YYYYMMDD`),
		Remove:    true,
	},
'''

# --- handlers.py:140 field=year why=pattern
# py pattern: '[([]?(?!^)(?<!\\d|Cap[. ]?)((?:19\\d|20[012])\\d)(?!\\d|kbps)[)\\]]?'
# transformer: integer
# options: {'remove': True}
# MATCH ratio=0.95 old go pattern: '(?i)(?:[(\\[*]|.)((?:\\d|[SE]|Cap[. ]?)?(?:19\\d|20[012])\\d(?:\\d|kbps)?)[*)\\]]?'
#   // ~ parser.add_handler("year", regex.compile(r"[^SE][([]?(?!^)(?<!\d|Cap[. ]?)((?:19\d|20[012])\d)(?!\d|kbps)[)\]]?", regex.IGNORECASE), integer, {"remove": True})
OVERRIDES[('year', '[([]?(?!^)(?<!\\d|Cap[. ]?)((?:19\\d|20[012])\\d)(?!\\d|kbps)[)\\]]?')] = r'''
	{
		Field: "year",
		Process: scan_valid("year", regexp.MustCompile(`(?i)[([]?((?:19\d|20[012])\d)[)\]]?`), func(title string, idxs []int) bool {
			if idxs[2] == 0 {
				return false // (?!^) after optional bracket
			}
			return !year_prefix_reject_regex.MatchString(title[:idxs[2]]) && !year_suffix_reject_regex.MatchString(title[idxs[3]:])
		}, false, false, false),
		Transform: to_int_string(),
		Remove:    true,
	},
'''

# --- handlers.py:141 field=year why=pattern
# py pattern: '(?!^\\w{4})^[([]?((?:19\\d|20[012])\\d)(?!\\d|kbps)[)\\]]?'
# transformer: integer
# options: {'remove': True}
# MATCH ratio=0.93 old go pattern: '^[(\\[]?((?:19\\d|20[012])\\d)(?:\\d|kbps)?[)\\]]?'
#   // parser.add_handler("year", regex.compile(r"(?!^\w{4})^[([]?((?:19\d|20[012])\d)(?!\d|kbps)[)\]]?", regex.IGNORECASE), integer, {"remove": True})
OVERRIDES[('year', '(?!^\\w{4})^[([]?((?:19\\d|20[012])\\d)(?!\\d|kbps)[)\\]]?')] = r'''
	{
		Field:   "year",
		Pattern: regexp.MustCompile(`^[(\[]?((?:19\d|20[012])\d)(?:\d|kbps)?[)\]]?`),
		ValidateMatch: func(input string, match []int) bool {
			mValue := input[match[0]:match[1]]
			if len(mValue) == 4 {
				return match[0] != 0
			}
			return len(strings.Trim(mValue, "()[]")) == 4
		},
		Transform: to_year(),
		Remove:    true,
	},
'''

# --- handlers.py:151 field=edition why=pattern
# py pattern: '\\buncut(?!.gems)\\b'
# transformer: value('Uncut')
# options: {'remove': True}
# MATCH ratio=1.00 old go pattern: '(?i)\\buncut(?:.gems)?\\b'
OVERRIDES[('edition', '\\buncut(?!.gems)\\b')] = r'''
	{
		Field:         "edition",
		Pattern:       regexp.MustCompile(`(?i)\buncut(?:.gems)?\b`),
		ValidateMatch: validate_not_match(regexp.MustCompile(`(?i)(?:.gems)`)),
		Transform:     to_value("Uncut"),
		Remove:        true,
	},
'''

# --- handlers.py:224 field=quality why=pattern
# py pattern: '\\b(?<!\\w.)WEB\\b|\\bWEB(?!([ \\.\\-\\(\\],]+\\d))\\b'
# transformer: value('WEB')
# options: {'remove': True, 'skipFromTitle': True}
# MATCH ratio=1.00 old go pattern: '(?i)\\b(?:\\w.)?WEB\\b|\\bWEB(?:(?:[ \\.\\-\\(\\],]+\\d))?\\b'
#   // parser.add_handler("quality", regex.compile(r"\b(?<!\w.)WEB\b|\bWEB(?!([ \.\-\(\],]+\d))\b", regex.IGNORECASE), value("WEB"), {"remove": True, "skipFromTitle": True})
OVERRIDES[('quality', '\\b(?<!\\w.)WEB\\b|\\bWEB(?!([ \\.\\-\\(\\],]+\\d))\\b')] = r'''
	{
		Field:         "quality",
		Pattern:       regexp.MustCompile(`(?i)\b(?:\w.)?WEB\b|\bWEB(?:(?:[ \.\-\(\],]+\d))?\b`),
		ValidateMatch: validate_not_match(regexp.MustCompile(`(?i)\b(?:\w.)WEB\b|\bWEB(?:(?:[ \.\-\(\],]+\d))\b`)),
		Transform:     to_value("WEB"),
		Remove:        true,
		SkipFromTitle: true,
	},
'''

# --- handlers.py:225 field=quality why=pattern
# py pattern: '\\b(?:H[DQ][ .-]*)?CAM(?!.?(S|E|\\()\\d+)(?:H[DQ])?(?:[ .-]*Rip|Rp)?\\b'
# transformer: value('CAM')
# options: {'remove': True, 'skipFromTitle': True}
# MATCH ratio=0.85 old go pattern: '(?i)\\b(?:H[DQ][ .-]*)?CAM(?:H[DQ])?(?:[ .-]*Rip)?\\b'
#   // parser.addHandler("source", /\b(?:H[DQ][ .-]*)?CAM(?:H[DQ])?(?:[ .-]*Rip)?\b/i, value("CAM"), { remove: true });
#   // parser.addHandler("source", /\b(?:H[DQ][ .-]*)?S[ .-]+print/i, value("CAM"), { remove: true });
#   // parser.addHandler("source", /\b(?:HD[ .-]*)?T(?:ELE)?S(?:YNC)?(?:Rip)?\b/i, value("TeleSync"), { remove: true });
#   // parser.addHandler("source", /\b(?:HD[ .-]*)?T(?:ELE)?C(?:INE)?(?:Rip)?\b/, value("TeleCine"), { remove: true });
OVERRIDES[('quality', '\\b(?:H[DQ][ .-]*)?CAM(?!.?(S|E|\\()\\d+)(?:H[DQ])?(?:[ .-]*Rip|Rp)?\\b')] = r'''
	{
		Field: "quality",
		Process: scan_valid("quality", regexp.MustCompile(`(?i)\b(?:H[DQ][ .-]*)?(CAM)(?:H[DQ])?(?:[ .-]*Rip|Rp)?\b`), func(title string, idxs []int) bool {
			return !trash_cam_reject_regex.MatchString(title[idxs[3]:])
		}, false, false, false),
		Transform:     to_value(`CAM`),
		Remove:        true,
		SkipFromTitle: true,
	},
'''

# --- handlers.py:276 field=audio why=pattern
# py pattern: '\\b(?!.+HR)(DTS.?HD.?Ma(ster)?|DTS.?X)\\b'
# transformer: uniq_concat(value('DTS Lossless'))
# options: {'remove': True, 'skipIfAlreadyFound': False}
# MATCH ratio=1.00 old go pattern: '(?i)\\b(?:.+HR)?(?:DTS.?HD.?Ma(?:ster)?|DTS.?X)\\b'
#   // parser.add_handler("audio", regex.compile(r"\b(?!.+HR)(DTS.?HD.?Ma(ster)?|DTS.?X)\b", regex.IGNORECASE), uniq_concat(value("DTS Lossless")), {"remove": True, "skipIfAlreadyFound": False})
#   // parser.add_handler("audio", regex.compile(r"\bDTS(?!(.?HD.?Ma(ster)?|.X)).?(HD.?HR|HD)?\b", regex.IGNORECASE), uniq_concat(value("DTS Lossy")), {"remove": True, "skipIfAlreadyFound": False})
#   // parser.add_handler("audio", regex.compile(r"\b(Dolby.?)?Atmos\b", regex.IGNORECASE), uniq_concat(value("Atmos")), {"remove": True, "skipIfAlreadyFound": False})
#   // parser.add_handler("audio", regex.compile(r"\b(True[ .-]?HD|\.True\.)\b", regex.IGNORECASE), uniq_concat(value("TrueHD")), {"remove": True, "skipIfAlreadyFound": False, "skipFromTitle": True})
#   // parser.add_handler("audio", regex.compile(r"\bTRUE\b"), uniq_concat(value("TrueHD")), {"remove": True, "skipIfAlreadyFound": False, "skipFromTitle": True})
OVERRIDES[('audio', '\\b(?!.+HR)(DTS.?HD.?Ma(ster)?|DTS.?X)\\b')] = r'''
	{
		Field: "audio",
		Process: scan_valid("audio", regexp.MustCompile(`(?i)\b(DTS.?HD.?Ma(?:ster)?|DTS.?X)\b`), func(title string, idxs []int) bool {
			return !audio_hr_after_regex.MatchString(title[idxs[0]:])
		}, true, false, true),
		Transform:    to_value_set(`DTS Lossless`),
		KeepMatching: true,
		Remove:       true,
	},
'''

# --- handlers.py:277 field=audio why=pattern
# py pattern: '\\bDTS(?!(.?HD.?Ma(ster)?|.X)).?(HD.?HR|HD)?\\b'
# transformer: uniq_concat(value('DTS Lossy'))
# options: {'remove': True, 'skipIfAlreadyFound': False}
# MATCH ratio=1.00 old go pattern: '(?i)\\bDTS(?:(?:.?HD.?Ma(?:ster)?|.X))?.?(?:HD.?HR|HD)?\\b'
OVERRIDES[('audio', '\\bDTS(?!(.?HD.?Ma(ster)?|.X)).?(HD.?HR|HD)?\\b')] = r'''
	{
		Field:         "audio",
		Pattern:       regexp.MustCompile(`(?i)\bDTS(?:(?:.?HD.?Ma(?:ster)?|.X))?.?(?:HD.?HR|HD)?\b`),
		ValidateMatch: validate_not_match(regexp.MustCompile(`(?i)DTS(?:.?HD.?Ma(?:ster)?|.X)`)),
		Transform:     to_value_set("DTS Lossy"),
		Remove:        true,
		KeepMatching:  true,
	},
'''

# --- handlers.py:290 field=group why=pattern
# py pattern: '- ?(?!\\d+$|S\\d+|\\d+x|ep?\\d+|[^[]+]$)([^\\-. []+[^\\-. [)\\]\\d][^\\-. [)\\]]*)(?:\\[[\\w.-]+])?(?=\\.\\w{2,4}$|$)'
# transformer: none
# options: {'remove': False}
# NO GOOD MATCH (best ratio=0.47) — hand-translate
OVERRIDES[('group', '- ?(?!\\d+$|S\\d+|\\d+x|ep?\\d+|[^[]+]$)([^\\-. []+[^\\-. [)\\]\\d][^\\-. [)\\]]*)(?:\\[[\\w.-]+])?(?=\\.\\w{2,4}$|$)')] = r'''
	custom_group_dash,
'''

# --- handlers.py:349 field=seasons why=pattern
# py pattern: '(?:(?:\\bthe\\W)?\\bcomplete)?(?<![a-z])\\bs(\\d{1,3})(?:[\\Wex]|\\d{2}\\b|$)'
# transformer: array(integer)
# options: {'remove': False, 'skipIfAlreadyFound': False}
# MATCH ratio=1.00 old go pattern: '(?i)(?:(?:\\bthe\\W)?\\bcomplete)?(?:[a-z])?\\bs(\\d{1,3})(?:[\\Wex]|\\d{2}\\b|$)'
#   // parser.add_handler("seasons", regex.compile(r"(?:(?:\bthe\W)?\bcomplete)?(?<![a-z])\bs(\d{1,3})(?:[\Wex]|\d{2}\b|$)", regex.IGNORECASE), array(integer), {"remove": False, "skipIfAlreadyFound": False})
OVERRIDES[('seasons', '(?:(?:\\bthe\\W)?\\bcomplete)?(?<![a-z])\\bs(\\d{1,3})(?:[\\Wex]|\\d{2}\\b|$)')] = r'''
	{
		Field:         "seasons",
		Pattern:       regexp.MustCompile(`(?i)(?:(?:\bthe\W)?\bcomplete)?(?:[a-z])?\bs(\d{1,3})(?:[\Wex]|\d{2}\b|$)`),
		ValidateMatch: validate_not_match(regexp.MustCompile(`(?i)(?:[a-z])\bs\d{1,3}`)),
		Transform:     to_int_array(),
		KeepMatching:  true,
	},
'''

# --- handlers.py:375 field=episodes why=pattern
# py pattern: '-\\s(\\d{1,3}[ .]*-[ .]*\\d{1,3})(?!-\\d)(?:\\W|$)'
# transformer: range_func
# MATCH ratio=1.00 old go pattern: '(?i)-\\s(\\d{1,3}[ .]*-[ .]*\\d{1,3})(?:-\\d*)?(?:\\W|$)'
OVERRIDES[('episodes', '-\\s(\\d{1,3}[ .]*-[ .]*\\d{1,3})(?!-\\d)(?:\\W|$)')] = r'''
	{
		Field:         "episodes",
		Pattern:       regexp.MustCompile(`(?i)-\s(\d{1,3}[ .]*-[ .]*\d{1,3})(?:-\d*)?(?:\W|$)`),
		ValidateMatch: validate_not_match(regexp.MustCompile(`(?i)-\s(\d{1,3}[ .]*-[ .]*\d{1,3})(?:-\d*)`)),
		Transform:     to_int_range(),
	},
'''

# --- handlers.py:393 field=episodes why=pattern
# py pattern: '(?<!^)\\[(?!720|1080)(\\d{2,3})](?!(?:\\.\\w{2,4})?$)'
# transformer: array(integer)
# MATCH ratio=0.63 old go pattern: '(?i)(?:^)?\\[(\\d{2,3})](?:(?:\\.\\w{2,4})?$)?'
OVERRIDES[('episodes', '(?<!^)\\[(?!720|1080)(\\d{2,3})](?!(?:\\.\\w{2,4})?$)')] = r'''
	{
		Field:   "episodes",
		Pattern: regexp.MustCompile(`(?i)(?:^)?\[(\d{2,3})](?:(?:\.\w{2,4})?$)?`),
		ValidateMatch: validate_and(
			validate_not_at_start(),
			validate_not_at_end(),
			validate_not_match(regexp.MustCompile(`(?i)(?:720|1080)|\[(\d{2,3})](?:(?:\.\w{2,4})$)`)),
		),
		Transform: to_int_array(),
	},
'''

# --- handlers.py:396 field=episodes why=pattern
# py pattern: '(?<!\\bMovie\\s-\\s)(?<=\\s-\\s)\\d+(?=\\s[-(\\s])'
# transformer: array(integer)
# options: {'remove': True, 'skipIfAlreadyFound': True}
# NO GOOD MATCH (best ratio=0.38) — hand-translate
OVERRIDES[('episodes', '(?<!\\bMovie\\s-\\s)(?<=\\s-\\s)\\d+(?=\\s[-(\\s])')] = r'''
	{
		Field:         "episodes",
		Pattern:       regexp.MustCompile(`\s-\s(\d+)`),
		ValidateMatch: validate_and(validate_lookbehind(`\bMovie`, ``, false), validate_lookahead(`\s[-(\s]`, ``, true)),
		MatchGroup:    1,
		Transform:     to_int_array(),
		Remove:        true,
	},
'''

# --- handlers.py:480 field=languages why=pattern
# py pattern: '\\bes(?=[ .,/-]+(?:[A-Z]{2}[ .,/-]+){2,})\\b'
# transformer: uniq_concat(value('es'))
# options: {'skipFromTitle': True, 'skipIfAlreadyFound': False}
# MATCH ratio=0.67 old go pattern: '(?i)\\beng?sub[A-Z]*\\b'
OVERRIDES[('languages', '\\bes(?=[ .,/-]+(?:[A-Z]{2}[ .,/-]+){2,})\\b')] = r'''
	{
		Field: "languages",
		Process: scan_valid("languages", regexp.MustCompile(`(?i)\bes\b`), func(title string, idxs []int) bool {
			return lang_codes2_suffix.MatchString(title[idxs[1]:])
		}, true, false, true),
		Transform:     to_value_set(`es`),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
'''

# --- handlers.py:481 field=languages why=pattern
# py pattern: '\\b(?<=[ .,/-]+(?:[A-Z]{2}[ .,/-]+){2,})es\\b'
# transformer: uniq_concat(value('es'))
# options: {'skipFromTitle': True, 'skipIfAlreadyFound': False}
# MATCH ratio=0.71 old go pattern: '(?i)\\bRO(?:[ .,/-]*(?:[A-Z]{2}[ .,/-]+)*sub)'
OVERRIDES[('languages', '\\b(?<=[ .,/-]+(?:[A-Z]{2}[ .,/-]+){2,})es\\b')] = r'''
	{
		Field: "languages",
		Process: scan_valid("languages", regexp.MustCompile(`(?i)\bes\b`), func(title string, idxs []int) bool {
			return lang_codes2_prefix.MatchString(title[:idxs[0]])
		}, true, false, true),
		Transform:     to_value_set(`es`),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
'''

# --- handlers.py:482 field=languages why=pattern
# py pattern: '\\b(?<=[ .,/-]+[A-Z]{2}[ .,/-]+)es(?=[ .,/-]+[A-Z]{2}[ .,/-]+)\\b'
# transformer: uniq_concat(value('es'))
# options: {'skipFromTitle': True, 'skipIfAlreadyFound': False}
# MATCH ratio=0.63 old go pattern: '(?i)\\bRO(?:[ .,/-]*(?:[A-Z]{2}[ .,/-]+)*sub)'
OVERRIDES[('languages', '\\b(?<=[ .,/-]+[A-Z]{2}[ .,/-]+)es(?=[ .,/-]+[A-Z]{2}[ .,/-]+)\\b')] = r'''
	{
		Field: "languages",
		Process: scan_valid("languages", regexp.MustCompile(`(?i)\bes\b`), func(title string, idxs []int) bool {
			return lang_code1_prefix.MatchString(title[:idxs[0]]) && lang_code1_suffix.MatchString(title[idxs[1]:])
		}, true, false, true),
		Transform:     to_value_set(`es`),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
'''

# --- handlers.py:497 field=languages why=pattern
# py pattern: '\\b(?<!w{3}\\.\\w+\\.)IT(?=[ .,/-]+(?:[a-zA-Z]{2}[ .,/-]+){2,})\\b'
# transformer: uniq_concat(value('it'))
# options: {'skipFromTitle': True, 'skipIfAlreadyFound': False}
# MATCH ratio=0.56 old go pattern: '(?i)\\bITA\\b'
#   // parser.addHandler("languages", /\bITA\b/i, uniqConcat(value("italian")), { skipIfAlreadyFound: false });
OVERRIDES[('languages', '\\b(?<!w{3}\\.\\w+\\.)IT(?=[ .,/-]+(?:[a-zA-Z]{2}[ .,/-]+){2,})\\b')] = r'''
	{
		Field: "languages",
		Process: scan_valid("languages", regexp.MustCompile(`\bIT\b`), func(title string, idxs []int) bool {
			return !lang_www_cs.MatchString(title[:idxs[0]]) && lang_it_codes_cs.MatchString(title[idxs[1]:])
		}, true, false, true),
		Transform:     to_value_set(`it`),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
'''

# --- handlers.py:502 field=languages why=pattern
# py pattern: '\\bde(?=[ .,/-]+(?:[A-Z]{2}[ .,/-]+){2,})\\b'
# transformer: uniq_concat(value('de'))
# options: {'skipFromTitle': True, 'skipIfAlreadyFound': False}
# MATCH ratio=0.67 old go pattern: '(?i)\\bde\\b'
#   // parser.addHandler("languages", /\bde(?=[ .,/-]+(?:[A-Z]{2}[ .,/-]+){2,})\b/i, uniqConcat(value("german")), { skipFromTitle: true, skipIfAlreadyFound: false });
OVERRIDES[('languages', '\\bde(?=[ .,/-]+(?:[A-Z]{2}[ .,/-]+){2,})\\b')] = r'''
	{
		Field: "languages",
		Process: scan_valid("languages", regexp.MustCompile(`(?i)\bde\b`), func(title string, idxs []int) bool {
			return lang_codes2_suffix.MatchString(title[idxs[1]:])
		}, true, false, true),
		Transform:     to_value_set(`de`),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
'''

# --- handlers.py:503 field=languages why=pattern
# py pattern: '\\b(?<=[ .,/-]+(?:[A-Z]{2}[ .,/-]+){2,})de\\b'
# transformer: uniq_concat(value('de'))
# options: {'skipFromTitle': True, 'skipIfAlreadyFound': False}
# MATCH ratio=0.67 old go pattern: '(?i)\\bde\\b'
#   // parser.addHandler("languages", /\bde(?=[ .,/-]+(?:[A-Z]{2}[ .,/-]+){2,})\b/i, uniqConcat(value("german")), { skipFromTitle: true, skipIfAlreadyFound: false });
OVERRIDES[('languages', '\\b(?<=[ .,/-]+(?:[A-Z]{2}[ .,/-]+){2,})de\\b')] = r'''
	{
		Field: "languages",
		Process: scan_valid("languages", regexp.MustCompile(`(?i)\bde\b`), func(title string, idxs []int) bool {
			return lang_codes2_prefix.MatchString(title[:idxs[0]])
		}, true, false, true),
		Transform:     to_value_set(`de`),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
'''

# --- handlers.py:504 field=languages why=pattern
# py pattern: '\\b(?<=[ .,/-]+[A-Z]{2}[ .,/-]+)de(?=[ .,/-]+[A-Z]{2}[ .,/-]+)\\b'
# transformer: uniq_concat(value('de'))
# options: {'skipFromTitle': True, 'skipIfAlreadyFound': False}
# MATCH ratio=0.57 old go pattern: '(?i)\\bde\\b'
#   // parser.addHandler("languages", /\bde(?=[ .,/-]+(?:[A-Z]{2}[ .,/-]+){2,})\b/i, uniqConcat(value("german")), { skipFromTitle: true, skipIfAlreadyFound: false });
OVERRIDES[('languages', '\\b(?<=[ .,/-]+[A-Z]{2}[ .,/-]+)de(?=[ .,/-]+[A-Z]{2}[ .,/-]+)\\b')] = r'''
	{
		Field: "languages",
		Process: scan_valid("languages", regexp.MustCompile(`(?i)\bde\b`), func(title string, idxs []int) bool {
			return lang_code1_prefix.MatchString(title[:idxs[0]]) && lang_code1_suffix.MatchString(title[idxs[1]:])
		}, true, false, true),
		Transform:     to_value_set(`de`),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
'''

# --- handlers.py:512 field=languages why=pattern
# py pattern: '\\b(?:(?<!w{3}\\.\\w+\\.)tel(?!\\W*aviv)|telugu)\\b'
# transformer: uniq_concat(value('te'))
# options: {'remove': True, 'skipIfAlreadyFound': False}
# MATCH ratio=1.00 old go pattern: '(?i)\\b(?:(?:w{3}\\.\\w+\\.)?tel(?:\\W*aviv)?|telugu)\\b'
#   // parser.addHandler("languages", /\b(?:(?<!w{3}\.\w+\.)tel(?!\W*aviv)|telugu)\b/i, uniqConcat(value("telugu")), { skipIfAlreadyFound: false });
OVERRIDES[('languages', '\\b(?:(?<!w{3}\\.\\w+\\.)tel(?!\\W*aviv)|telugu)\\b')] = r'''
	{
		Field: "languages",
		Process: scan_valid("languages", regexp.MustCompile(`(?i)\b(?:tel|telugu)\b`), lang_abbr_valid(map[string]bool{"telugu": true}, func(title string, idxs []int) bool {
			return !lang_te_aviv.MatchString(title[idxs[1]:])
		}), true, false, true),
		Transform:    to_value_set(`te`),
		KeepMatching: true,
		Remove:       true,
	},
'''

# --- handlers.py:514 field=languages why=pattern
# py pattern: '\\b(?:(?<!w{3}\\.\\w+\\.)MAL(?:ay)?|malayalam)\\b'
# transformer: uniq_concat(value('ml'))
# options: {'remove': True, 'skipIfFirst': True, 'skipIfAlreadyFound': False}
# MATCH ratio=1.00 old go pattern: '(?i)\\b(?:(?:w{3}\\.\\w+\\.)?MAL(?:ay)?|malayalam)\\b'
#   // parser.add_handler("languages", regex.compile(r"\b(?:(?<!w{3}\.\w+\.)MAL(?:ay)?|malayalam)\b", regex.IGNORECASE), uniq_concat(value("ml")), {"remove": True, "skipIfFirst": True, "skipIfAlreadyFound": False})
OVERRIDES[('languages', '\\b(?:(?<!w{3}\\.\\w+\\.)MAL(?:ay)?|malayalam)\\b')] = r'''
	{
		Field: "languages",
		Process: scan_valid("languages", regexp.MustCompile(`(?i)\b(?:MAL(?:ay)?|malayalam)\b`), lang_abbr_valid(map[string]bool{"malayalam": true}, nil), true, true, true),
		Transform:    to_value_set(`ml`),
		KeepMatching: true,
		Remove:       true,
	},
'''

# --- handlers.py:515 field=languages why=pattern
# py pattern: '\\b(?:(?<!w{3}\\.\\w+\\.)KAN(?:nada)?|kannada)\\b'
# transformer: uniq_concat(value('kn'))
# options: {'remove': True, 'skipIfAlreadyFound': False}
# MATCH ratio=1.00 old go pattern: '(?i)\\b(?:(?:w{3}\\.\\w+\\.)?KAN(?:nada)?|kannada)\\b'
#   // parser.add_handler("languages", regex.compile(r"\b(?:(?<!w{3}\.\w+\.)KAN(?:nada)?|kannada)\b", regex.IGNORECASE), uniq_concat(value("kn")), {"remove": True, "skipIfAlreadyFound": False})
OVERRIDES[('languages', '\\b(?:(?<!w{3}\\.\\w+\\.)KAN(?:nada)?|kannada)\\b')] = r'''
	{
		Field: "languages",
		Process: scan_valid("languages", regexp.MustCompile(`(?i)\b(?:KAN(?:nada)?|kannada)\b`), lang_abbr_valid(map[string]bool{"kannada": true}, nil), true, false, true),
		Transform:    to_value_set(`kn`),
		KeepMatching: true,
		Remove:       true,
	},
'''

# --- handlers.py:516 field=languages why=pattern
# py pattern: '\\b(?:(?<!w{3}\\.\\w+\\.)MAR(?:a(?:thi)?)?|marathi)\\b'
# transformer: uniq_concat(value('mr'))
# options: {'skipIfAlreadyFound': False}
# MATCH ratio=1.00 old go pattern: '(?i)\\b(?:(?:w{3}\\.\\w+\\.)?MAR(?:a(?:thi)?)?|marathi)\\b'
#   // parser.add_handler("languages", regex.compile(r"\b(?:(?<!w{3}\.\w+\.)MAR(?:a(?:thi)?)?|marathi)\b", regex.IGNORECASE), uniq_concat(value("mr")), {"skipIfAlreadyFound": False})
OVERRIDES[('languages', '\\b(?:(?<!w{3}\\.\\w+\\.)MAR(?:a(?:thi)?)?|marathi)\\b')] = r'''
	{
		Field: "languages",
		Process: scan_valid("languages", regexp.MustCompile(`(?i)\b(?:MAR(?:a(?:thi)?)?|marathi)\b`), lang_abbr_valid(map[string]bool{"marathi": true}, nil), true, false, true),
		Transform:    to_value_set(`mr`),
		KeepMatching: true,
	},
'''

# --- handlers.py:517 field=languages why=pattern
# py pattern: '\\b(?:(?<!w{3}\\.\\w+\\.)GUJ(?:arati)?|gujarati)\\b'
# transformer: uniq_concat(value('gu'))
# options: {'skipIfAlreadyFound': False}
# MATCH ratio=1.00 old go pattern: '(?i)\\b(?:(?:w{3}\\.\\w+\\.)?GUJ(?:arati)?|gujarati)\\b'
#   // parser.add_handler("languages", regex.compile(r"\b(?:(?<!w{3}\.\w+\.)GUJ(?:arati)?|gujarati)\b", regex.IGNORECASE), uniq_concat(value("gu")), {"skipIfAlreadyFound": False})
OVERRIDES[('languages', '\\b(?:(?<!w{3}\\.\\w+\\.)GUJ(?:arati)?|gujarati)\\b')] = r'''
	{
		Field: "languages",
		Process: scan_valid("languages", regexp.MustCompile(`(?i)\b(?:GUJ(?:arati)?|gujarati)\b`), lang_abbr_valid(map[string]bool{"gujarati": true}, nil), true, false, true),
		Transform:    to_value_set(`gu`),
		KeepMatching: true,
	},
'''

# --- handlers.py:518 field=languages why=pattern
# py pattern: '\\b(?:(?<!w{3}\\.\\w+\\.)PUN(?:jabi)?|punjabi)\\b'
# transformer: uniq_concat(value('pa'))
# options: {'skipIfAlreadyFound': False}
# MATCH ratio=1.00 old go pattern: '(?i)\\b(?:(?:w{3}\\.\\w+\\.)?PUN(?:jabi)?|punjabi)\\b'
#   // parser.add_handler("languages", regex.compile(r"\b(?:(?<!w{3}\.\w+\.)PUN(?:jabi)?|punjabi)\b", regex.IGNORECASE), uniq_concat(value("pa")), {"skipIfAlreadyFound": False})
OVERRIDES[('languages', '\\b(?:(?<!w{3}\\.\\w+\\.)PUN(?:jabi)?|punjabi)\\b')] = r'''
	{
		Field: "languages",
		Process: scan_valid("languages", regexp.MustCompile(`(?i)\b(?:PUN(?:jabi)?|punjabi)\b`), lang_abbr_valid(map[string]bool{"punjabi": true}, nil), true, false, true),
		Transform:    to_value_set(`pa`),
		KeepMatching: true,
	},
'''

# --- handlers.py:519 field=languages why=pattern
# py pattern: '\\b(?:(?<!w{3}\\.\\w+\\.)BEN(?!.\\bThe|and|of\\b)(?:gali)?|bengali)\\b'
# transformer: uniq_concat(value('bn'))
# options: {'skipIfFirst': True, 'skipIfAlreadyFound': False}
# MATCH ratio=1.00 old go pattern: '(?i)\\b(?:(?:w{3}\\.\\w+\\.)?BEN(?:.\\bThe|and|of\\b)?(?:gali)?|bengali)\\b'
#   // parser.add_handler("languages", regex.compile(r"\b(?:(?<!w{3}\.\w+\.)BEN(?!.\bThe|and|of\b)(?:gali)?|bengali)\b", regex.IGNORECASE), uniq_concat(value("bn")), {"skipIfFirst": True, "skipIfAlreadyFound": False})
OVERRIDES[('languages', '\\b(?:(?<!w{3}\\.\\w+\\.)BEN(?!.\\bThe|and|of\\b)(?:gali)?|bengali)\\b')] = r'''
	{
		Field: "languages",
		Process: scan_valid("languages", regexp.MustCompile(`(?i)\b(?:BEN(?:gali)?|bengali)\b`), lang_abbr_valid(map[string]bool{"bengali": true}, func(title string, idxs []int) bool {
			if strings.EqualFold(title[idxs[0]:idxs[1]], "ben") {
				return !lang_bn_reject.MatchString(title[idxs[1]:])
			}
			return true
		}), true, true, true),
		Transform:    to_value_set(`bn`),
		KeepMatching: true,
	},
'''

# --- handlers.py:520 field=languages why=pattern
# py pattern: '\\b(?<!YTS\\.)LT\\b'
# transformer: uniq_concat(value('lt'))
# options: {'skipFromTitle': True, 'skipIfAlreadyFound': False}
# MATCH ratio=1.00 old go pattern: '(?i)\\b(?:YTS\\.)?LT\\b'
#   // parser.addHandler("languages", /\b(?<!YTS\.)LT\b/, uniqConcat(value("lithuanian")), { skipFromTitle: true, skipIfAlreadyFound: false });
#   // parser.addHandler("languages", /\blithuanian\b/i, uniqConcat(value("lithuanian")), { skipIfFirst: true, skipIfAlreadyFound: false });
OVERRIDES[('languages', '\\b(?<!YTS\\.)LT\\b')] = r'''
	{
		Field: "languages",
		Process: scan_valid("languages", regexp.MustCompile(`\bLT\b`), func(title string, idxs []int) bool {
			return !lang_yts_prefix.MatchString(title[:idxs[0]])
		}, true, false, true),
		Transform:     to_value_set(`lt`),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
'''

# --- handlers.py:524 field=languages why=pattern
# py pattern: '\\b(?:(?<!w{3}\\.\\w+\\.)PL|pol)\\b'
# transformer: uniq_concat(value('pl'))
# options: {'skipIfAlreadyFound': False}
# MATCH ratio=1.00 old go pattern: '(?i)\\b(?:(?:w{3}\\.\\w+\\.)?PL|pol)\\b'
OVERRIDES[('languages', '\\b(?:(?<!w{3}\\.\\w+\\.)PL|pol)\\b')] = r'''
	{
		Field: "languages",
		Process: scan_valid("languages", regexp.MustCompile(`(?i)\b(?:PL|pol)\b`), lang_abbr_valid(map[string]bool{"pol": true}, nil), true, false, true),
		Transform:    to_value_set(`pl`),
		KeepMatching: true,
	},
'''

# --- handlers.py:537 field=languages why=pattern
# py pattern: '\\bHR(?=[ .,/-]*(?:[A-Z]{2}[ .,/-]+)*sub)\\b'
# transformer: uniq_concat(value('hr'))
# options: {'skipIfAlreadyFound': False}
# MATCH ratio=0.95 old go pattern: '(?i)\\bHR(?:[ .,/-]*(?:[A-Z]{2}[ .,/-]+)*sub\\w*)\\b'
OVERRIDES[('languages', '\\bHR(?=[ .,/-]*(?:[A-Z]{2}[ .,/-]+)*sub)\\b')] = r'''
	{
		Field: "languages",
		Process: scan_valid("languages", regexp.MustCompile(`(?i)\bHR\b`), func(title string, idxs []int) bool {
			return lang_hr_sub_suffix.MatchString(title[idxs[1]:])
		}, true, false, true),
		Transform:    to_value_set(`hr`),
		KeepMatching: true,
	},
'''

# --- handlers.py:539 field=languages why=pattern
# py pattern: '\\b(?:(?<!w{3}\\.\\w+\\.)NL|dut|holand[eê]s)\\b'
# transformer: uniq_concat(value('nl'))
# options: {'skipIfAlreadyFound': False}
# MATCH ratio=1.00 old go pattern: '(?i)\\b(?:(?:w{3}\\.\\w+\\.)?NL|dut|holand[eê]s)\\b'
#   // parser.addHandler("languages", /\b(?:(?<!w{3}\.\w+\.)NL|dut|holand[eê]s)\b/i, uniqConcat(value("dutch")), { skipIfAlreadyFound: false });
#   // parser.addHandler("languages", /\bdutch\b/i, uniqConcat(value("dutch")), { skipFromTitle: true, skipIfAlreadyFound: false });
#   // parser.addHandler("languages", /\bflemish\b/i, uniqConcat(value("dutch")), { skipIfAlreadyFound: false });
OVERRIDES[('languages', '\\b(?:(?<!w{3}\\.\\w+\\.)NL|dut|holand[eê]s)\\b')] = r'''
	{
		Field: "languages",
		Process: scan_valid("languages", regexp.MustCompile(`(?i)\b(?:NL|dut|holand[eê]s)\b`), lang_abbr_valid(map[string]bool{"dut": true, "holandes": true, "holandês": true}, nil), true, false, true),
		Transform:    to_value_set(`nl`),
		KeepMatching: true,
	},
'''

# --- handlers.py:545 field=languages why=pattern
# py pattern: '\\b(?:(?<!w{3}\\.\\w+\\.|Sci-)FI|finsk|finsub|nordic)\\b'
# transformer: uniq_concat(value('fi'))
# options: {'skipIfAlreadyFound': False}
# MATCH ratio=1.00 old go pattern: '(?i)\\b(?:(?:w{3}\\.\\w+\\.|Sci-)?FI|finsk|finsub|nordic)\\b'
#   // parser.addHandler("languages", /\b(?:(?<!w{3}\.\w+\.)FI|finsk|finsub|nordic)\b/i, uniqConcat(value("finnish")), { skipIfAlreadyFound: false });
#   // parser.addHandler("languages", /\bfinnish\b/i, uniqConcat(value("finnish")), { skipFromTitle: true, skipIfAlreadyFound: false });
OVERRIDES[('languages', '\\b(?:(?<!w{3}\\.\\w+\\.|Sci-)FI|finsk|finsub|nordic)\\b')] = r'''
	{
		Field: "languages",
		Process: scan_valid("languages", regexp.MustCompile(`(?i)\b(?:FI|finsk|finsub|nordic)\b`), lang_abbr_valid(map[string]bool{"finsk": true, "finsub": true, "nordic": true}, func(title string, idxs []int) bool {
			return !lang_sci_i.MatchString(title[:idxs[0]])
		}), true, false, true),
		Transform:    to_value_set(`fi`),
		KeepMatching: true,
	},
'''

# --- handlers.py:547 field=languages why=pattern
# py pattern: '\\b(?:(?<!w{3}\\.\\w+\\.)SE|swe|swesubs?|sv(?:ensk)?|nordic)\\b'
# transformer: uniq_concat(value('sv'))
# options: {'skipIfAlreadyFound': False}
# MATCH ratio=1.00 old go pattern: '(?i)\\b(?:(?:w{3}\\.\\w+\\.)?SE|swe|swesubs?|sv(?:ensk)?|nordic)\\b'
#   // parser.addHandler("languages", /\b(?:(?<!w{3}\.\w+\.)SE|swe|swesubs?|sv(?:ensk)?|nordic)\b/i, uniqConcat(value("swedish")), { skipIfAlreadyFound: false });
#   // parser.addHandler("languages", /\b(swedish|sueco)\b/i, uniqConcat(value("swedish")), { skipFromTitle: true, skipIfAlreadyFound: false });
OVERRIDES[('languages', '\\b(?:(?<!w{3}\\.\\w+\\.)SE|swe|swesubs?|sv(?:ensk)?|nordic)\\b')] = r'''
	{
		Field: "languages",
		Process: scan_valid("languages", regexp.MustCompile(`(?i)\b(?:SE|swe|swesubs?|sv(?:ensk)?|nordic)\b`), lang_abbr_valid(map[string]bool{"swe": true, "swesub": true, "swesubs": true, "sv": true, "svensk": true, "nordic": true}, nil), true, false, true),
		Transform:    to_value_set(`sv`),
		KeepMatching: true,
	},
'''

# --- handlers.py:550 field=languages why=pattern
# py pattern: '\\b(norwegian|noruegu[eê]s|bokm[aå]l|nob|nor(?=[\\]_)]?\\.\\w{2,4}$))\\b'
# transformer: uniq_concat(value('no'))
# options: {'skipFromTitle': True, 'skipIfAlreadyFound': False}
# MATCH ratio=1.00 old go pattern: '(?i)\\b(norwegian|noruegu[eê]s|bokm[aå]l|nob|nor(?:[\\]_)]?\\.\\w{2,4}$))\\b'
OVERRIDES[('languages', '\\b(norwegian|noruegu[eê]s|bokm[aå]l|nob|nor(?=[\\]_)]?\\.\\w{2,4}$))\\b')] = r'''
	{
		Field: "languages",
		Process: scan_valid("languages", regexp.MustCompile(`(?i)\b(?:norwegian|noruegu[eê]s|bokm[aå]l|nob|nor)\b`), func(title string, idxs []int) bool {
			if strings.EqualFold(title[idxs[0]:idxs[1]], "nor") {
				return lang_ext_suffix.MatchString(title[idxs[1]:])
			}
			return true
		}, true, false, true),
		Transform:     to_value_set(`no`),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
'''

# --- handlers.py:556 field=languages why=pattern
# py pattern: '\\bvietnamese\\b|\\bvie(?=[\\]_)]?\\.\\w{2,4}$)'
# transformer: uniq_concat(value('vi'))
# options: {'skipFromTitle': True, 'skipIfAlreadyFound': False}
# MATCH ratio=1.00 old go pattern: '(?i)\\bvietnamese\\b|\\bvie(?:[\\]_)]?\\.\\w{2,4}$)'
#   // parser.addHandler("languages", /\bvietnamese\b|\bvie(?=[\]_)]?\.\w{2,4}$)/i, uniqConcat(value("vietnamese")), { skipFromTitle: true, skipIfAlreadyFound: false });
OVERRIDES[('languages', '\\bvietnamese\\b|\\bvie(?=[\\]_)]?\\.\\w{2,4}$)')] = r'''
	{
		Field: "languages",
		Process: scan_valid("languages", regexp.MustCompile(`(?i)\b(?:vietnamese\b|vie)`), func(title string, idxs []int) bool {
			if strings.EqualFold(title[idxs[0]:idxs[1]], "vie") {
				return lang_ext_suffix.MatchString(title[idxs[1]:])
			}
			return true
		}, true, false, true),
		Transform:     to_value_set(`vi`),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
'''

# --- handlers.py:560 field=languages why=pattern
# py pattern: '\\b(?:malay|may(?=[\\]_)]?\\.\\w{2,4}$)|(?<=subs?\\([a-z,]+)may)\\b'
# transformer: uniq_concat(value('ms'))
# options: {'skipIfFirst': True, 'skipIfAlreadyFound': False}
# MATCH ratio=1.00 old go pattern: '(?i)\\b(?:malay|may(?:[\\]_)]?\\.\\w{2,4}$)|(?:subs?\\([a-z,]+)may)\\b'
#   // parser.addHandler("languages", /\b(?:malay|may(?=[\]_)]?\.\w{2,4}$)|(?<=subs?\([a-z,]+)may)\b/i, uniqConcat(value("malay")), { skipIfFirst: true, skipIfAlreadyFound: false });
OVERRIDES[('languages', '\\b(?:malay|may(?=[\\]_)]?\\.\\w{2,4}$)|(?<=subs?\\([a-z,]+)may)\\b')] = r'''
	{
		Field: "languages",
		Process: scan_valid("languages", regexp.MustCompile(`(?i)\b(?:malay|may)\b`), func(title string, idxs []int) bool {
			if strings.EqualFold(title[idxs[0]:idxs[1]], "may") {
				return lang_ext_suffix.MatchString(title[idxs[1]:]) || lang_subs_paren.MatchString(title[:idxs[0]])
			}
			return true
		}, true, true, true),
		Transform:    to_value_set(`ms`),
		KeepMatching: true,
	},
'''

# --- handlers.py:603 field=dubbed why=pattern
# py pattern: '\\b(?!.*\\bsub(s|bed)?\\b)([ _\\-\\[(\\.])?(dual|multi)([ _\\-\\[(\\.])?(audio)\\b'
# transformer: boolean
# options: {'remove': True}
# MATCH ratio=1.00 old go pattern: '(?i)\\b(?:.*\\bsub(?:s|bed)?\\b)?(?:[ _\\-\\[(\\.])?(dual|multi)(?:[ _\\-\\[(\\.])?(?:audio)\\b'
OVERRIDES[('dubbed', '\\b(?!.*\\bsub(s|bed)?\\b)([ _\\-\\[(\\.])?(dual|multi)([ _\\-\\[(\\.])?(audio)\\b')] = r'''
	{
		Field: "dubbed",
		Process: scan_valid("dubbed", regexp.MustCompile(`(?i)(?:[ _\-\[(\.])?(?:dual|multi)(?:[ _\-\[(\.])?audio\b`), func(title string, idxs []int) bool {
			return !dubbed_subs_after.MatchString(title[idxs[0]:])
		}, false, false, false),
		Transform: to_boolean(),
		Remove:    true,
	},
'''
