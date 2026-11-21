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
