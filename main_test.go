package main

import (
	"bytes"
	"encoding/json"
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

	// Caso real: Las etiquetas tenues "NOMBRES" y "APELLIDOS" no fueron leídas por el OCR,
	// pero sí las líneas en negrita con el nombre y los apellidos del carnet boliviano.
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
	// 1. Carnet clásico boliviano donde la etiqueta "A:" se perdió pero el nombre está encima de FIRMA DEL TITULAR
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

	// 2. Reverso MRZ IDBOL degradado donde el OCR leyó un solo '<' entre apellidos y nombre ("ROJAS<MAMANI<MIJAEL<<<<<<")
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


