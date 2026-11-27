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
	Service          string         `json:"service"`
	Engine           string         `json:"engine"`
	Confidence       string         `json:"confidence"`
	DetectedMRZ      bool           `json:"detected_mrz"`
	IsBolivia        bool           `json:"is_bolivia,omitempty"`
	BoliviaSubtype   string         `json:"bolivia_subtype,omitempty"`
	ExtractionMethod string         `json:"extraction_method,omitempty"`
	RawText          string         `json:"raw_text,omitempty"`
	Fields           DocumentFields `json:"fields"`
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

// Priorizar primero nombres explícitos de departamentos/ciudades antes de provincias ambiguas como CERCADO.
var boliviaCities = []cityRule{
	{regexp.MustCompile(`(?i)\b(LA\s+PAZ|EL\s+ALTO|VIACHA|ACHACACHI|COPACABANA|CARANAVI)\b`), "La Paz"},
	{regexp.MustCompile(`(?i)\b(SANTA\s+CRUZ|MONTERO|WARNES|CAMIRI|COTOCA)\b`), "Santa Cruz"},
	{regexp.MustCompile(`(?i)\b(COCHABAMBA|QUILLACOLLO|SACABA|TIQUIPAYA|PUNATA|VILLA\s+TUNARI)\b`), "Cochabamba"},
	{regexp.MustCompile(`(?i)\b(ORURO|CARACOLLO|HUANUNI|CHALLAPATA)\b`), "Oruro"},
	{regexp.MustCompile(`(?i)\b(POTOS[IÍ]|TUPIZA|VILLAZ[OÓ]N|UYUNI|LLALLAGUA)\b`), "Potosí"},
	{regexp.MustCompile(`(?i)\b(TARIJA|YACUIBA|BERMEJO|VILLAMONTES)\b`), "Tarija"},
	{regexp.MustCompile(`(?i)\b(CHUQUISACA|SUCRE|MONTEAGUDO)\b`), "Sucre"},
	{regexp.MustCompile(`(?i)\b(BENI|TRINIDAD|RIBERALTA|GUAYARAMER[IÍ]N)\b`), "Trinidad"},
	{regexp.MustCompile(`(?i)\b(PANDO|COBIJA)\b`), "Cobija"},
	{regexp.MustCompile(`(?i)\bMURILLO\b`), "La Paz"},
	{regexp.MustCompile(`(?i)\bANDR[EÉ]S\s+IB[AÁ][ÑN]EZ\b`), "Santa Cruz"},
	{regexp.MustCompile(`(?i)\bOROPEZA\b`), "Sucre"},
	{regexp.MustCompile(`(?i)\bCERCADO\b`), "Cochabamba"},
}

// Palabras prohibidas dentro de un nombre o apellido (membretes, etiquetas laterales, estados civiles, extensiones)
var forbiddenNameTokens = map[string]bool{
	"ESTADO": true, "PLURINACIONAL": true, "BOLIVIA": true, "SERVICIO": true, "GENERAL": true,
	"IDENTIFICACION": true, "IDENTIFICACIÓN": true, "PERSONAL": true, "SEGIP": true,
	"CEDULA": true, "CÉDULA": true, "IDENTIDAD": true, "REPUBLICA": true, "REPÚBLICA": true,
	"REGISTRO": true, "NACIONAL": true, "PERSONAS": true, "RENIEC": true, "DOCUMENTO": true,
	"CARNET": true, "PASAPORTE": true, "PASSPORT": true, "SERIE": true, "SECCION": true, "SECCIÓN": true,
	"FIRMA": true, "TITULAR": true, "DIRECTOR": true, "DIRECTORA": true, "HUELLA": true, "PULGAR": true,
	"DERECHO": true, "IZQUIERDO": true, "INDICE": true, "ÍNDICE": true,
	"FOTOGRAFIA": true, "FOTOGRAFÍA": true, "IMPRESION": true, "IMPRESIÓN": true, "DIGITAL": true,
	"CERTIFICA": true, "PERTENECE": true, "PERTENECEN": true, "BICENTENARIO": true,
	"MINISTERIO": true, "GOBIERNO": true, "POLICIA": true, "POLICÍA": true,
	"VALIDEZ": true, "VALIDA": true, "VÁLIDA": true, "VALIDO": true, "VÁLIDO": true,
	"VIGENCIA": true, "HASTA": true, "INDEFINIDA": true, "INDEFINIDO": true,
	"EMISION": true, "EMISIÓN": true, "EMITIDO": true, "EMITIDA": true,
	"EXPEDICION": true, "EXPEDICIÓN": true, "EXPEDIDA": true, "EXPEDIDO": true,
	"EXPIRACION": true, "EXPIRACIÓN": true, "VENCIMIENTO": true, "VENCE": true,
	"FECHA": true, "NACIMIENTO": true, "NACIDO": true, "NACIDA": true, "LUGAR": true, "NATURAL": true,
	"PROCEDENCIA": true, "NACIONALIDAD": true, "EXTRANJERA": true, "EXTRANJERO": true,
	"SEXO": true, "GENERO": true, "GÉNERO": true, "MASCULINO": true, "FEMENINO": true,
	"HOMBRE": true, "MUJER": true, "CIVIL": true, "SOLTERO": true, "SOLTERA": true,
	"CASADO": true, "CASADA": true, "VIUDO": true, "VIUDA": true, "DIVORCIADO": true, "DIVORCIADA": true,
	"OCUPACION": true, "OCUPACIÓN": true, "PROFESION": true, "PROFESIÓN": true,
	"DOMICILIO": true, "DIRECCION": true, "DIRECCIÓN": true, "NOMBRE": true, "NOMBRES": true,
	"APELLIDO": true, "APELLIDOS": true, "PRIMER": true, "SEGUNDO": true, "PATERNO": true, "MATERNO": true,
	"FOLIO": true, "PARTIDA": true, "LIBRO": true, "OFICIALIA": true, "OFICIALÍA": true,
	"GRUPO": true, "SANGUINEO": true, "SANGUÍNEO": true, "FACTOR": true, "OBSERVACIONES": true,
	"DATOS": true, "PERSONALES": true, "COMPLEMENTO": true, "CODIGO": true, "CÓDIGO": true,
	"ORIGINAL": true, "DUPLICADO": true, "RENOVACION": true, "RENOVACIÓN": true, "REPOSICION": true, "REPOSICIÓN": true,
	"TRAMITE": true, "TRÁMITE": true, "DEPARTAMENTAL": true,
	"NRO": true, "NUM": true, "NUMERO": true, "NÚMERO": true, "BOLIVIANA": true, "BOLIVIANO": true,
	"QUE": true, "CON": true, "POR": true, "PARA": true,
	"LP": true, "LPZ": true, "SC": true, "SCZ": true, "CB": true, "CBA": true,
	"OR": true, "ORU": true, "PT": true, "PSI": true, "TJ": true, "TJA": true,
	"CH": true, "CHQ": true, "BE": true, "BN": true, "PD": true, "PND": true,
}

// Cortadores de línea: si en la misma fila horizontal del nombre aparece una etiqueta de otra columna o una fecha, se corta ahí.
var reInlineLabelCut = regexp.MustCompile(`(?i)(?:\b(?:FECHA|FECH|NACIMIENTO|NACIM|NACID[OA]|LUGAR|SEXO|G[EÉ]NERO|MASCULIN[OA]|FEMENIN[OA]|ESTADO\s+CIVIL|EST\.\s*CIVIL|OCUPACI[OÓ]N|PROFESI[OÓ]N|DOMICILIO|SERIE|SECCI[OÓ]N|EXPIRACI[OÓ]N|VENCIMIENTO|EMISI[OÓ]N|EMITID[OA]|VALID[AO]|VALIDEZ|VIGENCIA|HASTA|INDEFINID[OA]|FIRMA|HUELLA|NACIONALIDAD|APELLIDOS?|PRIMER\s+APELLIDO|SEGUNDO\s+APELLIDO|GRUPO\s+SANGU|OBSERVACIONES|N[°ºo.]|C\.?\s*I\.?|NRO)\b|\b\d{1,2}\s*[/.-]\s*\d{1,2}\s*[/.-]\s*(?:19|20)?\d{2}\b).*$`)

// ParseAndMergeSides analiza el texto del frente y del reverso y fusiona los campos extraídos.
func ParseAndMergeSides(frontText, backText string) ParseResult {
	fRes := ParseDocumentText(frontText)
	if strings.TrimSpace(backText) == "" {
		return fRes
	}
	bRes := ParseDocumentText(backText)
	merged := MergeFieldsWithMRZPriority(fRes.Fields, fRes.DetectedMRZ, bRes.Fields, bRes.DetectedMRZ)
	detectedMRZ := fRes.DetectedMRZ || bRes.DetectedMRZ

	// Si alguna de las caras no tenía el nombre completo pero al unir frente + reverso se complementan
	if merged.FullName == "" {
		cRes := ParseDocumentText(frontText + "\n" + backText)
		if cRes.Fields.FullName != "" {
			merged.FirstName = firstNonEmpty(merged.FirstName, cRes.Fields.FirstName)
			merged.LastName = firstNonEmpty(merged.LastName, cRes.Fields.LastName)
			merged.FullName = cRes.Fields.FullName
		}
	}

	if merged.Gender == "" && merged.FirstName != "" {
		merged.Gender = InferGenderFromBoliviaFirstName(merged.FirstName)
	}

	conf := "PARCIAL"
	count := countNonEmpty(merged)
	if detectedMRZ {
		conf = "ALTA (MRZ Verificado)"
	} else if (fRes.IsBolivia || bRes.IsBolivia) && merged.FullName != "" && merged.DocumentNumber != "" {
		conf = "ALTA (SEGIP Bolivia Verificado)"
	} else if count >= 3 {
		conf = "MEDIA-ALTA"
	}

	return ParseResult{
		Service:          "sisaloja-ocr",
		Engine:           "hybrid-mrz-segip-bolivia",
		Confidence:       conf,
		DetectedMRZ:      detectedMRZ,
		IsBolivia:        fRes.IsBolivia || bRes.IsBolivia,
		BoliviaSubtype:   firstNonEmpty(bRes.BoliviaSubtype, fRes.BoliviaSubtype),
		ExtractionMethod: firstNonEmpty(bRes.ExtractionMethod, fRes.ExtractionMethod),
		Fields:           merged,
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
	bolAnalysis := IdentifyAndParseBoliviaDocument(lines, text)

	if bolAnalysis.ExtractionMethod == "MRZ_IDBOL" {
		mrzDetected = true
	}

	docType := mrzFields.DocumentType
	if docType == "" {
		docType = visFields.DocumentType
	}
	if docType == "" && bolAnalysis.IsBolivia {
		docType = bolAnalysis.Fields.DocumentType
	}
	if docType == "" {
		docType = "CI"
	}

	// Prioridad de Nombre Completo:
	// 1. MRZ verificado (IDBOL o ICAO 9303)
	// 2. Motor especializado SEGIP Bolivia (usa diccionario y boliviaNonNameWords)
	// 3. Etiquetas visuales generales
	finalFirst := visFields.FirstName
	finalLast := visFields.LastName
	finalFull := visFields.FullName

	if bolAnalysis.IsBolivia && bolAnalysis.Fields.FullName != "" && docType != "PASAPORTE" {
		finalFull = bolAnalysis.Fields.FullName
		finalFirst = firstNonEmpty(bolAnalysis.Fields.FirstName, finalFirst)
		finalLast = firstNonEmpty(bolAnalysis.Fields.LastName, finalLast)
	}

	if bolAnalysis.ExtractionMethod == "MRZ_IDBOL" && bolAnalysis.Fields.FullName != "" {
		finalFull = bolAnalysis.Fields.FullName
		finalFirst = firstNonEmpty(bolAnalysis.Fields.FirstName, finalFirst)
		finalLast = firstNonEmpty(bolAnalysis.Fields.LastName, finalLast)
	} else if mrzDetected && mrzFields.FullName != "" {
		finalFull = mrzFields.FullName
		if mrzFields.FirstName != "" {
			finalFirst = mrzFields.FirstName
		}
		if mrzFields.LastName != "" {
			finalLast = mrzFields.LastName
		}
	}

	// Priorizar número de documento del MRZ; solo enriquecer con extensión/complemento si comparten dígitos base
	visDoc := visFields.DocumentNumber
	if bolAnalysis.IsBolivia && bolAnalysis.Fields.DocumentNumber != "" {
		visDoc = bolAnalysis.Fields.DocumentNumber
	}
	docNum := mergeDocNumberWithMRZ(mrzFields.DocumentNumber, visDoc, mrzDetected)
	if bolAnalysis.IsBolivia && bolAnalysis.Fields.DocumentNumber != "" {
		docNum = mergeDocNumberWithMRZ(docNum, bolAnalysis.Fields.DocumentNumber, mrzDetected && mrzFields.DocumentNumber != "")
	}

	nationality := firstNonEmpty(mrzFields.Nationality, visFields.Nationality)
	if nationality == "" && bolAnalysis.IsBolivia && (docNum != "" || finalFull != "") {
		nationality = "Boliviana"
	}

	gender := firstNonEmpty(mrzFields.Gender, visFields.Gender)
	if gender == "" && bolAnalysis.Fields.Gender != "" {
		gender = bolAnalysis.Fields.Gender
	}
	if gender == "" && finalFirst != "" {
		gender = InferGenderFromBoliviaFirstName(finalFirst)
	}

	birthDate := firstNonEmpty(mrzFields.BirthDate, visFields.BirthDate)
	if birthDate == "" && bolAnalysis.Fields.BirthDate != "" {
		birthDate = bolAnalysis.Fields.BirthDate
	}

	fields := DocumentFields{
		DocumentType:   docType,
		DocumentNumber: docNum,
		FullName:       finalFull,
		FirstName:      finalFirst,
		LastName:       finalLast,
		Nationality:    nationality,
		Gender:         gender,
		BirthDate:      birthDate,
		Occupation:     visFields.Occupation,
		OriginCity:     visFields.OriginCity,
	}

	conf := "PARCIAL"
	if mrzDetected {
		conf = "ALTA (MRZ Verificado)"
	} else if bolAnalysis.IsBolivia && fields.FullName != "" && fields.DocumentNumber != "" {
		conf = "ALTA (SEGIP Bolivia Verificado)"
	} else if countNonEmpty(fields) >= 3 {
		conf = "MEDIA-ALTA"
	}

	return ParseResult{
		Service:          "sisaloja-ocr",
		Engine:           "hybrid-mrz-segip-bolivia",
		Confidence:       conf,
		DetectedMRZ:      mrzDetected,
		IsBolivia:        bolAnalysis.IsBolivia,
		BoliviaSubtype:   bolAnalysis.Subtype,
		ExtractionMethod: bolAnalysis.ExtractionMethod,
		Fields:           fields,
	}
}

// MergeFieldsWithMRZPriority fusiona Frente y Reverso, combinando Nombres + Apellidos si fueron leídos por separado y priorizando MRZ.
func MergeFieldsWithMRZPriority(current DocumentFields, curMRZ bool, incoming DocumentFields, incMRZ bool) DocumentFields {
	out := current
	if incoming.DocumentType != "" && (out.DocumentType == "" || out.DocumentType == "CI") {
		out.DocumentType = incoming.DocumentType
	}
	if incoming.DocumentNumber != "" {
		if incMRZ && !curMRZ {
			out.DocumentNumber = mergeDocNumberWithMRZ(incoming.DocumentNumber, out.DocumentNumber, true)
		} else {
			out.DocumentNumber = mergeDocNumberWithMRZ(out.DocumentNumber, incoming.DocumentNumber, curMRZ)
		}
	}

	// Combinar FirstName y LastName de ambas lecturas
	if incMRZ && incoming.FullName != "" {
		out.FullName = incoming.FullName
		out.FirstName = firstNonEmpty(incoming.FirstName, out.FirstName)
		out.LastName = firstNonEmpty(incoming.LastName, out.LastName)
	} else if !curMRZ {
		if out.FirstName == "" && incoming.FirstName != "" {
			out.FirstName = incoming.FirstName
		} else if incoming.FirstName != "" && len(strings.Fields(incoming.FirstName)) > len(strings.Fields(out.FirstName)) && len(strings.Fields(incoming.FirstName)) <= 3 && strings.HasPrefix(strings.ToUpper(incoming.FirstName), strings.ToUpper(out.FirstName)) {
			out.FirstName = incoming.FirstName
		}
		if out.LastName == "" && incoming.LastName != "" {
			out.LastName = incoming.LastName
		} else if incoming.LastName != "" && len(strings.Fields(incoming.LastName)) > len(strings.Fields(out.LastName)) && len(strings.Fields(incoming.LastName)) <= 3 && strings.HasPrefix(strings.ToUpper(incoming.LastName), strings.ToUpper(out.LastName)) {
			out.LastName = incoming.LastName
		}

		if out.FirstName != "" && out.LastName != "" {
			out.FullName = strings.TrimSpace(out.FirstName + " " + out.LastName)
		} else if incoming.FullName != "" && out.FullName == "" {
			out.FullName = incoming.FullName
			if incoming.FirstName != "" {
				out.FirstName = incoming.FirstName
			}
			if incoming.LastName != "" {
				out.LastName = incoming.LastName
			}
		} else if incoming.FullName != "" && len(strings.Fields(incoming.FullName)) > len(strings.Fields(out.FullName)) && len(strings.Fields(incoming.FullName)) <= 4 {
			out.FullName = incoming.FullName
			if incoming.FirstName != "" {
				out.FirstName = incoming.FirstName
			}
			if incoming.LastName != "" {
				out.LastName = incoming.LastName
			}
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
	return mergeDocNumberWithMRZ(a, b, false)
}

func mergeDocNumberWithMRZ(primary, secondary string, primaryIsMRZ bool) string {
	a := strings.TrimSpace(primary)
	b := strings.TrimSpace(secondary)
	if a != "" && b != "" {
		da := onlyDigits(a)
		db := onlyDigits(b)
		if da == db || (da != "" && strings.HasPrefix(db, da)) {
			if len(b) > len(a) {
				return b
			}
			return a
		}
		if primaryIsMRZ {
			return a
		}
		// Si alguno tiene extensión departamental boliviana y el otro no, preferir el que tiene extensión
		hasExtA := reBolCIWithExt.MatchString(a)
		hasExtB := reBolCIWithExt.MatchString(b)
		if hasExtB && !hasExtA {
			return b
		}
		if hasExtA && !hasExtB {
			return a
		}
		return a
	}
	if b != "" {
		return b
	}
	return a
}

var (
	reMRZPrefix = regexp.MustCompile(`(?:ID|I<|P<|A<|C<)[A-Z0-9]{3}`)
	reTD1Line1  = regexp.MustCompile(`(?:I[D<A-Z0-9]|A[C<A-Z0-9]|C[I<A-Z0-9])([A-Z08]{3})([A-Z0-9<]{7,25})`)
	reTD3Line1  = regexp.MustCompile(`^P[<A-Z]([A-Z08]{3})([A-Z<]{10,40})`)
	reTD1Line2  = regexp.MustCompile(`([0-9OQDILSZB]{6})[0-9OQDILSZB]([MF<])([0-9OQDILSZB]{6})[0-9OQDILSZB]([A-Z0-9<]{3})`)
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
			countryCode := fixMRZLetters(m[1])
			if nat, ok := countryCodeMap[countryCode]; ok {
				detected = true
				res.Nationality = nat
				if countryCode == "BOL" {
					res.DocumentType = "CI"
				} else {
					res.DocumentType = "DNI"
				}
				rawDoc := strings.Split(m[2], "<")[0]
				cleanDoc := cleanAlnum(fixMRZDigits(rawDoc))
				if len(cleanDoc) >= 5 {
					res.DocumentNumber = cleanDoc
				}
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
			natCode := fixMRZLetters(m[4])
			if nat, ok := countryCodeMap[natCode]; ok {
				detected = true
				if bDate := parseMRZDate(m[1]); bDate != "" {
					res.BirthDate = bDate
				}
				if m[2] == "M" || m[2] == "F" {
					res.Gender = m[2]
				}
				if res.Nationality == "" {
					res.Nationality = nat
				}
			}
		}

		if strings.Contains(l, "<<") &&
			!strings.Contains(l, "IDBOL") &&
			!strings.Contains(l, "1DBOL") &&
			!strings.Contains(l, "BOL<<") &&
			!strings.HasPrefix(l, "ID") &&
			!strings.HasPrefix(l, "I<") &&
			!strings.HasPrefix(l, "P<") &&
			len(onlyDigits(l)) <= 3 &&
			!reHas4Digit.MatchString(l) {
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
	return parseBoliviaMRZNameLine(mrzLine)
}

var (
	rePassport = regexp.MustCompile(`(?i)PASAPORTE|PASSPORT`)
	reDNI      = regexp.MustCompile(`(?i)\b(DNI|RENIEC|DOCUMENTO\s+NACIONAL)\b`)
	reBolivia  = regexp.MustCompile(`(?i)\b(BOLIVIA|PLURINACIONAL|SEGIP|BOLIVIANA|BOLIVIANO)\b`)
	rePeru     = regexp.MustCompile(`(?i)\b(PER[UÚ]|RENIEC|PERUANA|PERUANO)\b`)
	reArg      = regexp.MustCompile(`(?i)\b(ARGENTINA|ARGENTINO)\b`)
	reChile    = regexp.MustCompile(`(?i)\b(CHILE|CHILENA|CHILENO)\b`)
	reBrasil   = regexp.MustCompile(`(?i)\b(BRASIL|BRASILE[ÑN]A)\b`)

	reDoc1 = regexp.MustCompile(`(?i)\b(?:N[°º*№']|N[o0]\.(?:\s|$)|N[o0]\s*[:#-]|NRO\.?|C\.?\s*I\.?|CEDULA|C[EÉ]DULA|DOCUMENTO)\s*[:.#-]?\s*(([0-9]{5,9}|[0-9]{1,3}(?:\.[0-9]{3}){1,2}|[0-9]{1,3}[ \t][0-9]{3}[ \t][0-9]{3})(?:[ \t]*[-][ \t]*[0-9A-Z]{1,3}\b)?(?:[ \t]+(?:LP|LPZ|SC|SCZ|CB|CBA|OR|ORU|PT|PSI|TJ|TJA|CH|CHQ|BE|BN|PD|PND)\b)?)`)
	reDoc2 = regexp.MustCompile(`(?i)\b([0-9]{5,9})[ \t]+(LP|LPZ|SC|SCZ|CB|CBA|OR|ORU|PT|PSI|TJ|TJA|CH|CHQ|BE|BN|PD|PND)\b`)
	reDoc3 = regexp.MustCompile(`\b([0-9]{7,9})\b`)

	reTextDate = regexp.MustCompile(`(?i)\b([0-9O]{1,2})\s*(?:DE\s+|DEL\s+|[-/.\s]+)(ENERO|FEBRERO|MARZO|ABRIL|MAYO|JUNIO|JULIO|AGOSTO|SEPTIEMBRE|SETIEMBRE|OCTUBRE|NOVIEMBRE|DICIEMBRE|ENE|FEB|MAR|ABR|MAY|JUN|JUL|AGO|SEP|SET|OCT|NOV|DIC)\s*(?:DE\s+|DEL\s+|[-/.\s]+)((?:19|2[0O])[0-9O]{2})\b`)
	reNacDate  = regexp.MustCompile(`(?i)(?:NAC[I1]M[I1][E3]NT[O0]|NAC[I1]D[O0]|F\.?\s*NAC)[\s\S]{0,45}?\b([0-9O]{1,2})\s*(?:[/.-]\s*|\s+)([0-9O]{1,2})\s*(?:[/.-]\s*|\s+)((?:19|2[0O])[0-9O]{2})\b`)
	reAnyDate  = regexp.MustCompile(`\b([0-9O]{1,2})\s*(?:[/.-]\s*|\s+)([0-9O]{1,2})\s*(?:[/.-]\s*|\s+)((?:19[0-9O]{2}|2[0O][012O][0-9O]))\b`)
	reEmitExp  = regexp.MustCompile(`(?i)\b(EM[I1]S[I1][OÓ0]N|EM[I1]T[I1]D|EXP[I1]RAC[I1][OÓ0]N|V[E3]NC[I1]M[I1][E3]NT[O0]|VAL[I1]D[AO]|V[I1]G[E3]NC[I1]A)\b`)

	// Etiquetas tolerantes a errores comunes de OCR (0 por O, 1 por I) y sin ancla ^ rígida
	reBothApellidoHeaders = regexp.MustCompile(`(?i)PR[I1l]MER\s+AP[E3][LI1]{1,2}[I1l][D0O][O0].*S[E3]GUND[O0]\s+AP[E3][LI1]{1,2}[I1l][D0O][O0]|AP[E3][LI1]{1,2}[I1l][D0O][O0]\s+PAT[E3]RN[O0].*AP[E3][LI1]{1,2}[I1l][D0O][O0]\s+MAT[E3]RN[O0]`)
	reNombres             = regexp.MustCompile(`(?i)\b[MN][O0]M[B8]R[E3][S5]?\b\s*[:.;-]?\s*(.*)$`)
	reApellido1           = regexp.MustCompile(`(?i)\b(?:PR[I1l]MER\s+AP[E3][LI1]{1,2}[I1l][D0O][O0]|AP[E3][LI1]{1,2}[I1l][D0O][O0]\s+PAT[E3]RN[O0])\s*[:.;-]?\s*(.*)$`)
	reApellido2           = regexp.MustCompile(`(?i)\b(?:S[E3]GUND[O0]\s+AP[E3][LI1]{1,2}[I1l][D0O][O0]|AP[E3][LI1]{1,2}[I1l][D0O][O0]\s+MAT[E3]RN[O0])\s*[:.;-]?\s*(.*)$`)
	reApellidos           = regexp.MustCompile(`(?i)\bAP[E3][LI1]{1,2}[I1l][D0O][O0][S5]?\b\s*[:.;-]?\s*(.*)$`)
	reClassicA            = regexp.MustCompile(`(?i)(?:^\s*|P[E3]RT[E3]N[E3]C[E3]N?\s+)A\s*(?:[:.;-]\s*|\s+)(.*)$`)
	reOcupacion           = regexp.MustCompile(`(?i)\b(?:[O0]CUPAC[I1][OÓ0]N|PR[O0]FES[I1][OÓ0]N)(?:\s*[/]\s*(?:PR[O0]FES[I1][OÓ0]N|[O0]CUPAC[I1][OÓ0]N))?\s*[:.;-]?\s*(.*)$`)
	reOrigen              = regexp.MustCompile(`(?i)\b(?:LUGAR\s+DE\s+NAC[I1]M[I1]ENT[O0]|NATURAL\s+DE|NAC[I1]D[O0]\s+EN|DEPARTAMENT[O0]|EN)\s*[:.;-]\s*(.*)$`)
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

	if out.Nationality == "Boliviana" || out.Nationality == "" {
		out.DocumentNumber = ExtractBoliviaDocumentNumber(lines, fullText)
	}
	if out.DocumentNumber == "" {
		if m := reDoc1.FindStringSubmatch(fullText); m != nil && len(onlyDigits(m[1])) >= 6 {
			out.DocumentNumber = normalizeDocNumberMatch(m[1])
		} else if m := reDoc2.FindStringSubmatch(fullText); m != nil {
			out.DocumentNumber = strings.ToUpper(m[1] + " " + m[2])
		} else if m := reDoc3.FindStringSubmatch(fullText); m != nil {
			out.DocumentNumber = m[1]
		}
	}

	out.BirthDate = extractBirthDateFromLines(lines, fullText)

	bolFirst, bolLast, bolFull, _, _ := extractBoliviaPersonName(lines)
	out.FirstName = bolFirst
	out.LastName = bolLast
	out.FullName = bolFull

	for i, line := range lines {
		nextLine := ""
		if i+1 < len(lines) {
			nextLine = lines[i+1]
		}

		// OCUPACIÓN / PROFESIÓN
		if m := reOcupacion.FindStringSubmatch(line); m != nil && out.Occupation == "" {
			inline := cleanLabelValue(m[1])
			if len(inline) >= 3 {
				out.Occupation = inline
			} else if nextLine != "" && !strings.Contains(nextLine, "<<") && !reAnyDate.MatchString(nextLine) && !reBolStartsMetaLine.MatchString(nextLine) {
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

	if out.OriginCity == "" {
		out.OriginCity = detectBoliviaCity(fullText)
	}
	if out.OriginCity != "" && out.Nationality == "" {
		out.Nationality = "Boliviana"
	}
	if out.Gender == "" && out.FirstName != "" {
		out.Gender = InferGenderFromBoliviaFirstName(out.FirstName)
	}

	return out
}

func extractBirthDateFromLines(lines []string, fullText string) string {
	// 1. Fecha textual explícita ("14 DE MAYO DE 1992" o "2O DE NOVIEMBRE DEL 2OO3")
	for _, line := range lines {
		if reEmitExp.MatchString(line) && !regexp.MustCompile(`(?i)NAC`).MatchString(line) {
			continue
		}
		if m := reTextDate.FindStringSubmatch(line); m != nil {
			dd, _ := strconv.Atoi(strings.ReplaceAll(strings.ToUpper(m[1]), "O", "0"))
			mm := monthsES[strings.ToUpper(m[2])]
			yyyy := strings.ReplaceAll(strings.ToUpper(m[3]), "O", "0")
			if mm != "" && dd >= 1 && dd <= 31 {
				return fmt.Sprintf("%s-%s-%02d", yyyy, mm, dd)
			}
		}
	}
	if m := reTextDate.FindStringSubmatch(fullText); m != nil {
		dd, _ := strconv.Atoi(strings.ReplaceAll(strings.ToUpper(m[1]), "O", "0"))
		mm := monthsES[strings.ToUpper(m[2])]
		yyyy := strings.ReplaceAll(strings.ToUpper(m[3]), "O", "0")
		if mm != "" && dd >= 1 && dd <= 31 {
			return fmt.Sprintf("%s-%s-%02d", yyyy, mm, dd)
		}
	}

	// 2. Fecha cercana a etiqueta NACIMIENTO / NACIDO
	if dm := reNacDate.FindStringSubmatch(fullText); dm != nil {
		if formatted := formatValidDMY(dm[1], dm[2], dm[3]); formatted != "" {
			return formatted
		}
	}

	// 3. Elegir la fecha más antigua válida que NO esté precedida inmediatamente (28 chars) por EMISION/EXPIRACION
	maxBirthYear := time.Now().Year() - 5
	bestYear := 9999
	bestDate := ""

	for _, line := range lines {
		matches := reAnyDate.FindAllStringSubmatchIndex(line, -1)
		for _, loc := range matches {
			start := loc[0] - 28
			if start < 0 {
				start = 0
			}
			recentPrefix := line[start:loc[0]]
			if reEmitExp.MatchString(recentPrefix) {
				continue
			}
			ddStr := line[loc[2]:loc[3]]
			mmStr := line[loc[4]:loc[5]]
			yyyyStr := line[loc[6]:loc[7]]
			if formatted := formatValidDMY(ddStr, mmStr, yyyyStr); formatted != "" {
				cleanY := strings.ReplaceAll(strings.ToUpper(yyyyStr), "O", "0")
				y, _ := strconv.Atoi(cleanY)
				if y >= 1915 && y <= maxBirthYear && y < bestYear {
					bestYear = y
					bestDate = formatted
				}
			}
		}
	}
	if bestDate != "" {
		return bestDate
	}

	if dm := reAnyDate.FindStringSubmatch(fullText); dm != nil {
		return formatValidDMY(dm[1], dm[2], dm[3])
	}
	return ""
}

func formatValidDMY(ddStr, mmStr, yyyyStr string) string {
	dd, err1 := strconv.Atoi(strings.ReplaceAll(strings.ToUpper(ddStr), "O", "0"))
	mm, err2 := strconv.Atoi(strings.ReplaceAll(strings.ToUpper(mmStr), "O", "0"))
	cleanY := strings.ReplaceAll(strings.ToUpper(yyyyStr), "O", "0")
	if err1 == nil && err2 == nil && mm >= 1 && mm <= 12 && dd >= 1 && dd <= 31 && len(cleanY) == 4 {
		return fmt.Sprintf("%s-%02d-%02d", cleanY, mm, dd)
	}
	return ""
}

// sanitizePersonTokens limpia una línea de nombre o apellido:
// 1. Corta la línea si aparece una etiqueta de otra columna o fecha.
// 2. Evalúa tokens crudos antes de dividir por dígitos para descartar basura OCR como "PLUR1NAC10NAL" o "T1TULAR".
// 3. Descarta tokens de 1 sola letra, palabras sin vocales, siglas de departamentos y palabras de membrete.
func sanitizePersonTokens(raw string, maxWords int) string {
	if strings.TrimSpace(raw) == "" {
		return ""
	}
	if maxWords <= 0 {
		maxWords = 3
	}

	// 1. Cortar en cuanto aparezca una etiqueta intrusa o fecha en la misma línea horizontal
	cut := reInlineLabelCut.ReplaceAllString(raw, "")
	rawTokens := strings.Fields(cut)

	var valid []string
	connectors := map[string]bool{"DE": true, "DEL": true, "LA": true, "LAS": true, "LOS": true}

	for _, tok := range rawTokens {
		cleanedTok := cleanSingleRawNameToken(tok)
		if cleanedTok == "" {
			continue
		}
		// Puede contener espacios si hubo signos internos como guión/punto
		for _, w := range strings.Fields(cleanedTok) {
			up := strings.ToUpper(w)
			if isForbiddenOrHeaderToken(up) {
				continue
			}
			if connectors[up] {
				if len(valid) > 0 {
					valid = append(valid, up)
				}
				continue
			}
			rLen := len([]rune(up))
			if rLen < 2 || rLen > 18 {
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
		if len(valid) >= maxWords {
			break
		}
	}

	for len(valid) > 0 && connectors[valid[len(valid)-1]] {
		valid = valid[:len(valid)-1]
	}

	return strings.Join(valid, " ")
}

func cleanSingleRawNameToken(tok string) string {
	trimmed := strings.Trim(tok, ".,:;-_|/\"'()[]{}<>«»°º*")
	if trimmed == "" {
		return ""
	}
	digitCount := 0
	for _, r := range trimmed {
		if unicode.IsDigit(r) {
			digitCount++
		}
	}
	if digitCount > 0 {
		// Si tiene dígitos mezclados con letras (ej. PLUR1NAC10NAL, T1TULAR, 14/05/1992):
		// normalizar leetspeak; si coincide con palabra prohibida o tiene >=2 dígitos, descartar completo.
		leetspeak := fixOCRLeetspeakToLetters(trimmed)
		alphaOnly := keepAlphaOnly(leetspeak)
		if isForbiddenOrHeaderToken(alphaOnly) || boliviaNonNameWords[alphaOnly] {
			return ""
		}
		if digitCount == 1 {
			if boliviaFirstNamesSet[alphaOnly] || boliviaSurnamesSet[alphaOnly] {
				return alphaOnly
			}
			if len([]rune(alphaOnly)) >= 4 && allWordsLookLikeSpanishNames([]string{alphaOnly}) {
				return alphaOnly
			}
		}
		return ""
	}
	return keepAlphaAndSpaces(trimmed)
}

func fixOCRLeetspeakToLetters(s string) string {
	r := strings.NewReplacer("0", "O", "1", "I", "3", "E", "4", "A", "5", "S", "8", "B")
	return r.Replace(strings.ToUpper(s))
}

func keepAlphaOnly(s string) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) {
			b.WriteRune(unicode.ToUpper(r))
		}
	}
	return b.String()
}

func keepAlphaAndSpaces(s string) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsSpace(r) {
			b.WriteRune(r)
		} else {
			b.WriteRune(' ')
		}
	}
	return b.String()
}

func foldAccents(s string) string {
	r := strings.NewReplacer(
		"Á", "A", "À", "A", "Ä", "A",
		"É", "E", "È", "E", "Ë", "E",
		"Í", "I", "Ì", "I", "Ï", "I",
		"Ó", "O", "Ò", "O", "Ö", "O",
		"Ú", "U", "Ù", "U", "Ü", "U",
	)
	return r.Replace(strings.ToUpper(s))
}

func isForbiddenOrHeaderToken(up string) bool {
	if up == "" {
		return true
	}
	return forbiddenNameTokens[up] || forbiddenNameTokens[foldAccents(up)]
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
		if isForbiddenOrHeaderToken(up) || len([]rune(up)) < 2 || !hasSpanishVowel(up) {
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
