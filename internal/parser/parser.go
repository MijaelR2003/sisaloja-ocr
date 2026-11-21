package parser

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// DocumentFields contiene los campos estructurados de un huésped extraídos de su documento.
type DocumentFields struct {
	DocumentType   string `json:"document_type"`
	DocumentNumber string `json:"document_number"`
	FullName       string `json:"full_name"`
	Nationality    string `json:"nationality"`
	Gender         string `json:"gender"`
	BirthDate      string `json:"birth_date"`
	Occupation     string `json:"occupation"`
	OriginCity     string `json:"origin_city"`
}

// ParseResult representa el resultado de analizar el texto de una o ambas caras de un documento.
type ParseResult struct {
	Service     string         `json:"service"`
	Engine      string         `json:"engine"`
	Confidence  string         `json:"confidence"`
	DetectedMRZ bool           `json:"detected_mrz"`
	Fields      DocumentFields `json:"fields"`
}

var countryCodeMap = map[string]string{
	"BOL": "Boliviana",
	"PER": "Peruana",
	"ARG": "Argentina",
	"CHL": "Chilena",
	"BRA": "Brasileña",
	"COL": "Colombiana",
	"ECU": "Ecuatoriana",
	"PRY": "Paraguaya",
	"URY": "Uruguaya",
	"VEN": "Venezolana",
	"MEX": "Mexicana",
	"ESP": "Española",
	"USA": "Estadounidense",
}

var monthsES = map[string]string{
	"ENE": "01", "ENERO": "01",
	"FEB": "02", "FEBRERO": "02",
	"MAR": "03", "MARZO": "03",
	"ABR": "04", "ABRIL": "04",
	"MAY": "05", "MAYO": "05",
	"JUN": "06", "JUNIO": "06",
	"JUL": "07", "JULIO": "07",
	"AGO": "08", "AGOSTO": "08",
	"SEP": "09", "SET": "09", "SEPTIEMBRE": "09", "SETIEMBRE": "09",
	"OCT": "10", "OCTUBRE": "10",
	"NOV": "11", "NOVIEMBRE": "11",
	"DIC": "12", "DICIEMBRE": "12",
}

type cityRule struct {
	re   *regexp.Regexp
	city string
}

var boliviaCities = []cityRule{
	{regexp.MustCompile(`(?i)\b(LA\s+PAZ|MURILLO|EL\s+ALTO|VIACHA|ACHACACHI|COPACABANA|CARANAVI)\b`), "La Paz"},
	{regexp.MustCompile(`(?i)\b(SANTA\s+CRUZ|ANDR[EÉ]S\s+IB[AÁ][ÑN]EZ|MONTERO|WARNES|CAMIRI|COTOCA)\b`), "Santa Cruz"},
	{regexp.MustCompile(`(?i)\b(COCHABAMBA|CERCADO|QUILLACOLLO|SACABA|TIQUIPAYA|PUNATA|VILLA\s+TUNARI)\b`), "Cochabamba"},
	{regexp.MustCompile(`(?i)\b(ORURO|CARACOLLO|HUANUNI|CHALLAPATA)\b`), "Oruro"},
	{regexp.MustCompile(`(?i)\b(POTOS[IÍ]|TUPIZA|VILLAZ[OÓ]N|UYUNI|LLALLAGUA)\b`), "Potosí"},
	{regexp.MustCompile(`(?i)\b(TARIJA|YACUIBA|BERMEJO|VILLAMONTES)\b`), "Tarija"},
	{regexp.MustCompile(`(?i)\b(CHUQUISACA|SUCRE|OROPEZA|MONTEAGUDO)\b`), "Sucre"},
	{regexp.MustCompile(`(?i)\b(BENI|TRINIDAD|RIBERALTA|GUAYARAMER[IÍ]N)\b`), "Trinidad"},
	{regexp.MustCompile(`(?i)\b(PANDO|COBIJA)\b`), "Cobija"},
}

var headerStopwords = regexp.MustCompile(`(?i)\b(ESTADO|PLURINACIONAL|BOLIVIA|SERVICIO|GENERAL|IDENTIFICACI[OÓ]N|PERSONAL|SEGIP|CEDULA|C[EÉ]DULA|IDENTIDAD|REPUBLICA|REP[UÚ]BLICA|REGISTRO|NACIONAL|PERSONAS|RENIEC|DOCUMENTO|CARNET|PASAPORTE|PASSPORT|SERIE|SECCION|SECCI[OÓ]N|FIRMA|TITULAR|DIRECTOR|HUELLA|PULGAR|DERECHO|FOTOGRAF[IÍ]A|VALIDEZ|EMISI[OÓ]N|EXPIRACI[OÓ]N|VENCIMIENTO)\b`)

// ParseAndMergeSides analiza el texto del frente y del reverso y fusiona los campos extraídos.
func ParseAndMergeSides(frontText, backText string) ParseResult {
	fRes := ParseDocumentText(frontText)
	if strings.TrimSpace(backText) == "" {
		return fRes
	}
	bRes := ParseDocumentText(backText)
	merged := MergeFields(fRes.Fields, bRes.Fields)
	detectedMRZ := fRes.DetectedMRZ || bRes.DetectedMRZ

	conf := "PARCIAL"
	count := countNonEmpty(merged)
	if detectedMRZ {
		conf = "ALTA (MRZ Verificado)"
	} else if count >= 3 {
		conf = "MEDIA-ALTA"
	}

	return ParseResult{
		Service:     "sisaloja-ocr",
		Engine:      "hybrid-mrz-segip",
		Confidence:  conf,
		DetectedMRZ: detectedMRZ,
		Fields:      merged,
	}
}

// ParseDocumentText analiza un bloque de texto OCR (con o sin zona MRZ ICAO 9303) y extrae los datos del huésped.
func ParseDocumentText(rawText string) ParseResult {
	text := strings.ReplaceAll(rawText, "\r", "\n")
	rawLines := strings.Split(text, "\n")
	var lines []string
	for _, l := range rawLines {
		t := strings.TrimSpace(l)
		if t != "" {
			lines = append(lines, t)
		}
	}

	mrzFields, mrzDetected := parseMRZFromLines(lines)
	visFields := parseVisualLabels(lines, text)

	docType := mrzFields.DocumentType
	if docType == "" {
		docType = visFields.DocumentType
	}
	if docType == "" {
		docType = "CI"
	}

	fields := DocumentFields{
		DocumentType:   docType,
		DocumentNumber: mergeDocNumber(mrzFields.DocumentNumber, visFields.DocumentNumber),
		FullName:       chooseBestName(mrzFields.FullName, visFields.FullName),
		Nationality:    firstNonEmpty(mrzFields.Nationality, visFields.Nationality),
		Gender:         firstNonEmpty(mrzFields.Gender, visFields.Gender),
		BirthDate:      firstNonEmpty(mrzFields.BirthDate, visFields.BirthDate),
		Occupation:     visFields.Occupation,
		OriginCity:     visFields.OriginCity,
	}

	conf := "PARCIAL"
	if mrzDetected {
		conf = "ALTA (MRZ Verificado)"
	} else if countNonEmpty(fields) >= 3 {
		conf = "MEDIA-ALTA"
	}

	return ParseResult{
		Service:     "sisaloja-ocr",
		Engine:      "hybrid-mrz-segip",
		Confidence:  conf,
		DetectedMRZ: mrzDetected,
		Fields:      fields,
	}
}

// MergeFields fusiona dos estructuras DocumentFields sin sobrescribir valores válidos con vacíos.
func MergeFields(current, incoming DocumentFields) DocumentFields {
	out := current
	if incoming.DocumentType != "" && (out.DocumentType == "" || out.DocumentType == "CI") {
		out.DocumentType = incoming.DocumentType
	}
	if incoming.DocumentNumber != "" {
		out.DocumentNumber = mergeDocNumber(out.DocumentNumber, incoming.DocumentNumber)
	}
	if incoming.FullName != "" {
		out.FullName = chooseBestName(out.FullName, incoming.FullName)
	}
	if out.Nationality == "" && incoming.Nationality != "" {
		out.Nationality = incoming.Nationality
	}
	if out.Gender == "" && incoming.Gender != "" {
		out.Gender = incoming.Gender
	}
	if out.BirthDate == "" && incoming.BirthDate != "" {
		out.BirthDate = incoming.BirthDate
	}
	if out.Occupation == "" && incoming.Occupation != "" {
		out.Occupation = incoming.Occupation
	}
	if out.OriginCity == "" && incoming.OriginCity != "" {
		out.OriginCity = incoming.OriginCity
	}
	return out
}

func countNonEmpty(f DocumentFields) int {
	n := 0
	for _, v := range []string{f.DocumentNumber, f.FullName, f.Nationality, f.Gender, f.BirthDate, f.Occupation, f.OriginCity} {
		if strings.TrimSpace(v) != "" {
			n++
		}
	}
	return n
}

func firstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}

func onlyDigits(s string) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func mergeDocNumber(a, b string) string {
	a = strings.TrimSpace(a)
	b = strings.TrimSpace(b)
	if a != "" && b != "" {
		if onlyDigits(a) == onlyDigits(b) {
			if len(b) > len(a) {
				return b
			}
			return a
		}
		if len(b) > len(a) {
			return b
		}
		return a
	}
	if b != "" {
		return b
	}
	return a
}

func chooseBestName(a, b string) string {
	a = strings.TrimSpace(a)
	b = strings.TrimSpace(b)
	aWords := len(strings.Fields(a))
	bWords := len(strings.Fields(b))
	if aWords >= 2 && aWords >= bWords {
		return a
	}
	if bWords >= 2 {
		return b
	}
	return firstNonEmpty(a, b)
}

var (
	reMRZPrefix = regexp.MustCompile(`^(ID|I<|P<|A<|C<)[A-Z]{3}`)
	reTD1Line1  = regexp.MustCompile(`^(I[D<A-Z0-9]|A[C<A-Z0-9]|C[I<A-Z0-9])([A-Z]{3})([A-Z0-9<]{7,25})`)
	reTD3Line1  = regexp.MustCompile(`^P[<A-Z]([A-Z]{3})([A-Z<]{10,40})`)
	reTD1Line2  = regexp.MustCompile(`([0-9OQDILSZB]{6})[0-9OQDILSZB]([MF<])([0-9OQDILSZB]{6})[0-9OQDILSZB]([A-Z]{3})`)
	reHas4Digit = regexp.MustCompile(`\d{4}`)
)

func parseMRZFromLines(lines []string) (DocumentFields, bool) {
	var res DocumentFields
	detected := false

	var mrzLines []string
	replacer := strings.NewReplacer(" ", "", "\t", "", "«", "<", "»", "<", "‹", "<", "›", "<")
	for _, l := range lines {
		norm := replacer.Replace(strings.ToUpper(l))
		if (strings.Contains(norm, "<<") && len(norm) >= 18) || reMRZPrefix.MatchString(norm) {
			mrzLines = append(mrzLines, norm)
		}
	}

	for _, l := range mrzLines {
		if m := reTD1Line1.FindStringSubmatch(l); m != nil {
			detected = true
			countryCode := fixMRZLetters(m[2])
			if nat, ok := countryCodeMap[countryCode]; ok {
				res.Nationality = nat
				if countryCode == "BOL" {
					res.DocumentType = "CI"
				} else {
					res.DocumentType = "DNI"
				}
			}
			rawDoc := strings.Split(m[3], "<")[0]
			cleanDoc := cleanAlnum(fixMRZDigits(rawDoc))
			if len(cleanDoc) >= 5 {
				res.DocumentNumber = cleanDoc
			}
		}

		if m := reTD3Line1.FindStringSubmatch(l); m != nil {
			detected = true
			res.DocumentType = "PASAPORTE"
			countryCode := fixMRZLetters(m[1])
			if nat, ok := countryCodeMap[countryCode]; ok {
				res.Nationality = nat
			}
			if name := parseMRZNameBlock(m[2]); name != "" {
				res.FullName = name
			}
		}

		if m := reTD1Line2.FindStringSubmatch(l); m != nil {
			detected = true
			if bDate := parseMRZDate(m[1]); bDate != "" {
				res.BirthDate = bDate
			}
			if m[2] == "M" || m[2] == "F" {
				res.Gender = m[2]
			}
			natCode := fixMRZLetters(m[4])
			if nat, ok := countryCodeMap[natCode]; ok && res.Nationality == "" {
				res.Nationality = nat
			}
		}

		prefix10 := l
		if len(prefix10) > 10 {
			prefix10 = prefix10[:10]
		}
		if strings.Contains(l, "<<") &&
			!strings.HasPrefix(l, "ID") &&
			!strings.HasPrefix(l, "I<") &&
			!strings.HasPrefix(l, "P<") &&
			!reHas4Digit.MatchString(prefix10) {
			if name := parseMRZNameBlock(l); len(name) >= 5 {
				detected = true
				res.FullName = name
			}
		}
	}

	return res, detected
}

func fixMRZDigits(s string) string {
	r := strings.NewReplacer(
		"O", "0", "Q", "0", "D", "0",
		"I", "1", "L", "1", "|", "1",
		"Z", "2", "S", "5", "B", "8",
	)
	return r.Replace(strings.ToUpper(s))
}

func fixMRZLetters(s string) string {
	r := strings.NewReplacer("0", "O", "1", "I", "5", "S", "8", "B")
	return r.Replace(strings.ToUpper(s))
}

func cleanAlnum(s string) string {
	var b strings.Builder
	for _, r := range s {
		if (r >= '0' && r <= '9') || (r >= 'A' && r <= 'Z') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func parseMRZDate(yymmdd string) string {
	clean := fixMRZDigits(yymmdd)
	if len(clean) != 6 {
		return ""
	}
	yy, err1 := strconv.Atoi(clean[0:2])
	mm, err2 := strconv.Atoi(clean[2:4])
	dd, err3 := strconv.Atoi(clean[4:6])
	if err1 != nil || err2 != nil || err3 != nil || mm < 1 || mm > 12 || dd < 1 || dd > 31 {
		return ""
	}
	curYY := time.Now().Year() % 100
	year := 2000 + yy
	if yy > curYY {
		year = 1900 + yy
	}
	return fmt.Sprintf("%04d-%02d-%02d", year, mm, dd)
}

func parseMRZNameBlock(mrzLine string) string {
	fixed := fixMRZLetters(mrzLine)
	var b strings.Builder
	for _, r := range fixed {
		if (r >= 'A' && r <= 'Z') || r == '<' {
			b.WriteRune(r)
		}
	}
	parts := strings.Split(b.String(), "<<")
	if len(parts) < 2 {
		return ""
	}
	surnames := strings.Join(strings.Fields(strings.ReplaceAll(parts[0], "<", " ")), " ")
	given := strings.Join(strings.Fields(strings.ReplaceAll(strings.Join(parts[1:], " "), "<", " ")), " ")
	if surnames == "" && given == "" {
		return ""
	}
	full := surnames
	if given != "" {
		full = strings.TrimSpace(given + " " + surnames)
	}
	return toTitleCase(full)
}

var (
	rePassport = regexp.MustCompile(`(?i)PASAPORTE|PASSPORT`)
	reDNI      = regexp.MustCompile(`(?i)\b(DNI|RENIEC|DOCUMENTO\s+NACIONAL)\b`)
	reBolivia  = regexp.MustCompile(`(?i)\b(BOLIVIA|PLURINACIONAL|SEGIP|BOLIVIANA|BOLIVIANO)\b`)
	rePeru     = regexp.MustCompile(`(?i)\b(PER[UÚ]|RENIEC|PERUANA|PERUANO)\b`)
	reArg      = regexp.MustCompile(`(?i)\b(ARGENTINA|ARGENTINO)\b`)
	reChile    = regexp.MustCompile(`(?i)\b(CHILE|CHILENA|CHILENO)\b`)
	reBrasil   = regexp.MustCompile(`(?i)\b(BRASIL|BRASILE[ÑN]A)\b`)

	reDoc1 = regexp.MustCompile(`(?i)(?:N[°ºo.]|NRO\.?|C\.?I\.?|CEDULA|C[EÉ]DULA|DOCUMENTO|SERIE)\s*[:.#-]?\s*([0-9]{5,10}(?:\s*[-]\s*[0-9A-Z]{1,3})?(?:\s+(?:LP|SC|CB|CBA|OR|PT|TJ|CH|BE|BN|PD))?)`)
	reDoc2 = regexp.MustCompile(`(?i)\b([0-9]{6,9})\s+(LP|SC|CB|CBA|OR|PT|TJ|CH|BE|PD)\b`)
	reDoc3 = regexp.MustCompile(`\b([0-9]{7,9})\b`)

	reTextDate = regexp.MustCompile(`(?i)\b(\d{1,2})\s*(?:DE\s+|[-/.\s]+)(ENERO|FEBRERO|MARZO|ABRIL|MAYO|JUNIO|JULIO|AGOSTO|SEPTIEMBRE|SETIEMBRE|OCTUBRE|NOVIEMBRE|DICIEMBRE|ENE|FEB|MAR|ABR|MAY|JUN|JUL|AGO|SEP|SET|OCT|NOV|DIC)\s*(?:DE\s+|DEL\s+|[-/.\s]+)((?:19|20)\d{2})\b`)
	reNacDate  = regexp.MustCompile(`(?i)NACIMIENTO[\s\S]{0,45}?\b(\d{1,2})[/.-](\d{1,2})[/.-]((19|20)\d{2})\b`)
	reAnyDate  = regexp.MustCompile(`\b(\d{1,2})[/.-](\d{1,2})[/.-]((19\d{2}|20[012]\d))\b`)

	reNombres   = regexp.MustCompile(`(?i)\bNOMBRES?\s*[:.]?\s*(.*)$`)
	reApellido1 = regexp.MustCompile(`(?i)\b(?:PRIMER\s+APELLIDO|APELLIDO\s+PATERNO)\s*[:.]?\s*(.*)$`)
	reApellido2 = regexp.MustCompile(`(?i)\b(?:SEGUNDO\s+APELLIDO|APELLIDO\s+MATERNO)\s*[:.]?\s*(.*)$`)
	reApellidos = regexp.MustCompile(`(?i)^\s*APELLIDOS?\s*[:.]?\s*(.*)$`)
	reClassicA  = regexp.MustCompile(`(?i)^\s*A\s*[:.]\s*([A-ZÁÉÍÓÚÑ\s]{5,45})$`)
	reOcupacion = regexp.MustCompile(`(?i)\b(?:OCUPACI[OÓ]N|PROFESI[OÓ]N)(?:\s*[/]\s*(?:PROFESI[OÓ]N|OCUPACI[OÓ]N))?\s*[:.]?\s*(.*)$`)
	reOrigen    = regexp.MustCompile(`(?i)\b(?:LUGAR\s+DE\s+NACIMIENTO|NATURAL\s+DE|NACIDO\s+EN|DEPARTAMENTO)\s*[:.]?\s*(.*)$`)
	reSexo      = regexp.MustCompile(`(?i)\b(?:SEXO|G[EÉ]NERO)\s*[:.]?\s*(MASCULINO|FEMENINO|HOMBRE|MUJER|\bM\b|\bF\b)`)
)

func parseVisualLabels(lines []string, fullText string) DocumentFields {
	out := DocumentFields{DocumentType: "CI"}

	if rePassport.MatchString(fullText) {
		out.DocumentType = "PASAPORTE"
	} else if reDNI.MatchString(fullText) {
		out.DocumentType = "DNI"
	}

	if reBolivia.MatchString(fullText) {
		out.Nationality = "Boliviana"
		if out.DocumentType != "PASAPORTE" {
			out.DocumentType = "CI"
		}
	} else if rePeru.MatchString(fullText) {
		out.Nationality = "Peruana"
	} else if reArg.MatchString(fullText) {
		out.Nationality = "Argentina"
	} else if reChile.MatchString(fullText) {
		out.Nationality = "Chilena"
	} else if reBrasil.MatchString(fullText) {
		out.Nationality = "Brasileña"
	}

	if m := reDoc1.FindStringSubmatch(fullText); m != nil && len(onlyDigits(m[1])) >= 6 {
		out.DocumentNumber = strings.ToUpper(strings.Join(strings.Fields(m[1]), " "))
	} else if m := reDoc2.FindStringSubmatch(fullText); m != nil {
		out.DocumentNumber = strings.ToUpper(m[1] + " " + m[2])
	} else if m := reDoc3.FindStringSubmatch(fullText); m != nil {
		out.DocumentNumber = m[1]
	}

	if m := reTextDate.FindStringSubmatch(fullText); m != nil {
		dd, _ := strconv.Atoi(m[1])
		mm := monthsES[strings.ToUpper(m[2])]
		yyyy := m[3]
		if mm != "" {
			out.BirthDate = fmt.Sprintf("%s-%s-%02d", yyyy, mm, dd)
		}
	}
	if out.BirthDate == "" {
		dm := reNacDate.FindStringSubmatch(fullText)
		if dm == nil {
			dm = reAnyDate.FindStringSubmatch(fullText)
		}
		if dm != nil {
			dd, _ := strconv.Atoi(dm[1])
			mm, _ := strconv.Atoi(dm[2])
			yyyy := dm[3]
			if mm >= 1 && mm <= 12 && dd >= 1 && dd <= 31 {
				out.BirthDate = fmt.Sprintf("%s-%02d-%02d", yyyy, mm, dd)
			}
		}
	}

	var givenNames, surnames, firstSurname, secondSurname, singleLineName string

	for i, line := range lines {
		nextLine := ""
		if i+1 < len(lines) {
			nextLine = lines[i+1]
		}

		if m := reNombres.FindStringSubmatch(line); m != nil && givenNames == "" {
			inline := cleanPersonName(m[1])
			if len(inline) >= 2 {
				givenNames = inline
			} else {
				givenNames = cleanPersonName(nextLine)
			}
		}
		if m := reApellido1.FindStringSubmatch(line); m != nil && firstSurname == "" {
			inline := cleanPersonName(m[1])
			if len(inline) >= 2 {
				firstSurname = inline
			} else {
				firstSurname = cleanPersonName(nextLine)
			}
		}
		if m := reApellido2.FindStringSubmatch(line); m != nil && secondSurname == "" {
			inline := cleanPersonName(m[1])
			if len(inline) >= 2 {
				secondSurname = inline
			} else {
				secondSurname = cleanPersonName(nextLine)
			}
		}
		if m := reApellidos.FindStringSubmatch(line); m != nil && surnames == "" {
			inline := cleanPersonName(m[1])
			if len(inline) >= 2 {
				surnames = inline
			} else {
				surnames = cleanPersonName(nextLine)
			}
		}
		if m := reClassicA.FindStringSubmatch(line); m != nil && singleLineName == "" {
			singleLineName = cleanPersonName(m[1])
		}
		if m := reOcupacion.FindStringSubmatch(line); m != nil && out.Occupation == "" {
			inline := cleanLabelValue(m[1])
			if len(inline) >= 3 {
				out.Occupation = inline
			} else {
				out.Occupation = cleanLabelValue(nextLine)
			}
		}
		if m := reOrigen.FindStringSubmatch(line); m != nil && out.OriginCity == "" {
			candidate := m[1] + " " + nextLine
			city := detectBoliviaCity(candidate)
			if city != "" {
				out.OriginCity = city
			} else {
				out.OriginCity = cleanLabelValue(m[1])
			}
		}
		if m := reSexo.FindStringSubmatch(line); m != nil && out.Gender == "" {
			v := strings.ToUpper(m[1])
			if strings.HasPrefix(v, "F") || strings.HasPrefix(v, "MUJ") {
				out.Gender = "F"
			} else {
				out.Gender = "M"
			}
		}
	}

	combinedSurnames := strings.TrimSpace(firstSurname + " " + secondSurname)
	if combinedSurnames == "" {
		combinedSurnames = surnames
	}

	if givenNames != "" && combinedSurnames != "" {
		out.FullName = toTitleCase(givenNames + " " + combinedSurnames)
	} else if singleLineName != "" {
		out.FullName = toTitleCase(singleLineName)
	} else if givenNames != "" || combinedSurnames != "" {
		out.FullName = toTitleCase(strings.TrimSpace(givenNames + " " + combinedSurnames))
	}

	if out.OriginCity == "" {
		out.OriginCity = detectBoliviaCity(fullText)
	}
	if out.OriginCity != "" && out.Nationality == "" {
		out.Nationality = "Boliviana"
	}

	return out
}

func cleanPersonName(raw string) string {
	var b strings.Builder
	for _, r := range raw {
		if unicode.IsLetter(r) || unicode.IsSpace(r) {
			b.WriteRune(r)
		} else {
			b.WriteRune(' ')
		}
	}
	cleaned := strings.Join(strings.Fields(b.String()), " ")
	if len(cleaned) < 2 || headerStopwords.MatchString(cleaned) {
		return ""
	}
	return cleaned
}

func cleanLabelValue(raw string) string {
	var b strings.Builder
	for _, r := range raw {
		if unicode.IsLetter(r) || unicode.IsSpace(r) || r == '.' || r == '-' {
			b.WriteRune(r)
		} else {
			b.WriteRune(' ')
		}
	}
	cleaned := strings.Join(strings.Fields(b.String()), " ")
	if len(cleaned) < 3 || headerStopwords.MatchString(cleaned) {
		return ""
	}
	if len(cleaned) > 40 {
		cleaned = cleaned[:40]
	}
	return toTitleCase(cleaned)
}

func detectBoliviaCity(text string) string {
	for _, rule := range boliviaCities {
		if rule.re.MatchString(text) {
			return rule.city
		}
	}
	return ""
}

func toTitleCase(str string) string {
	words := strings.Fields(strings.ToLower(str))
	for i, w := range words {
		runes := []rune(w)
		if len(runes) > 0 {
			runes[0] = unicode.ToUpper(runes[0])
			words[i] = string(runes)
		}
	}
	return strings.Join(words, " ")
}
