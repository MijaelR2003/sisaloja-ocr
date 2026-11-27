package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MijaelR2003/sisaloja-ocr/internal/parser"
)

func TestOCRMicroservice_BoliviaFrontAndBackMRZ(t *testing.T) {
	app := NewApp()

	// 1. Health check
	reqHealth := httptest.NewRequest(http.MethodGet, "/health", nil)
	respHealth, err := app.Test(reqHealth)
	if err != nil || respHealth.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on /health, got %v", respHealth.StatusCode)
	}

	// 2. Escaneo combinando Frente (SEGIP Bolivia) + Reverso (Ocupación, Procedencia y Zona MRZ TD1)
	payload := map[string]string{
		"front_text": `
			ESTADO PLURINACIONAL DE BOLIVIA
			SERVICIO GENERAL DE IDENTIFICACION PERSONAL
			CEDULA DE IDENTIDAD
			N° 8392011 LP
			NOMBRES: ALEJANDRO
			APELLIDOS: TORREZ SILVA
			FECHA DE NACIMIENTO: 14 DE MAYO DE 1992
		`,
		"back_text": `
			LUGAR DE NACIMIENTO: MURILLO - LA PAZ
			OCUPACION / PROFESION: INGENIERO DE SISTEMAS
			ESTADO CIVIL: SOLTERO
			IDBOL8392011<<4<<<<<<<<<<<<<<<
			9205148M3011205BOL<<<<<<<<<<<6
			TORREZ<SILVA<<ALEJANDRO<<<<<<<
		`,
	}
	body, _ := json.Marshal(payload)
	reqScan := httptest.NewRequest(http.MethodPost, "/api/v1/ocr/scan", bytes.NewReader(body))
	reqScan.Header.Set("Content-Type", "application/json")

	respScan, err := app.Test(reqScan)
	if err != nil || respScan.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on /api/v1/ocr/scan, got %v", respScan.StatusCode)
	}

	var res parser.ParseResult
	if err := json.NewDecoder(respScan.Body).Decode(&res); err != nil {
		t.Fatalf("failed to decode scan response: %v", err)
	}

	if !res.DetectedMRZ {
		t.Fatalf("expected DetectedMRZ=true")
	}
	if res.Fields.DocumentType != "CI" {
		t.Fatalf("expected DocumentType=CI, got %s", res.Fields.DocumentType)
	}
	if res.Fields.DocumentNumber != "8392011 LP" {
		t.Fatalf("expected DocumentNumber='8392011 LP', got '%s'", res.Fields.DocumentNumber)
	}
	if res.Fields.FullName != "Alejandro Torrez Silva" {
		t.Fatalf("expected FullName='Alejandro Torrez Silva', got '%s'", res.Fields.FullName)
	}
	if res.Fields.Nationality != "Boliviana" {
		t.Fatalf("expected Nationality='Boliviana', got '%s'", res.Fields.Nationality)
	}
	if res.Fields.Gender != "M" {
		t.Fatalf("expected Gender='M', got '%s'", res.Fields.Gender)
	}
	if res.Fields.BirthDate != "1992-05-14" {
		t.Fatalf("expected BirthDate='1992-05-14', got '%s'", res.Fields.BirthDate)
	}
	if res.Fields.Occupation != "Ingeniero De Sistemas" {
		t.Fatalf("expected Occupation='Ingeniero De Sistemas', got '%s'", res.Fields.Occupation)
	}
	if res.Fields.OriginCity != "La Paz" {
		t.Fatalf("expected OriginCity='La Paz', got '%s'", res.Fields.OriginCity)
	}
}

func TestOCRMicroservice_NoisyColumnsAndSplitLastNames(t *testing.T) {
	app := NewApp()

	payload := map[string]string{
		"front_text": `
			ESTADO PLURINACIONAL DE BOLIVIA
			SERVICIO GENERAL DE IDENTIFICACION PERSONAL
			CEDULA DE IDENTIDAD N° 9412833 CB
			NOMBRES: MIJAEL   FECHA DE NACIMIENTO 20/11/2003   SEXO M
			PRIMER APELLIDO   SEGUNDO APELLIDO
			RODRIGUEZ   VARGAS   ESTADO CIVIL SOLTERO
		`,
	}
	body, _ := json.Marshal(payload)
	reqScan := httptest.NewRequest(http.MethodPost, "/api/v1/ocr/scan", bytes.NewReader(body))
	reqScan.Header.Set("Content-Type", "application/json")

	respScan, err := app.Test(reqScan)
	if err != nil || respScan.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on /api/v1/ocr/scan, got %v", respScan.StatusCode)
	}

	var res parser.ParseResult
	if err := json.NewDecoder(respScan.Body).Decode(&res); err != nil {
		t.Fatalf("failed to decode scan response: %v", err)
	}

	if res.Fields.FirstName != "Mijael" {
		t.Fatalf("expected FirstName='Mijael', got '%s'", res.Fields.FirstName)
	}
	if res.Fields.LastName != "Rodriguez Vargas" {
		t.Fatalf("expected LastName='Rodriguez Vargas', got '%s'", res.Fields.LastName)
	}
	if res.Fields.FullName != "Mijael Rodriguez Vargas" {
		t.Fatalf("expected FullName='Mijael Rodriguez Vargas', got '%s'", res.Fields.FullName)
	}
}

func TestOCRMicroservice_BoliviaMissingLabelsAndStructuralWindow(t *testing.T) {
	app := NewApp()

	payload := map[string]string{
		"front_text": `
			ESTADO PLURINACIONAL DE BOLIVIA
			SERVICIO GENERAL DE IDENTIFICACION PERSONAL
			CEDULA DE IDENTIDAD
			N° 12450987 LP
			MIJAEL
			ROJAS MAMANI
			14 DE MAYO DE 2003
			FIRMA DEL TITULAR
		`,
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ocr/scan", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %v", resp.StatusCode)
	}

	var res parser.ParseResult
	_ = json.NewDecoder(resp.Body).Decode(&res)

	if !res.IsBolivia {
		t.Fatalf("expected IsBolivia=true")
	}
	if res.Fields.FullName != "Mijael Rojas Mamani" {
		t.Fatalf("expected FullName='Mijael Rojas Mamani', got '%s'", res.Fields.FullName)
	}
	if res.Fields.DocumentNumber != "12450987 LP" {
		t.Fatalf("expected DocumentNumber='12450987 LP', got '%s'", res.Fields.DocumentNumber)
	}
	if res.Fields.BirthDate != "2003-05-14" {
		t.Fatalf("expected BirthDate='2003-05-14', got '%s'", res.Fields.BirthDate)
	}
}

func TestOCRMicroservice_BoliviaClassicCardAndDegradedIDBOL(t *testing.T) {
	resClassic := parser.ParseDocumentText(`
		ESTADO PLURINACIONAL DE BOLIVIA
		CEDULA DE IDENTIDAD
		SERIE B  SECCION 1421
		N° 6841209
		JUAN CARLOS QUISPE CONDORI
		FIRMA DEL TITULAR
	`)
	if resClassic.Fields.FullName != "Juan Carlos Quispe Condori" {
		t.Fatalf("expected 'Juan Carlos Quispe Condori', got '%s'", resClassic.Fields.FullName)
	}
	if resClassic.Fields.DocumentNumber != "6841209" {
		t.Fatalf("expected DocumentNumber='6841209', got '%s'", resClassic.Fields.DocumentNumber)
	}

	resMRZ := parser.ParseDocumentText(`
		LUGAR DE NACIMIENTO: CERCADO - COCHABAMBA
		OCUPACION: ESTUDIANTE
		1DB0L9412833<<4<<<<<<<<<<<<<<<
		0311208M3011205BOL<<<<<<<<<<<6
		ROJAS<MAMANI<MIJAELKK<<<<<<<<<
	`)
	if resMRZ.Fields.FullName != "Mijael Rojas Mamani" {
		t.Fatalf("expected 'Mijael Rojas Mamani' from degraded IDBOL MRZ, got '%s'", resMRZ.Fields.FullName)
	}
	if resMRZ.Fields.OriginCity != "Cochabamba" {
		t.Fatalf("expected OriginCity='Cochabamba', got '%s'", resMRZ.Fields.OriginCity)
	}
}

func TestBoliviaSEGIPNueva_NoisyTesseractColumnsDatesAndRealNames(t *testing.T) {
	// Simulación realista de Tesseract sobre SEGIP Nueva con:
	// - Serie antes de N° (no debe tomar la Serie como CI)
	// - Complemento -1A y extensión LP
	// - Etiquetas con ruido OCR (N0MBRES, APELL1D0S) encima de filas que comparten columna con fechas y EMISION
	// - Nombres reales antes bloqueados por error (JULIO, MONTERO, MURILLO)
	raw := `
		ESTAD0 PLUR1NAC10NAL DE B0L1V1A
		S3RV1C10 GENERAL DE 1DENT1F1CAC10N PERS0NAL
		SERIE 00918274   SECCION 120400
		CEDULA DE IDENTIDAD
		N0. 8.392.011-1A LP
		N0MBRES           FECHA DE NACIMIENTO
		JULIO CESAR       14/05/1990
		APELL1D0S         EMISION: 15/11/2023
		MONTERO MURILLO   EXPIRACION: 15/11/2028
		NACIONALIDAD: BOLIVIANA   SEXO: MASCULINO
		F1RMA DEL T1TULAR
	`
	res := parser.ParseDocumentText(raw)
	if res.Fields.DocumentNumber != "8392011-1A LP" {
		t.Fatalf("expected DocumentNumber='8392011-1A LP', got '%s'", res.Fields.DocumentNumber)
	}
	if res.Fields.FirstName != "Julio Cesar" {
		t.Fatalf("expected FirstName='Julio Cesar', got '%s'", res.Fields.FirstName)
	}
	if res.Fields.LastName != "Montero Murillo" {
		t.Fatalf("expected LastName='Montero Murillo', got '%s'", res.Fields.LastName)
	}
	if res.Fields.FullName != "Julio Cesar Montero Murillo" {
		t.Fatalf("expected FullName='Julio Cesar Montero Murillo', got '%s'", res.Fields.FullName)
	}
	if res.Fields.BirthDate != "1990-05-14" {
		t.Fatalf("expected BirthDate='1990-05-14', got '%s'", res.Fields.BirthDate)
	}
	if res.Fields.Gender != "M" {
		t.Fatalf("expected Gender='M', got '%s'", res.Fields.Gender)
	}
}

func TestBoliviaSEGIPNueva_NoLabelsLotsOfWatermarkNoiseAndFemaleGender(t *testing.T) {
	// Sin etiquetas NOMBRES/APELLIDOS y con ruido de marca de agua, leetspeak, EMISION antes de nacimiento
	raw := `
		ESTAD0 PLUR1NAC10NAL DE B0LIVIA
		BICENTENARIO DE BOLIVIA
		N° 612345 SC
		NACIONALIDAD BOLIVIANA
		VALIDA HASTA INDEFINIDA
		EMISION: 10/01/2024
		EXPIRACION: 10/01/2029
		ELIZABETH
		QUISPE CONDORI
		22/08/1988
		FIRMA DEL T1TULAR
	`
	res := parser.ParseDocumentText(raw)
	if res.Fields.DocumentNumber != "612345 SC" {
		t.Fatalf("expected DocumentNumber='612345 SC', got '%s'", res.Fields.DocumentNumber)
	}
	if res.Fields.FullName != "Elizabeth Quispe Condori" {
		t.Fatalf("expected FullName='Elizabeth Quispe Condori', got '%s'", res.Fields.FullName)
	}
	if res.Fields.Gender != "F" {
		t.Fatalf("expected Gender='F' for Elizabeth, got '%s'", res.Fields.Gender)
	}
	if res.Fields.BirthDate != "1988-08-22" {
		t.Fatalf("expected BirthDate='1988-08-22' (not emission 2024), got '%s'", res.Fields.BirthDate)
	}
}

func TestBoliviaSEGIPClasica_PertenecenAAndTarijaCercado(t *testing.T) {
	// Carnet Clásico real con "SERIE", "SECCION", "CERTIFICA Que ... pertenecen a:" con el nombre en la línea siguiente,
	// "NACIDO EL" textual y "EN: TARIJA - CERCADO" (CERCADO no debe confundirse con Cochabamba)
	raw := `
		ESTADO PLURINACIONAL DE BOLIVIA
		CEDULA DE IDENTIDAD
		SERIE A   SECCION 180200
		No. 5421987 TJ
		EL SERVICIO GENERAL DE IDENTIFICACION PERSONAL
		CERTIFICA: Que la fotografia, firma e impresion
		pertenecen a:
		MARIA ELENA VARGAS MENDOZA
		Nacido el: 03 de Julio de 1985
		En: Tarija - Cercado - Tarija
		Estado Civil: Casada   Profesion / Ocupacion: Comerciante
	`
	res := parser.ParseDocumentText(raw)
	if res.BoliviaSubtype != "SEGIP_CLASICO" {
		t.Fatalf("expected Subtype='SEGIP_CLASICO', got '%s'", res.BoliviaSubtype)
	}
	if res.Fields.DocumentNumber != "5421987 TJ" {
		t.Fatalf("expected DocumentNumber='5421987 TJ', got '%s'", res.Fields.DocumentNumber)
	}
	if res.Fields.FullName != "Maria Elena Vargas Mendoza" {
		t.Fatalf("expected FullName='Maria Elena Vargas Mendoza', got '%s'", res.Fields.FullName)
	}
	if res.Fields.BirthDate != "1985-07-03" {
		t.Fatalf("expected BirthDate='1985-07-03', got '%s'", res.Fields.BirthDate)
	}
	if res.Fields.OriginCity != "Tarija" {
		t.Fatalf("expected OriginCity='Tarija', got '%s'", res.Fields.OriginCity)
	}
	if res.Fields.Occupation != "Comerciante" {
		t.Fatalf("expected Occupation='Comerciante', got '%s'", res.Fields.Occupation)
	}
	if res.Fields.Gender != "F" {
		t.Fatalf("expected Gender='F', got '%s'", res.Fields.Gender)
	}
}

func TestBoliviaMRZ_IDBOLNoisyBorderAndDegradedLine2(t *testing.T) {
	// Línea 1 con ruido inicial '|', Línea 2 con 'B0L' y fecha de nacimiento en Line 2 con O/I,
	// Línea 3 con apellido TRINIDAD, y número visual distinto (SERIE 99887766) que NO debe sobrescribir al MRZ.
	raw := `
		DOMICILIO: ZONA CENTRAL CALLE BOLIVAR NRO 45
		OCUPACION: DOCENTE
		SERIE 99887766
		|IDBOL7091234<<2<<<<<<<<<<<<<<<
		9SO3128F3011205B0L<<<<<<<<<<<8
		VALDIVIA<TRINIDAD<<CARLA<ANDREA
	`
	res := parser.ParseDocumentText(raw)
	if !res.DetectedMRZ {
		t.Fatalf("expected DetectedMRZ=true")
	}
	if res.Fields.DocumentNumber != "7091234" {
		t.Fatalf("expected MRZ DocumentNumber='7091234' (must not be replaced by SERIE 99887766), got '%s'", res.Fields.DocumentNumber)
	}
	if res.Fields.FullName != "Carla Andrea Valdivia Trinidad" {
		t.Fatalf("expected FullName='Carla Andrea Valdivia Trinidad', got '%s'", res.Fields.FullName)
	}
	if res.Fields.BirthDate != "1995-03-12" {
		t.Fatalf("expected BirthDate='1995-03-12', got '%s'", res.Fields.BirthDate)
	}
	if res.Fields.Gender != "F" {
		t.Fatalf("expected Gender='F', got '%s'", res.Fields.Gender)
	}
}

func TestOCRMicroservice_ScanEndpointWithImagePayload(t *testing.T) {
	app := NewApp()

	// Generar una imagen PNG válida mínima en memoria para verificar que el endpoint /scan la decodifica sin fallos
	img := image.NewRGBA(image.Rect(0, 0, 120, 80))
	for y := 0; y < 80; y++ {
		for x := 0; x < 120; x++ {
			img.Set(x, y, color.RGBA{R: 245, G: 245, B: 245, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("png encode failed: %v", err)
	}
	b64 := "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())

	payload := map[string]string{
		"front_image_base64": b64,
		"front_text": `
			ESTADO PLURINACIONAL DE BOLIVIA
			CEDULA DE IDENTIDAD N° 4829103 OR
			NOMBRES: PEDRO
			APELLIDOS: MAMANI FLORES
		`,
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ocr/scan", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %v (err=%v)", resp.StatusCode, err)
	}
	var res parser.ParseResult
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if res.Fields.DocumentNumber != "4829103 OR" || res.Fields.FullName != "Pedro Mamani Flores" {
		t.Fatalf("unexpected fields from image+text scan: %+v", res.Fields)
	}
}

func TestBoliviaSEGIP_OCRDigitInUnlistedNameSpacedDateAndClassicANoColon(t *testing.T) {
	// 1. Nombre/apellido fuera del diccionario con 1 dígito confundido por OCR + fecha separada por espacios con O por 0
	raw1 := `
		ESTADO PLURINACIONAL DE BOLIVIA
		CEDULA DE IDENTIDAD
		N° 9182736 CB
		NOMBRES: YHAM1L
		APELLIDOS: K0RDOVA TINTAYA
		FECHA DE NACIMIENTO: 2O 11 2OO3
	`
	res1 := parser.ParseDocumentText(raw1)
	if res1.Fields.FirstName != "Yhamil" {
		t.Fatalf("expected FirstName='Yhamil', got '%s'", res1.Fields.FirstName)
	}
	if res1.Fields.LastName != "Kordova Tintaya" {
		t.Fatalf("expected LastName='Kordova Tintaya', got '%s'", res1.Fields.LastName)
	}
	if res1.Fields.BirthDate != "2003-11-20" {
		t.Fatalf("expected BirthDate='2003-11-20', got '%s'", res1.Fields.BirthDate)
	}

	// 2. Carnet clásico "A JUAN CARLOS QUISPE CONDORI" sin dos puntos ':'
	raw2 := `
		ESTADO PLURINACIONAL DE BOLIVIA
		CEDULA DE IDENTIDAD
		N° 6841209 LP
		A JUAN CARLOS QUISPE CONDORI
		EMISION: 12/05/2023        NACIDO EL: 14 08 1991
	`
	res2 := parser.ParseDocumentText(raw2)
	if res2.Fields.FullName != "Juan Carlos Quispe Condori" {
		t.Fatalf("expected FullName='Juan Carlos Quispe Condori', got '%s'", res2.Fields.FullName)
	}
	if res2.Fields.BirthDate != "1991-08-14" {
		t.Fatalf("expected BirthDate='1991-08-14', got '%s'", res2.Fields.BirthDate)
	}
}
