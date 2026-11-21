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
	FirstName      string `json:"first_name,omitempty"`
	LastName       string `json:"last_name,omitempty"`
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

// Palabras prohibidas dentro de un nombre o apellido (membretes, etiquetas laterales, estados civiles, extensiones)
var forbiddenNameTokens = map[string]bool{
	"ESTADO": true, "PLURINACIONAL": true, "BOLIVIA": true, "SERVICIO": true, "GENERAL": true,
	"IDENTIFICACION": true, "IDENTIFICACIÓN": true, "PERSONAL": true, "SEGIP": true,
	"CEDULA": true, "CÉDULA": true, "IDENTIDAD": true, "REPUBLICA": true, "REPÚBLICA": true,
	"REGISTRO": true, "NACIONAL": true, "PERSONAS": true, "RENIEC": true, "DOCUMENTO": true,
	"CARNET": true, "PASAPORTE": true, "PASSPORT": true, "SERIE": true, "SECCION": true, "SECCIÓN": true,
	"FIRMA": true, "TITULAR": true, "DIRECTOR": true, "HUELLA": true, "PULGAR": true, "DERECHO": true,
	"FOTOGRAFIA": true, "FOTOGRAFÍA": true, "VALIDEZ": true, "EMISION": true, "EMISIÓN": true,
	"EXPIRACION": true, "EXPIRACIÓN": true, "VENCIMIENTO": true, "FECHA": true, "NACIMIENTO": true,
	"LUGAR": true, "SEXO": true, "GENERO": true, "GÉNERO": true, "CIVIL": true, "SOLTERO": true,
	"SOLTERA": true, "CASADO": true, "CASADA": true, "VIUDO": true, "VIUDA": true, "DIVORCIADO": true,
	"DIVORCIADA": true, "OCUPACION": true, "OCUPACIÓN": true, "PROFESION": true, "PROFESIÓN": true,
	"DOMICILIO": true, "DIRECCION": true, "DIRECCIÓN": true, "NOMBRE": true, "NOMBRES": true,
	"APELLIDO": true, "APELLIDOS": true, "PRIMER": true, "SEGUNDO": true, "PATERNO": true, "MATERNO": true,
	"NRO": true, "NUM": true, "NUMERO": true, "NÚMERO": true, "BOLIVIANA": true, "BOLIVIANO": true,
	"LP": true, "SC": true, "CB": true, "CBA": true, "OR": true, "PT": true, "TJ": true, "CH": true, "BE": true, "PD": true,
}

// Cortadores de línea: si en la misma fila horizontal del nombre aparece una etiqueta de otra columna, se corta ahí.
var reInlineLabelCut = regexp.MustCompile(`(?i)\b(FECHA|FECH|NACIMIENTO|NACIM|LUGAR|SEXO|G[EÉ]NERO|ESTADO\s+CIVIL|EST\.\s*CIVIL|OCUPACI[OÓ]N|PROFESI[OÓ]N|DOMICILIO|SERIE|SECCI[OÓ]N|EXPIRACI[OÓ]N|VENCIMIENTO|EMISI[OÓ]N|FIRMA|HUELLA|N[°ºo.]|C\.?I\.?|NRO)\b.*$`)

// ParseAndMergeSides analiza el texto del frente y del reverso y fusiona los campos extraídos.
func ParseAndMergeSides(frontText, backText string) ParseResult {
	fRes := ParseDocumentText(frontText)
	if strings.TrimSpace(backText) == "" {
		return fRes
	}
	bRes := ParseDocumentText(backText)
	merged := MergeFieldsWithMRZPriority(fRes.Fields, fRes.DetectedMRZ, bRes.Fields, bRes.DetectedMRZ)
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

	// Si el MRZ detectó el nombre, tiene prioridad absoluta porque está libre de ruido visual del frente
	finalFirst := visFields.FirstName
	finalLast := visFields.LastName
	finalFull := visFields.FullName
	if mrzDetected && mrzFields.FullName != "" {
		finalFull = mrzFields.FullName
		if mrzFields.FirstName != "" {
			finalFirst = mrzFields.FirstName
		}
		if mrzFields.LastName != "" {
			finalLast = mrzFields.LastName
		}
	}

	fields := DocumentFields{
		DocumentType:   docType,
		DocumentNumber: mergeDocNumber(mrzFields.DocumentNumber, visFields.DocumentNumber),
		FullName:       finalFull,
		FirstName:      finalFirst,
		LastName:       finalLast,
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

// MergeFieldsWithMRZPriority fusiona Frente y Reverso, combinando Nombres + Apellidos si fueron leídos por separado y priorizando MRZ.
func MergeFieldsWithMRZPriority(current DocumentFields, curMRZ bool, incoming DocumentFields, incMRZ bool) DocumentFields {
	out := current
	if incoming.DocumentType != "" && (out.DocumentType == "" || out.DocumentType == "CI") {
		out.DocumentType = incoming.DocumentType
	}
	if incoming.DocumentNumber != "" {
		out.DocumentNumber = mergeDocNumber(out.DocumentNumber, incoming.DocumentNumber)
	}

	// Combinar FirstName y LastName de ambas lecturas
	if incMRZ && incoming.FullName != "" {
		out.FullName = incoming.FullName
		out.FirstName = firstNonEmpty(incoming.FirstName, out.FirstName)
		out.LastName = firstNonEmpty(incoming.LastName, out.LastName)
	} else if !curMRZ {
		if out.FirstName == "" && incoming.FirstName != "" {
			out.FirstName = incoming.FirstName
		}
		if out.LastName == "" && incoming.LastName != "" {
			out.LastName = incoming.LastName
		}
		if out.FirstName != "" && out.LastName != "" {
			out.FullName = strings.TrimSpace(out.FirstName + " " + out.LastName)
		} else if incoming.FullName != "" && out.FullName == "" {
			out.FullName = incoming.FullName
		} else if incoming.FullName != "" && len(strings.Fields(incoming.FullName)) > len(strings.Fields(out.FullName)) && len(strings.Fields(incoming.FullName)) <= 4 {
			out.FullName = incoming.FullName
		}
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

// MergeFields mantiene compatibilidad con llamadas directas sin banderas MRZ.
func MergeFields(current, incoming DocumentFields) DocumentFields {
	return MergeFieldsWithMRZPriority(current, false, incoming, false)
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
			first, last, full := parseMRZNameParts(m[2])
			if full != "" {
				res.FirstName = first
				res.LastName = last
				res.FullName = full
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
			first, last, full := parseMRZNameParts(l)
			if len(full) >= 5 {
				detected = true
				res.FirstName = first
				res.LastName = last
				res.FullName = full
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

func parseMRZNameParts(mrzLine string) (firstName, lastName, fullName string) {
	fixed := fixMRZLetters(mrzLine)
	var b strings.Builder
	for _, r := range fixed {
		if (r >= 'A' && r <= 'Z') || r == '<' {
			b.WriteRune(r)
		}
	}
	parts := strings.Split(b.String(), "<<")
	if len(parts) < 2 {
		return "", "", ""
	}
	surnames := toTitleCase(strings.Join(strings.Fields(strings.ReplaceAll(parts[0], "<", " ")), " "))
	given := toTitleCase(strings.Join(strings.Fields(strings.ReplaceAll(strings.Join(parts[1:], " "), "<", " ")), " "))
	if surnames == "" && given == "" {
		return "", "", ""
	}
	full := surnames
	if given != "" {
		full = strings.TrimSpace(given + " " + surnames)
	}
	return given, surnames, full
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

	// Etiquetas tolerantes a errores comunes de OCR (0 por O, 1 por I) y sin ancla ^ rígida
	reBothApellidoHeaders = regexp.MustCompile(`(?i)PR[I1]MER\s+APELL[I1]D[O0].*SEGUND[O0]\s+APELL[I1]D[O0]|APELL[I1]D[O0]\s+PATERN[O0].*APELL[I1]D[O0]\s+MATERN[O0]`)
	reNombres             = regexp.MustCompile(`(?i)\bN[O0]MBRES?\s*[:.;-]?\s*(.*)$`)
	reApellido1           = regexp.MustCompile(`(?i)\b(?:PR[I1]MER\s+APELL[I1]D[O0]|APELL[I1]D[O0]\s+PATERN[O0])\s*[:.;-]?\s*(.*)$`)
	reApellido2           = regexp.MustCompile(`(?i)\b(?:SEGUND[O0]\s+APELL[I1]D[O0]|APELL[I1]D[O0]\s+MATERN[O0])\s*[:.;-]?\s*(.*)$`)
	reApellidos           = regexp.MustCompile(`(?i)\bAPELL[I1]D[O0]S?\s*[:.;-]?\s*(.*)$`)
	reClassicA            = regexp.MustCompile(`(?i)\bA\s*[:.]\s*([A-ZÁÉÍÓÚÑ\s]{5,45})$`)
	reOcupacion           = regexp.MustCompile(`(?i)\b(?:[O0]CUPAC[I1][OÓ0]N|PR[O0]FES[I1][OÓ0]N)(?:\s*[/]\s*(?:PR[O0]FES[I1][OÓ0]N|[O0]CUPAC[I1][OÓ0]N))?\s*[:.;-]?\s*(.*)$`)
	reOrigen              = regexp.MustCompile(`(?i)\b(?:LUGAR\s+DE\s+NAC[I1]M[I1]ENT[O0]|NATURAL\s+DE|NAC[I1]D[O0]\s+EN|DEPARTAMENT[O0])\s*[:.;-]?\s*(.*)$`)
	reSexo                = regexp.MustCompile(`(?i)\b(?:SEX[O0]|G[EÉ]NER[O0])\s*[:.;-]?\s*(MASCULINO|FEMENINO|HOMBRE|MUJER|\bM\b|\bF\b)`)
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
	givenLineIdx := -1
	surnameLineIdx := -1

	for i, line := range lines {
		nextLine := ""
		if i+1 < len(lines) {
			nextLine = lines[i+1]
		}

		// Caso 1: La línea tiene los dos encabezados en columnas: "PRIMER APELLIDO   SEGUNDO APELLIDO"
		if reBothApellidoHeaders.MatchString(line) && surnames == "" {
			surnames = sanitizePersonTokens(nextLine, 3)
			if surnames != "" {
				surnameLineIdx = i + 1
			}
			continue
		}

		// Caso 2: NOMBRES
		if m := reNombres.FindStringSubmatch(line); m != nil && givenNames == "" {
			inline := sanitizePersonTokens(m[1], 3)
			if len(inline) >= 2 {
				givenNames = inline
				givenLineIdx = i
			} else {
				givenNames = sanitizePersonTokens(nextLine, 3)
				if givenNames != "" {
					givenLineIdx = i + 1
				}
			}
		}

		// Caso 3: PRIMER APELLIDO
		if m := reApellido1.FindStringSubmatch(line); m != nil && firstSurname == "" {
			inline := sanitizePersonTokens(m[1], 2)
			if len(inline) >= 2 {
				firstSurname = inline
				surnameLineIdx = i
			} else {
				firstSurname = sanitizePersonTokens(nextLine, 2)
				if firstSurname != "" {
					surnameLineIdx = i + 1
				}
			}
		}

		// Caso 4: SEGUNDO APELLIDO
		if m := reApellido2.FindStringSubmatch(line); m != nil && secondSurname == "" {
			inline := sanitizePersonTokens(m[1], 2)
			if len(inline) >= 2 {
				secondSurname = inline
			} else {
				secondSurname = sanitizePersonTokens(nextLine, 2)
			}
		}

		// Caso 5: APELLIDOS (sin ser PRIMER/SEGUNDO)
		if !reApellido1.MatchString(line) && !reApellido2.MatchString(line) {
			if m := reApellidos.FindStringSubmatch(line); m != nil && surnames == "" {
				inline := sanitizePersonTokens(m[1], 3)
				if len(inline) >= 2 {
					surnames = inline
					surnameLineIdx = i
				} else {
					surnames = sanitizePersonTokens(nextLine, 3)
					if surnames != "" {
						surnameLineIdx = i + 1
					}
				}
			}
		}

		// Caso 6: Carnet clásico "A: ALEJANDRO TORREZ SILVA"
		if m := reClassicA.FindStringSubmatch(line); m != nil && singleLineName == "" {
			singleLineName = sanitizePersonTokens(m[1], 4)
		}

		// OCUPACIÓN / PROFESIÓN
		if m := reOcupacion.FindStringSubmatch(line); m != nil && out.Occupation == "" {
			inline := cleanLabelValue(m[1])
			if len(inline) >= 3 {
				out.Occupation = inline
			} else {
				out.Occupation = cleanLabelValue(nextLine)
			}
		}

		// LUGAR DE NACIMIENTO / PROCEDENCIA
		if m := reOrigen.FindStringSubmatch(line); m != nil && out.OriginCity == "" {
			candidate := m[1] + " " + nextLine
			city := detectBoliviaCity(candidate)
			if city != "" {
				out.OriginCity = city
			} else {
				out.OriginCity = cleanLabelValue(m[1])
			}
		}

		// SEXO / GÉNERO
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

	// Respaldo posicional: si el OCR leyó "NOMBRES: MIJAEL" pero la etiqueta "APELLIDOS" salió ilegible,
	// revisar la línea inmediatamente siguiente o anterior al nombre por si contiene los apellidos limpios.
	if givenNames != "" && combinedSurnames == "" && givenLineIdx >= 0 {
		for _, candidateIdx := range []int{givenLineIdx + 1, givenLineIdx + 2, givenLineIdx - 1} {
			if candidateIdx >= 0 && candidateIdx < len(lines) && candidateIdx != givenLineIdx {
				rawCand := lines[candidateIdx]
				if !strings.Contains(rawCand, ":") && !reAnyDate.MatchString(rawCand) && !reDoc3.MatchString(rawCand) {
					candTokens := sanitizePersonTokens(rawCand, 3)
					if candTokens != "" && !strings.EqualFold(candTokens, givenNames) && len(strings.Fields(candTokens)) >= 1 && len(strings.Fields(candTokens)) <= 3 {
						combinedSurnames = candTokens
						break
					}
				}
			}
		}
	}

	// Y viceversa: si detectó los apellidos pero la etiqueta "NOMBRES" salió borrosa
	if combinedSurnames != "" && givenNames == "" && surnameLineIdx > 0 {
		for _, candidateIdx := range []int{surnameLineIdx - 1, surnameLineIdx + 1} {
			if candidateIdx >= 0 && candidateIdx < len(lines) && candidateIdx != surnameLineIdx {
				rawCand := lines[candidateIdx]
				if !strings.Contains(rawCand, ":") && !reAnyDate.MatchString(rawCand) && !reDoc3.MatchString(rawCand) {
					candTokens := sanitizePersonTokens(rawCand, 2)
					if candTokens != "" && !strings.EqualFold(candTokens, combinedSurnames) {
						givenNames = candTokens
						break
					}
				}
			}
		}
	}

	out.FirstName = toTitleCase(givenNames)
	out.LastName = toTitleCase(combinedSurnames)

	if out.FirstName != "" && out.LastName != "" {
		out.FullName = strings.TrimSpace(out.FirstName + " " + out.LastName)
	} else if singleLineName != "" {
		out.FullName = toTitleCase(singleLineName)
	} else if out.FirstName != "" || out.LastName != "" {
		out.FullName = strings.TrimSpace(out.FirstName + " " + out.LastName)
	}

	if out.OriginCity == "" {
		out.OriginCity = detectBoliviaCity(fullText)
	}
	if out.OriginCity != "" && out.Nationality == "" {
		out.Nationality = "Boliviana"
	}

	return out
}

// sanitizePersonTokens limpia una línea de nombre o apellido:
// 1. Corta la línea si aparece una etiqueta de otra columna (FECHA, NACIMIENTO, SEXO, ESTADO CIVIL, etc.).
// 2. Descarta tokens de 1 sola letra (excepto "Y" / "DE" / "DEL" / "LA"), palabras sin vocales (ruido del escudo/fondo),
//    siglas de departamentos (LP, SC, CB) y palabras de membrete.
// 3. Limita la cantidad máxima de palabras (maxWords) para no arrastrar basura del final de la línea.
func sanitizePersonTokens(raw string, maxWords int) string {
	if strings.TrimSpace(raw) == "" {
		return ""
	}
	if maxWords <= 0 {
		maxWords = 3
	}

	// 1. Cortar en cuanto aparezca una etiqueta intrusa en la misma línea horizontal
	cut := reInlineLabelCut.ReplaceAllString(raw, "")

	// 2. Dejar solo letras y espacios
	var b strings.Builder
	for _, r := range cut {
		if unicode.IsLetter(r) || unicode.IsSpace(r) {
			b.WriteRune(r)
		} else {
			b.WriteRune(' ')
		}
	}

	words := strings.Fields(b.String())
	var valid []string
	connectors := map[string]bool{"DE": true, "DEL": true, "LA": true, "LAS": true, "LOS": true}

	for _, w := range words {
		up := strings.ToUpper(w)
		if forbiddenNameTokens[up] {
			continue
		}
		if connectors[up] {
			if len(valid) > 0 {
				valid = append(valid, up)
			}
			continue
		}
		if len([]rune(up)) < 2 || len([]rune(up)) > 18 {
			continue
		}
		if !hasSpanishVowel(up) || hasTripleRepeat(up) {
			continue
		}
		valid = append(valid, up)
		if len(valid) >= maxWords {
			break
		}
	}

	// Si el último token quedó siendo un conector suelto ("DE"), quitarlo
	for len(valid) > 0 && connectors[valid[len(valid)-1]] {
		valid = valid[:len(valid)-1]
	}

	return strings.Join(valid, " ")
}

func hasSpanishVowel(s string) bool {
	for _, r := range s {
		switch unicode.ToUpper(r) {
		case 'A', 'E', 'I', 'O', 'U', 'Á', 'É', 'Í', 'Ó', 'Ú', 'Ü':
			return true
		}
	}
	return false
}

func hasTripleRepeat(s string) bool {
	runes := []rune(s)
	for i := 2; i < len(runes); i++ {
		if runes[i] == runes[i-1] && runes[i] == runes[i-2] {
			return true
		}
	}
	return false
}

func cleanLabelValue(raw string) string {
	cut := reInlineLabelCut.ReplaceAllString(raw, "")
	var b strings.Builder
	for _, r := range cut {
		if unicode.IsLetter(r) || unicode.IsSpace(r) || r == '.' || r == '-' {
			b.WriteRune(r)
		} else {
			b.WriteRune(' ')
		}
	}
	words := strings.Fields(b.String())
	var valid []string
	for _, w := range words {
		up := strings.ToUpper(w)
		if forbiddenNameTokens[up] || len([]rune(up)) < 2 || !hasSpanishVowel(up) {
			continue
		}
		valid = append(valid, w)
		if len(valid) >= 4 {
			break
		}
	}
	cleaned := strings.Join(valid, " ")
	if len(cleaned) < 3 {
		return ""
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
